const fs = require('fs');
const path = require('path');
const Database = require('better-sqlite3');

// PDV_DB_PATH permite apontar pra um banco separado (ex: durante testes),
// sem nunca tocar no arquivo real de produção em data/pdv.db.
const dbPath = process.env.PDV_DB_PATH || path.join(__dirname, '..', 'data', 'pdv.db');
fs.mkdirSync(path.dirname(dbPath), { recursive: true });
const db = new Database(dbPath);

db.pragma('journal_mode = WAL');
db.pragma('foreign_keys = ON');

function tableExists(name) {
  return !!db.prepare("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?").get(name);
}

// Migration 1: full schema (categories, products, orders with status/payment,
// normalized order_items, cash_register with payment-method breakdown,
// cash_register_movements). If a database from the original MVP (sopas /
// pedidos / caixa, single-table, no payment method) already exists, its data
// is preserved: the old tables are renamed to "_legacy_*" and copied into the
// new schema. Nothing is dropped.
function migration1() {
  const isLegacy = tableExists('sopas');

  if (isLegacy) {
    db.exec('ALTER TABLE sopas RENAME TO _legacy_sopas');
    if (tableExists('pedidos')) db.exec('ALTER TABLE pedidos RENAME TO _legacy_pedidos');
    if (tableExists('caixa')) db.exec('ALTER TABLE caixa RENAME TO _legacy_caixa');
  }

  db.exec(`
    CREATE TABLE categories (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      name TEXT NOT NULL,
      sort_order INTEGER NOT NULL DEFAULT 0,
      active INTEGER NOT NULL DEFAULT 1
    );

    CREATE TABLE products (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      name TEXT NOT NULL,
      price REAL NOT NULL,
      category_id INTEGER REFERENCES categories(id),
      active INTEGER NOT NULL DEFAULT 1,
      available_today INTEGER NOT NULL DEFAULT 1,
      sort_order INTEGER NOT NULL DEFAULT 0
    );

    CREATE TABLE cash_register (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      opening_amount REAL NOT NULL,
      opened_at TEXT NOT NULL,
      closed_at TEXT,
      counted_amount REAL,
      total_cash REAL,
      total_pix REAL,
      total_debit REAL,
      total_credit REAL,
      total_cash_out REAL,
      total_cash_in REAL,
      expected_amount REAL,
      difference REAL,
      status TEXT NOT NULL DEFAULT 'open'
    );

    CREATE TABLE cash_register_movements (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      cash_register_id INTEGER NOT NULL REFERENCES cash_register(id),
      type TEXT NOT NULL,
      amount REAL NOT NULL,
      reason TEXT,
      created_at TEXT NOT NULL
    );

    CREATE TABLE orders (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      daily_number INTEGER NOT NULL,
      date TEXT NOT NULL,
      source TEXT NOT NULL DEFAULT 'counter',
      reference TEXT,
      status TEXT NOT NULL DEFAULT 'open',
      total REAL NOT NULL,
      payment_method TEXT,
      amount_received REAL,
      change_due REAL,
      cash_register_id INTEGER REFERENCES cash_register(id),
      note TEXT,
      created_at TEXT NOT NULL,
      paid_at TEXT,
      canceled_at TEXT,
      cancellation_reason TEXT,
      printed_at TEXT,
      print_attempts INTEGER NOT NULL DEFAULT 0
    );

    CREATE TABLE order_items (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      order_id INTEGER NOT NULL REFERENCES orders(id),
      product_id INTEGER,
      name TEXT NOT NULL,
      unit_price REAL NOT NULL,
      quantity INTEGER NOT NULL,
      note TEXT
    );

    CREATE INDEX idx_orders_date ON orders(date);
    CREATE INDEX idx_orders_status ON orders(status);
    CREATE INDEX idx_order_items_order ON order_items(order_id);
    CREATE INDEX idx_cash_register_movements ON cash_register_movements(cash_register_id);
  `);

  if (isLegacy) {
    const legacyProducts = db.prepare('SELECT * FROM _legacy_sopas').all();
    const insertProduct = db.prepare(
      'INSERT INTO products (name, price, category_id, active, available_today, sort_order) VALUES (?, ?, NULL, ?, 1, ?)'
    );
    legacyProducts.forEach((product, index) => {
      insertProduct.run(product.nome, product.preco, product.ativo, index);
    });

    if (tableExists('_legacy_pedidos')) {
      const legacyOrders = db.prepare('SELECT * FROM _legacy_pedidos ORDER BY id').all();
      const countByDate = {};
      const insertOrder = db.prepare(`
        INSERT INTO orders
          (daily_number, date, source, reference, status, total, payment_method,
           cash_register_id, note, created_at, paid_at)
        VALUES (?, ?, 'counter', NULL, 'paid', ?, NULL, NULL, ?, ?, ?)
      `);
      const insertItem = db.prepare(`
        INSERT INTO order_items (order_id, product_id, name, unit_price, quantity, note)
        VALUES (?, NULL, ?, ?, ?, NULL)
      `);

      for (const order of legacyOrders) {
        const date = String(order.criado_em).slice(0, 10);
        countByDate[date] = (countByDate[date] || 0) + 1;

        const result = insertOrder.run(
          countByDate[date],
          date,
          order.total,
          order.observacao,
          order.criado_em,
          order.criado_em
        );

        let items = [];
        try {
          items = JSON.parse(order.itens_json);
        } catch (err) {
          items = [];
        }
        for (const item of items) {
          insertItem.run(result.lastInsertRowid, item.nome, item.preco, item.quantidade);
        }
      }
    }

    if (tableExists('_legacy_caixa')) {
      // The legacy schema didn't track payment method — every sale was
      // treated as cash in the drawer (the same assumption the original
      // closing-balance bug made). Folding total_vendas into total_cash
      // preserves the number exactly as the old system computed it; there's
      // no data to retroactively split out pix/card sales.
      const legacyRegisters = db.prepare('SELECT * FROM _legacy_caixa ORDER BY id').all();
      const insertRegister = db.prepare(`
        INSERT INTO cash_register
          (opening_amount, opened_at, closed_at, counted_amount, total_cash,
           total_cash_out, total_cash_in, expected_amount, difference, status)
        VALUES (?, ?, ?, ?, ?, 0, 0, ?, ?, ?)
      `);
      for (const register of legacyRegisters) {
        const expected =
          register.valor_fechamento !== null && register.valor_fechamento !== undefined
            ? register.valor_abertura + (register.total_vendas || 0)
            : null;
        const difference = register.valor_fechamento !== null ? register.valor_fechamento - expected : null;
        insertRegister.run(
          register.valor_abertura,
          register.aberto_em,
          register.fechado_em,
          register.valor_fechamento,
          register.total_vendas,
          expected,
          difference,
          register.status === 'aberto' ? 'open' : 'closed'
        );
      }
    }
  }
}

const MIGRATIONS = [migration1];

function migrate() {
  const currentVersion = db.pragma('user_version', { simple: true });
  for (let version = currentVersion; version < MIGRATIONS.length; version += 1) {
    const run = db.transaction(() => {
      MIGRATIONS[version]();
      db.pragma(`user_version = ${version + 1}`);
    });
    run();
  }
}

migrate();

module.exports = db;
