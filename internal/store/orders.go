package store

import (
	"database/sql"
)

// Queryer is satisfied by both *sql.DB and *sql.Tx, so order creation can
// run inside a transaction (matching the Node backend's db.transaction())
// while every other call goes straight through the pooled connection.
type Queryer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

func (s *Store) Begin() (*sql.Tx, error) {
	return s.DB.Begin()
}

func (s *Store) NextDailyNumber(q Queryer, date string) (int64, error) {
	var highest int64
	err := q.QueryRow("SELECT COALESCE(MAX(daily_number), 0) FROM orders WHERE date = ?", date).Scan(&highest)
	return highest + 1, err
}

type NewOrder struct {
	DailyNumber    int64
	Date           string
	Source         string
	Reference      *string
	Status         string
	Total          float64
	PaymentMethod  *string
	AmountReceived *float64
	ChangeDue      *float64
	CashRegisterID *int64
	Note           *string
	CreatedAt      string
	PaidAt         *string
}

func (s *Store) InsertOrder(q Queryer, o NewOrder) (int64, error) {
	result, err := q.Exec(
		`INSERT INTO orders
			(daily_number, date, source, reference, status, total, payment_method,
			 amount_received, change_due, cash_register_id, note, created_at, paid_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.DailyNumber, o.Date, o.Source, o.Reference, o.Status, o.Total, o.PaymentMethod,
		o.AmountReceived, o.ChangeDue, o.CashRegisterID, o.Note, o.CreatedAt, o.PaidAt,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

type NewOrderItem struct {
	ProductID int64
	Name      string
	UnitPrice float64
	Quantity  int64
	Note      *string
}

func (s *Store) InsertOrderItem(q Queryer, orderID int64, item NewOrderItem) error {
	_, err := q.Exec(
		`INSERT INTO order_items (order_id, product_id, name, unit_price, quantity, note)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		orderID, item.ProductID, item.Name, item.UnitPrice, item.Quantity, item.Note,
	)
	return err
}

func (s *Store) LoadOrder(id int64) (*Order, error) {
	var o Order
	err := s.DB.QueryRow(
		`SELECT id, daily_number, date, source, reference, status, total, payment_method,
		        amount_received, change_due, cash_register_id, note, created_at, paid_at,
		        canceled_at, cancellation_reason, printed_at, print_attempts
		 FROM orders WHERE id = ?`, id,
	).Scan(
		&o.ID, &o.DailyNumber, &o.Date, &o.Source, &o.Reference, &o.Status, &o.Total, &o.PaymentMethod,
		&o.AmountReceived, &o.ChangeDue, &o.CashRegisterID, &o.Note, &o.CreatedAt, &o.PaidAt,
		&o.CanceledAt, &o.CancellationReason, &o.PrintedAt, &o.PrintAttempts,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	items, err := s.loadOrderItems(id)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return &o, nil
}

func (s *Store) loadOrderItems(orderID int64) ([]OrderItem, error) {
	rows, err := s.DB.Query(
		"SELECT id, order_id, product_id, name, unit_price, quantity, note FROM order_items WHERE order_id = ? ORDER BY id",
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []OrderItem{}
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.Name, &it.UnitPrice, &it.Quantity, &it.Note); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) ListOrdersForDay(date string, status string) ([]Order, error) {
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = s.DB.Query(
			`SELECT id, daily_number, date, source, reference, status, total, payment_method,
			        amount_received, change_due, cash_register_id, note, created_at, paid_at,
			        canceled_at, cancellation_reason, printed_at, print_attempts
			 FROM orders WHERE date = ? AND status = ? ORDER BY id DESC`, date, status,
		)
	} else {
		rows, err = s.DB.Query(
			`SELECT id, daily_number, date, source, reference, status, total, payment_method,
			        amount_received, change_due, cash_register_id, note, created_at, paid_at,
			        canceled_at, cancellation_reason, printed_at, print_attempts
			 FROM orders WHERE date = ? ORDER BY id DESC`, date,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID, &o.DailyNumber, &o.Date, &o.Source, &o.Reference, &o.Status, &o.Total, &o.PaymentMethod,
			&o.AmountReceived, &o.ChangeDue, &o.CashRegisterID, &o.Note, &o.CreatedAt, &o.PaidAt,
			&o.CanceledAt, &o.CancellationReason, &o.PrintedAt, &o.PrintAttempts,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range orders {
		items, err := s.loadOrderItems(orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}
	return orders, nil
}

func (s *Store) MarkPaid(orderID int64, paymentMethod string, amountReceived *float64, changeDue float64, cashRegisterID int64, paidAt string) error {
	_, err := s.DB.Exec(
		`UPDATE orders
		 SET status = 'paid', payment_method = ?, amount_received = ?, change_due = ?,
		     cash_register_id = ?, paid_at = ?
		 WHERE id = ?`,
		paymentMethod, amountReceived, changeDue, cashRegisterID, paidAt, orderID,
	)
	return err
}

func (s *Store) MarkCanceled(orderID int64, reason, canceledAt string) error {
	_, err := s.DB.Exec(
		"UPDATE orders SET status = 'canceled', canceled_at = ?, cancellation_reason = ? WHERE id = ?",
		canceledAt, reason, orderID,
	)
	return err
}

func (s *Store) MarkPrinted(orderID int64, success bool, printedAt string) error {
	if success {
		_, err := s.DB.Exec(
			"UPDATE orders SET printed_at = ?, print_attempts = print_attempts + 1 WHERE id = ?",
			printedAt, orderID,
		)
		return err
	}
	_, err := s.DB.Exec("UPDATE orders SET print_attempts = print_attempts + 1 WHERE id = ?", orderID)
	return err
}
