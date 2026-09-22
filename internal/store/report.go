package store

type PaymentMethodTotal struct {
	PaymentMethod *string `json:"paymentMethod"`
	Count         int64   `json:"count"`
	Total         float64 `json:"total"`
}

type TopProduct struct {
	Name     string  `json:"name"`
	Quantity int64   `json:"quantity"`
	Total    float64 `json:"total"`
}

type DaySummary struct {
	PaidOrders     int64   `json:"paidOrders"`
	CanceledOrders int64   `json:"canceledOrders"`
	OpenTabs       int64   `json:"openTabs"`
	TotalSold      float64 `json:"totalSold"`
}

func (s *Store) ByPaymentMethod(date string) ([]PaymentMethodTotal, error) {
	rows, err := s.DB.Query(
		`SELECT payment_method, COUNT(*) AS count, COALESCE(SUM(total), 0) AS total
		 FROM orders WHERE date = ? AND status = 'paid' GROUP BY payment_method`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []PaymentMethodTotal{}
	for rows.Next() {
		var r PaymentMethodTotal
		if err := rows.Scan(&r.PaymentMethod, &r.Count, &r.Total); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) TopProducts(date string) ([]TopProduct, error) {
	rows, err := s.DB.Query(
		`SELECT oi.name, SUM(oi.quantity) AS quantity, SUM(oi.quantity * oi.unit_price) AS total
		 FROM order_items oi
		 JOIN orders o ON o.id = oi.order_id
		 WHERE o.date = ? AND o.status = 'paid'
		 GROUP BY oi.name
		 ORDER BY quantity DESC
		 LIMIT 10`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []TopProduct{}
	for rows.Next() {
		var r TopProduct
		if err := rows.Scan(&r.Name, &r.Quantity, &r.Total); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) DaySummary(date string) (DaySummary, error) {
	var summary DaySummary
	err := s.DB.QueryRow(
		`SELECT
		   COUNT(*) FILTER (WHERE status = 'paid') AS paidOrders,
		   COUNT(*) FILTER (WHERE status = 'canceled') AS canceledOrders,
		   COUNT(*) FILTER (WHERE status = 'open') AS openTabs,
		   COALESCE(SUM(total) FILTER (WHERE status = 'paid'), 0) AS totalSold
		 FROM orders WHERE date = ?`,
		date,
	).Scan(&summary.PaidOrders, &summary.CanceledOrders, &summary.OpenTabs, &summary.TotalSold)
	return summary, err
}
