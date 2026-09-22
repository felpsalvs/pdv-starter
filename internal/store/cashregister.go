package store

import "database/sql"

func (s *Store) scanCashRegister(row *sql.Row) (*CashRegister, error) {
	var r CashRegister
	err := row.Scan(
		&r.ID, &r.OpeningAmount, &r.OpenedAt, &r.ClosedAt, &r.CountedAmount,
		&r.TotalCash, &r.TotalPix, &r.TotalDebit, &r.TotalCredit,
		&r.TotalCashOut, &r.TotalCashIn, &r.ExpectedAmount, &r.Difference, &r.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

const cashRegisterColumns = `id, opening_amount, opened_at, closed_at, counted_amount,
	total_cash, total_pix, total_debit, total_credit,
	total_cash_out, total_cash_in, expected_amount, difference, status`

func (s *Store) OpenRegister() (*CashRegister, error) {
	row := s.DB.QueryRow(
		"SELECT " + cashRegisterColumns + " FROM cash_register WHERE status = 'open' ORDER BY id DESC LIMIT 1",
	)
	return s.scanCashRegister(row)
}

func (s *Store) GetCashRegister(id int64) (*CashRegister, error) {
	row := s.DB.QueryRow("SELECT "+cashRegisterColumns+" FROM cash_register WHERE id = ?", id)
	return s.scanCashRegister(row)
}

func (s *Store) CreateCashRegister(openingAmount float64, openedAt string) (int64, error) {
	result, err := s.DB.Exec("INSERT INTO cash_register (opening_amount, opened_at, status) VALUES (?, ?, 'open')", openingAmount, openedAt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

type PaymentTotals struct {
	Cash   float64 `json:"cash"`
	Pix    float64 `json:"pix"`
	Debit  float64 `json:"debit"`
	Credit float64 `json:"credit"`
}

func (s *Store) SalesByPaymentMethod(cashRegisterID int64) (PaymentTotals, error) {
	rows, err := s.DB.Query(
		`SELECT payment_method, COALESCE(SUM(total), 0) FROM orders
		 WHERE cash_register_id = ? AND status = 'paid' GROUP BY payment_method`,
		cashRegisterID,
	)
	if err != nil {
		return PaymentTotals{}, err
	}
	defer rows.Close()

	var totals PaymentTotals
	for rows.Next() {
		var method string
		var sum float64
		if err := rows.Scan(&method, &sum); err != nil {
			return PaymentTotals{}, err
		}
		switch method {
		case "cash":
			totals.Cash = sum
		case "pix":
			totals.Pix = sum
		case "debit":
			totals.Debit = sum
		case "credit":
			totals.Credit = sum
		}
	}
	return totals, rows.Err()
}

type MovementTotals struct {
	CashOut float64 `json:"cashOut"`
	CashIn  float64 `json:"cashIn"`
}

func (s *Store) MovementTotals(cashRegisterID int64) (MovementTotals, error) {
	rows, err := s.DB.Query(
		`SELECT type, COALESCE(SUM(amount), 0) FROM cash_register_movements
		 WHERE cash_register_id = ? GROUP BY type`,
		cashRegisterID,
	)
	if err != nil {
		return MovementTotals{}, err
	}
	defer rows.Close()

	var totals MovementTotals
	for rows.Next() {
		var t string
		var sum float64
		if err := rows.Scan(&t, &sum); err != nil {
			return MovementTotals{}, err
		}
		if t == "cash_out" {
			totals.CashOut = sum
		}
		if t == "cash_in" {
			totals.CashIn = sum
		}
	}
	return totals, rows.Err()
}

type OpenTabsSummary struct {
	Count int64   `json:"count"`
	Total float64 `json:"total"`
}

func (s *Store) OpenTabsSummary() (OpenTabsSummary, error) {
	var summary OpenTabsSummary
	err := s.DB.QueryRow("SELECT COUNT(*), COALESCE(SUM(total), 0) FROM orders WHERE status = 'open'").
		Scan(&summary.Count, &summary.Total)
	return summary, err
}

func (s *Store) InsertCashMovement(cashRegisterID int64, movementType string, amount float64, reason, createdAt string) error {
	_, err := s.DB.Exec(
		"INSERT INTO cash_register_movements (cash_register_id, type, amount, reason, created_at) VALUES (?, ?, ?, ?, ?)",
		cashRegisterID, movementType, amount, reason, createdAt,
	)
	return err
}

type CloseRegisterInput struct {
	ClosedAt       string
	CountedAmount  float64
	TotalCash      float64
	TotalPix       float64
	TotalDebit     float64
	TotalCredit    float64
	TotalCashOut   float64
	TotalCashIn    float64
	ExpectedAmount float64
	Difference     float64
}

func (s *Store) CloseRegister(id int64, in CloseRegisterInput) error {
	_, err := s.DB.Exec(
		`UPDATE cash_register
		 SET closed_at = ?, counted_amount = ?, total_cash = ?, total_pix = ?,
		     total_debit = ?, total_credit = ?, total_cash_out = ?, total_cash_in = ?,
		     expected_amount = ?, difference = ?, status = 'closed'
		 WHERE id = ?`,
		in.ClosedAt, in.CountedAmount, in.TotalCash, in.TotalPix,
		in.TotalDebit, in.TotalCredit, in.TotalCashOut, in.TotalCashIn,
		in.ExpectedAmount, in.Difference, id,
	)
	return err
}

func (s *Store) ListClosedRegisters() ([]CashRegister, error) {
	rows, err := s.DB.Query("SELECT " + cashRegisterColumns + " FROM cash_register WHERE status = 'closed' ORDER BY id DESC LIMIT 30")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	registers := []CashRegister{}
	for rows.Next() {
		var r CashRegister
		if err := rows.Scan(
			&r.ID, &r.OpeningAmount, &r.OpenedAt, &r.ClosedAt, &r.CountedAmount,
			&r.TotalCash, &r.TotalPix, &r.TotalDebit, &r.TotalCredit,
			&r.TotalCashOut, &r.TotalCashIn, &r.ExpectedAmount, &r.Difference, &r.Status,
		); err != nil {
			return nil, err
		}
		registers = append(registers, r)
	}
	return registers, rows.Err()
}
