package db

import (
	"database/sql"
	"net/url"
	"path/filepath"
	"testing"
)

// openRaw opens a database without running migrations, so tests can seed a
// legacy schema before Migrate() runs.
func openRaw(t *testing.T, path string) (*sql.DB, error) {
	t.Helper()
	dsn := "file:" + url.PathEscape(path) + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	return sql.Open("sqlite", dsn)
}

func TestOpenFreshDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pdv.db")
	sqlDB, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer sqlDB.Close()

	for _, table := range []string{"categories", "products", "orders", "order_items", "cash_register", "cash_register_movements"} {
		var name string
		if err := sqlDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name); err != nil {
			t.Errorf("table %s missing: %v", table, err)
		}
	}

	var version int
	if err := sqlDB.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != len(migrations) {
		t.Errorf("user_version = %d, want %d", version, len(migrations))
	}

	// Re-opening must be a no-op (no re-run of migration1, no duplicate tables).
	sqlDB.Close()
	sqlDB2, err := Open(path)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer sqlDB2.Close()
}

func TestMigrateLegacyData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	sqlDB, err := openRaw(t, path)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	seed := `
		CREATE TABLE sopas (id INTEGER PRIMARY KEY, nome TEXT, preco REAL, ativo INTEGER);
		INSERT INTO sopas (nome, preco, ativo) VALUES ('Caldo verde', 18.5, 1), ('Feijoada', 22.0, 1);

		CREATE TABLE pedidos (id INTEGER PRIMARY KEY, total REAL, observacao TEXT, criado_em TEXT, itens_json TEXT);
		INSERT INTO pedidos (total, observacao, criado_em, itens_json) VALUES
			(18.5, NULL, '2024-01-05 12:00:00', '[{"nome":"Caldo verde","preco":18.5,"quantidade":1}]'),
			(44.0, 'sem cebola', '2024-01-05 12:30:00', '[{"nome":"Feijoada","preco":22.0,"quantidade":2}]');

		CREATE TABLE caixa (id INTEGER PRIMARY KEY, valor_abertura REAL, aberto_em TEXT, fechado_em TEXT, valor_fechamento REAL, total_vendas REAL, status TEXT);
		INSERT INTO caixa (valor_abertura, aberto_em, fechado_em, valor_fechamento, total_vendas, status) VALUES
			(100.0, '2024-01-05 08:00:00', '2024-01-05 18:00:00', 162.5, 62.5, 'fechado');
	`
	if _, err := sqlDB.Exec(seed); err != nil {
		t.Fatal(err)
	}

	if err := Migrate(sqlDB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var productCount int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM products").Scan(&productCount); err != nil {
		t.Fatal(err)
	}
	if productCount != 2 {
		t.Errorf("products = %d, want 2", productCount)
	}

	var orderCount, itemCount int
	sqlDB.QueryRow("SELECT COUNT(*) FROM orders").Scan(&orderCount)
	sqlDB.QueryRow("SELECT COUNT(*) FROM order_items").Scan(&itemCount)
	if orderCount != 2 {
		t.Errorf("orders = %d, want 2", orderCount)
	}
	if itemCount != 2 {
		t.Errorf("order_items = %d, want 2", itemCount)
	}

	var status string
	var expected, difference float64
	if err := sqlDB.QueryRow("SELECT status, expected_amount, difference FROM cash_register").Scan(&status, &expected, &difference); err != nil {
		t.Fatal(err)
	}
	if status != "closed" {
		t.Errorf("status = %q, want closed", status)
	}
	if expected != 162.5 {
		t.Errorf("expected_amount = %v, want 162.5", expected)
	}
	if difference != 0 {
		t.Errorf("difference = %v, want 0", difference)
	}
}
