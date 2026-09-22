package store

import "database/sql"

// Store wraps the shared *sql.DB with the query methods the services need.
// A single struct keeps every hand-written SQL statement in one package,
// mirroring how the Node backend's routes/services called db.prepare(...)
// directly but without scattering raw SQL across HTTP handlers.
type Store struct {
	DB *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{DB: db}
}

// ErrNotFound is returned by single-row lookups when nothing matches.
var ErrNotFound = sql.ErrNoRows
