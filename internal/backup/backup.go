// Package backup periodically snapshots the SQLite database, replacing
// src/backup.js's better-sqlite3 .backup() calls with SQLite's own
// `VACUUM INTO` — safe to run against a live WAL-mode database without
// pausing writers, same as the native backup API was.
package backup

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	interval       = 4 * time.Hour
	retentionCount = 30
)

type Backupper struct {
	db  *sql.DB
	dir string
}

func New(db *sql.DB, dir string) *Backupper {
	return &Backupper{db: db, dir: dir}
}

func (b *Backupper) fileName() string {
	now := time.Now()
	return fmt.Sprintf("pdv-%s-%s.db", now.Format("2006-01-02"), now.Format("150405"))
}

func (b *Backupper) pruneOld() {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "pdv-") && strings.HasSuffix(name, ".db") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	excess := len(names) - retentionCount
	if excess <= 0 {
		return
	}
	for _, name := range names[:excess] {
		// If an old backup can't be deleted, that's not a reason to stop anything.
		_ = os.Remove(filepath.Join(b.dir, name))
	}
}

// Create runs one backup synchronously: VACUUM INTO the destination file,
// then prune anything past the retention window.
func (b *Backupper) Create() error {
	if err := os.MkdirAll(b.dir, 0o755); err != nil {
		return fmt.Errorf("não foi possível fazer o backup automático do banco: %w", err)
	}
	destination := filepath.Join(b.dir, b.fileName())

	if _, err := b.db.Exec("VACUUM INTO ?", destination); err != nil {
		return fmt.Errorf("não foi possível fazer o backup automático do banco: %w", err)
	}
	b.pruneOld()
	log.Printf("Backup do banco salvo em %s", destination)
	return nil
}

// Schedule runs an immediate backup and then one every `interval`, until
// ctx is canceled.
func (b *Backupper) Schedule(ctx context.Context) {
	if err := b.Create(); err != nil {
		log.Println(err)
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := b.Create(); err != nil {
					log.Println(err)
				}
			}
		}
	}()
}
