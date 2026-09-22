package store

import (
	"database/sql"
	"strings"
)

type ProductFilter struct {
	CategoryID      string // empty = no filter
	AvailableToday  bool
	IncludeInactive bool
}

func (s *Store) ListProducts(f ProductFilter) ([]Product, error) {
	conditions := []string{}
	args := []interface{}{}

	if !f.IncludeInactive {
		conditions = append(conditions, "active = 1")
	}
	if f.CategoryID != "" {
		conditions = append(conditions, "category_id = ?")
		args = append(args, f.CategoryID)
	}
	if f.AvailableToday {
		conditions = append(conditions, "available_today = 1")
	}

	query := "SELECT id, name, price, category_id, active, available_today, sort_order FROM products"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY sort_order, name"

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.CategoryID, &p.Active, &p.AvailableToday, &p.SortOrder); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (s *Store) GetProduct(id int64) (*Product, error) {
	var p Product
	err := s.DB.QueryRow(
		"SELECT id, name, price, category_id, active, available_today, sort_order FROM products WHERE id = ?", id,
	).Scan(&p.ID, &p.Name, &p.Price, &p.CategoryID, &p.Active, &p.AvailableToday, &p.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// FindActiveProduct is the lookup order validation uses: only active
// products can be added to a new order.
func (s *Store) FindActiveProduct(id int64) (*Product, error) {
	p, err := s.GetProduct(id)
	if err != nil || p == nil || p.Active != 1 {
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	return p, nil
}

func (s *Store) HighestProductSortOrder() (int64, error) {
	var highest int64
	err := s.DB.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM products").Scan(&highest)
	return highest, err
}

func (s *Store) CreateProduct(name string, price float64, categoryID *int64, sortOrder int64) (*Product, error) {
	result, err := s.DB.Exec(
		"INSERT INTO products (name, price, category_id, active, available_today, sort_order) VALUES (?, ?, ?, 1, 1, ?)",
		name, price, categoryID, sortOrder,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetProduct(id)
}

func (s *Store) UpdateProduct(id int64, name string, price float64, categoryID *int64, active int64) (*Product, error) {
	_, err := s.DB.Exec(
		"UPDATE products SET name = ?, price = ?, category_id = ?, active = ? WHERE id = ?",
		name, price, categoryID, active, id,
	)
	if err != nil {
		return nil, err
	}
	return s.GetProduct(id)
}

func (s *Store) SetProductAvailability(id int64, availableToday int64) (*Product, error) {
	if _, err := s.DB.Exec("UPDATE products SET available_today = ? WHERE id = ?", availableToday, id); err != nil {
		return nil, err
	}
	return s.GetProduct(id)
}

func (s *Store) DeactivateProduct(id int64) error {
	_, err := s.DB.Exec("UPDATE products SET active = 0 WHERE id = ?", id)
	return err
}
