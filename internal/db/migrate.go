package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// migration is a single schema step, applied inside a transaction with
// PRAGMA user_version bumped right after. Ported 1:1 from src/db.js's
// MIGRATIONS array — add new steps by appending to the migrations slice
// below, never by editing an already-shipped one.
type migration func(tx *sql.Tx) error

var migrations = []migration{migration1}

// Migrate brings the database up to the latest schema version, tracked with
// SQLite's own PRAGMA user_version (exactly like the Node backend did).
func Migrate(sqlDB *sql.DB) error {
	var current int
	if err := sqlDB.QueryRow("PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("reading user_version: %w", err)
	}

	for version := current; version < len(migrations); version++ {
		tx, err := sqlDB.Begin()
		if err != nil {
			return err
		}
		if err := migrations[version](tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", version+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func tableExists(tx *sql.Tx, name string) (bool, error) {
	var found string
	err := tx.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// migration1 is the full schema (categories, products, orders with
// status/payment, normalized order_items, cash_register with
// payment-method breakdown, cash_register_movements). If a database from
// the original MVP (sopas / pedidos / caixa, single-table, no payment
// method) already exists, its data is preserved: the old tables are
// renamed to "_legacy_*" and copied into the new schema. Nothing is
// dropped.
func migration1(tx *sql.Tx) error {
	isLegacy, err := tableExists(tx, "sopas")
	if err != nil {
		return err
	}

	if isLegacy {
		if _, err := tx.Exec("ALTER TABLE sopas RENAME TO _legacy_sopas"); err != nil {
			return err
		}
		if ok, err := tableExists(tx, "pedidos"); err != nil {
			return err
		} else if ok {
			if _, err := tx.Exec("ALTER TABLE pedidos RENAME TO _legacy_pedidos"); err != nil {
				return err
			}
		}
		if ok, err := tableExists(tx, "caixa"); err != nil {
			return err
		} else if ok {
			if _, err := tx.Exec("ALTER TABLE caixa RENAME TO _legacy_caixa"); err != nil {
				return err
			}
		}
	}

	schema := `
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
	`
	if _, err := tx.Exec(schema); err != nil {
		return err
	}

	if !isLegacy {
		return nil
	}
	return importLegacyData(tx)
}

func importLegacyData(tx *sql.Tx) error {
	type legacyProduct struct {
		Nome  string
		Preco float64
		Ativo int
	}
	rows, err := tx.Query("SELECT nome, preco, ativo FROM _legacy_sopas")
	if err != nil {
		return err
	}
	var legacyProducts []legacyProduct
	for rows.Next() {
		var p legacyProduct
		if err := rows.Scan(&p.Nome, &p.Preco, &p.Ativo); err != nil {
			rows.Close()
			return err
		}
		legacyProducts = append(legacyProducts, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	insertProduct, err := tx.Prepare(
		"INSERT INTO products (name, price, category_id, active, available_today, sort_order) VALUES (?, ?, NULL, ?, 1, ?)",
	)
	if err != nil {
		return err
	}
	defer insertProduct.Close()
	for i, p := range legacyProducts {
		if _, err := insertProduct.Exec(p.Nome, p.Preco, p.Ativo, i); err != nil {
			return err
		}
	}

	if ok, err := tableExists(tx, "_legacy_pedidos"); err != nil {
		return err
	} else if ok {
		if err := importLegacyOrders(tx); err != nil {
			return err
		}
	}

	if ok, err := tableExists(tx, "_legacy_caixa"); err != nil {
		return err
	} else if ok {
		if err := importLegacyCashRegisters(tx); err != nil {
			return err
		}
	}

	return nil
}

func importLegacyOrders(tx *sql.Tx) error {
	type legacyOrder struct {
		ID         int64
		Total      float64
		Observacao sql.NullString
		CriadoEm   string
		ItensJSON  sql.NullString
	}
	rows, err := tx.Query("SELECT id, total, observacao, criado_em, itens_json FROM _legacy_pedidos ORDER BY id")
	if err != nil {
		return err
	}
	var orders []legacyOrder
	for rows.Next() {
		var o legacyOrder
		if err := rows.Scan(&o.ID, &o.Total, &o.Observacao, &o.CriadoEm, &o.ItensJSON); err != nil {
			rows.Close()
			return err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	insertOrder, err := tx.Prepare(`
		INSERT INTO orders
			(daily_number, date, source, reference, status, total, payment_method,
			 cash_register_id, note, created_at, paid_at)
		VALUES (?, ?, 'counter', NULL, 'paid', ?, NULL, NULL, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer insertOrder.Close()

	insertItem, err := tx.Prepare(`
		INSERT INTO order_items (order_id, product_id, name, unit_price, quantity, note)
		VALUES (?, NULL, ?, ?, ?, NULL)
	`)
	if err != nil {
		return err
	}
	defer insertItem.Close()

	countByDate := map[string]int{}
	for _, o := range orders {
		date := o.CriadoEm
		if len(date) >= 10 {
			date = date[:10]
		}
		countByDate[date]++

		var note interface{}
		if o.Observacao.Valid {
			note = o.Observacao.String
		}

		result, err := insertOrder.Exec(countByDate[date], date, o.Total, note, o.CriadoEm, o.CriadoEm)
		if err != nil {
			return err
		}
		orderID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		var items []struct {
			Nome       string  `json:"nome"`
			Preco      float64 `json:"preco"`
			Quantidade int     `json:"quantidade"`
		}
		if o.ItensJSON.Valid {
			_ = json.Unmarshal([]byte(o.ItensJSON.String), &items)
		}
		for _, item := range items {
			if _, err := insertItem.Exec(orderID, item.Nome, item.Preco, item.Quantidade); err != nil {
				return err
			}
		}
	}
	return nil
}

func importLegacyCashRegisters(tx *sql.Tx) error {
	type legacyRegister struct {
		ValorAbertura   float64
		AbertoEm        string
		FechadoEm       sql.NullString
		ValorFechamento sql.NullFloat64
		TotalVendas     sql.NullFloat64
		Status          string
	}
	rows, err := tx.Query(
		"SELECT valor_abertura, aberto_em, fechado_em, valor_fechamento, total_vendas, status FROM _legacy_caixa ORDER BY id",
	)
	if err != nil {
		return err
	}
	var registers []legacyRegister
	for rows.Next() {
		var r legacyRegister
		if err := rows.Scan(&r.ValorAbertura, &r.AbertoEm, &r.FechadoEm, &r.ValorFechamento, &r.TotalVendas, &r.Status); err != nil {
			rows.Close()
			return err
		}
		registers = append(registers, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	// The legacy schema didn't track payment method — every sale was
	// treated as cash in the drawer (the same assumption the original
	// closing-balance bug made). Folding total_vendas into total_cash
	// preserves the number exactly as the old system computed it; there's
	// no data to retroactively split out pix/card sales.
	insertRegister, err := tx.Prepare(`
		INSERT INTO cash_register
			(opening_amount, opened_at, closed_at, counted_amount, total_cash,
			 total_cash_out, total_cash_in, expected_amount, difference, status)
		VALUES (?, ?, ?, ?, ?, 0, 0, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer insertRegister.Close()

	for _, r := range registers {
		status := "closed"
		if r.Status == "aberto" {
			status = "open"
		}

		var closedAt, expected, difference, totalVendas, countedAmount interface{}
		if r.FechadoEm.Valid {
			closedAt = r.FechadoEm.String
		}
		totalVendasVal := 0.0
		if r.TotalVendas.Valid {
			totalVendasVal = r.TotalVendas.Float64
			totalVendas = totalVendasVal
		} else {
			totalVendas = nil
		}
		if r.ValorFechamento.Valid {
			countedAmount = r.ValorFechamento.Float64
			expectedVal := r.ValorAbertura + totalVendasVal
			expected = expectedVal
			difference = r.ValorFechamento.Float64 - expectedVal
		}

		if _, err := insertRegister.Exec(
			r.ValorAbertura, r.AbertoEm, closedAt, countedAmount, totalVendas, expected, difference, status,
		); err != nil {
			return err
		}
	}
	return nil
}
