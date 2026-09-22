package service

import (
	"github.com/felpsalvs/pdv-starter/internal/clock"
	"github.com/felpsalvs/pdv-starter/internal/store"
)

type CashRegisterService struct {
	store *store.Store
}

func NewCashRegisterService(s *store.Store) *CashRegisterService {
	return &CashRegisterService{store: s}
}

// CurrentSummary is the JSON shape returned by GET /current and embedded in
// every mutation response — it flattens the cash_register row's own fields
// (via the embedded struct) next to the computed sales/movements totals,
// exactly like the Node backend's `{ ...toCamelCase(register), sales, ... }`.
type CurrentSummary struct {
	store.CashRegister
	Sales            store.PaymentTotals   `json:"sales"`
	Movements        store.MovementTotals  `json:"movements"`
	TotalSales       float64               `json:"totalSales"`
	ExpectedInDrawer float64               `json:"expectedInDrawer"`
	OpenTabs         store.OpenTabsSummary `json:"openTabs"`
}

func (s *CashRegisterService) OpenRegister() (*store.CashRegister, error) {
	return s.store.OpenRegister()
}

func (s *CashRegisterService) CurrentSummary() (*CurrentSummary, error) {
	register, err := s.store.OpenRegister()
	if err != nil {
		return nil, err
	}
	if register == nil {
		return nil, nil
	}
	return s.buildSummary(register)
}

func (s *CashRegisterService) buildSummary(register *store.CashRegister) (*CurrentSummary, error) {
	sales, err := s.store.SalesByPaymentMethod(register.ID)
	if err != nil {
		return nil, err
	}
	movements, err := s.store.MovementTotals(register.ID)
	if err != nil {
		return nil, err
	}
	openTabs, err := s.store.OpenTabsSummary()
	if err != nil {
		return nil, err
	}

	totalSales := sales.Cash + sales.Pix + sales.Debit + sales.Credit
	expectedInDrawer := register.OpeningAmount + sales.Cash + movements.CashIn - movements.CashOut

	return &CurrentSummary{
		CashRegister:     *register,
		Sales:            sales,
		Movements:        movements,
		TotalSales:       totalSales,
		ExpectedInDrawer: expectedInDrawer,
		OpenTabs:         openTabs,
	}, nil
}

func (s *CashRegisterService) Open(openingAmount float64, hasAmount bool) (*store.CashRegister, error) {
	existing, err := s.store.OpenRegister()
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, NewValidationError("A cash register is already open.")
	}

	if !hasAmount || openingAmount < 0 {
		return nil, NewValidationError("Enter a valid opening amount.")
	}

	id, err := s.store.CreateCashRegister(openingAmount, clock.NowLocal())
	if err != nil {
		return nil, err
	}
	return s.store.GetCashRegister(id)
}

func (s *CashRegisterService) RegisterMovement(movementType string, amount float64, hasAmount bool, reason string) (*CurrentSummary, error) {
	if movementType != "cash_out" && movementType != "cash_in" {
		return nil, NewValidationError("Invalid movement type.")
	}
	register, err := s.store.OpenRegister()
	if err != nil {
		return nil, err
	}
	if register == nil {
		return nil, NewValidationError("There is no open cash register.")
	}
	if !hasAmount || amount <= 0 {
		return nil, NewValidationError("Enter a valid amount.")
	}
	if reason == "" {
		return nil, NewValidationError("Enter a reason for the movement.")
	}

	if err := s.store.InsertCashMovement(register.ID, movementType, amount, reason, clock.NowLocal()); err != nil {
		return nil, err
	}
	return s.CurrentSummary()
}

type CloseResult struct {
	Register      store.CashRegister `json:"register"`
	Summary       CurrentSummary     `json:"summary"`
	CountedAmount float64            `json:"countedAmount"`
	Difference    float64            `json:"difference"`
}

func (s *CashRegisterService) Close(countedAmount float64, hasAmount bool) (*CloseResult, error) {
	register, err := s.store.OpenRegister()
	if err != nil {
		return nil, err
	}
	if register == nil {
		return nil, NewValidationError("There is no open cash register to close.")
	}
	if !hasAmount || countedAmount < 0 {
		return nil, NewValidationError("Enter a valid counted amount.")
	}

	summary, err := s.buildSummary(register)
	if err != nil {
		return nil, err
	}
	difference := round2(countedAmount - summary.ExpectedInDrawer)

	err = s.store.CloseRegister(register.ID, store.CloseRegisterInput{
		ClosedAt:       clock.NowLocal(),
		CountedAmount:  countedAmount,
		TotalCash:      summary.Sales.Cash,
		TotalPix:       summary.Sales.Pix,
		TotalDebit:     summary.Sales.Debit,
		TotalCredit:    summary.Sales.Credit,
		TotalCashOut:   summary.Movements.CashOut,
		TotalCashIn:    summary.Movements.CashIn,
		ExpectedAmount: summary.ExpectedInDrawer,
		Difference:     difference,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.store.GetCashRegister(register.ID)
	if err != nil {
		return nil, err
	}

	return &CloseResult{
		Register:      *updated,
		Summary:       *summary,
		CountedAmount: countedAmount,
		Difference:    difference,
	}, nil
}

func (s *CashRegisterService) ListClosures() ([]store.CashRegister, error) {
	return s.store.ListClosedRegisters()
}
