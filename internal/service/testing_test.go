package service

import (
	"path/filepath"
	"testing"

	"github.com/felpsalvs/pdv-starter/internal/db"
	"github.com/felpsalvs/pdv-starter/internal/printer"
	"github.com/felpsalvs/pdv-starter/internal/store"
)

// newTestServices opens a fresh temp database (migrated, empty) and wires
// up the same services main.go does, plus a printer pointed at a config
// file that never exists — so every print attempt returns the expected
// "not configured" failure, matching a fresh install with no printer set
// up yet.
func newTestServices(t *testing.T) (*store.Store, *OrdersService, *CashRegisterService) {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "pdv.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	st := store.New(sqlDB)
	prn := printer.New(filepath.Join(t.TempDir(), "printer.config.json"))
	return st, NewOrdersService(st, prn), NewCashRegisterService(st)
}

func seedProduct(t *testing.T, st *store.Store, name string, price float64) int64 {
	t.Helper()
	p, err := st.CreateProduct(name, price, nil, 0)
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	return p.ID
}
