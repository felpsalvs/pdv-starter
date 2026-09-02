const path = require('path');
const Database = require('better-sqlite3');

const dbPath = path.join(__dirname, '..', 'data', 'pdv.db');
const db = new Database(dbPath);

db.pragma('journal_mode = WAL');

db.exec(`
  CREATE TABLE IF NOT EXISTS sopas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL,
    preco REAL NOT NULL,
    ativo INTEGER NOT NULL DEFAULT 1
  );

  CREATE TABLE IF NOT EXISTS pedidos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    itens_json TEXT NOT NULL,
    total REAL NOT NULL,
    observacao TEXT,
    criado_em TEXT NOT NULL
  );

  CREATE TABLE IF NOT EXISTS caixa (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    valor_abertura REAL NOT NULL,
    aberto_em TEXT NOT NULL,
    valor_fechamento REAL,
    fechado_em TEXT,
    total_vendas REAL,
    status TEXT NOT NULL DEFAULT 'aberto'
  );
`);

module.exports = db;
