// Package db opens the SQLite database and runs migrations. It uses
// modernc.org/sqlite (pure Go, no CGO) so the binary cross-compiles for
// Windows from any host without a C toolchain.
package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open creates the parent directory if needed, opens the database with the
// same pragmas the Node backend used (WAL journal mode, foreign keys on),
// and runs any pending migrations.
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	dsn := "file:" + url.PathEscape(path) +
		"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	// better-sqlite3 is synchronous and single-connection by nature; matching
	// that here avoids SQLITE_BUSY races and keeps daily_number allocation
	// (SELECT MAX + INSERT) safe without extra locking.
	sqlDB.SetMaxOpenConns(1)

	if err := Migrate(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrating db: %w", err)
	}

	return sqlDB, nil
}
