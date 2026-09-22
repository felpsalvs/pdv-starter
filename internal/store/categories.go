package store

import "database/sql"

func (s *Store) ListActiveCategories() ([]Category, error) {
	rows, err := s.DB.Query("SELECT id, name, sort_order, active FROM categories WHERE active = 1 ORDER BY sort_order, name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.SortOrder, &c.Active); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (s *Store) GetCategory(id int64) (*Category, error) {
	var c Category
	err := s.DB.QueryRow("SELECT id, name, sort_order, active FROM categories WHERE id = ?", id).
		Scan(&c.ID, &c.Name, &c.SortOrder, &c.Active)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) HighestCategorySortOrder() (int64, error) {
	var highest int64
	err := s.DB.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM categories").Scan(&highest)
	return highest, err
}

func (s *Store) CreateCategory(name string, sortOrder int64) (*Category, error) {
	result, err := s.DB.Exec("INSERT INTO categories (name, sort_order, active) VALUES (?, ?, 1)", name, sortOrder)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetCategory(id)
}

func (s *Store) UpdateCategory(id int64, name string, sortOrder int64) (*Category, error) {
	if _, err := s.DB.Exec("UPDATE categories SET name = ?, sort_order = ? WHERE id = ?", name, sortOrder, id); err != nil {
		return nil, err
	}
	return s.GetCategory(id)
}

// DeactivateCategory deactivates the category and, in the same transaction,
// clears category_id on every product that pointed to it — otherwise those
// products would keep referencing an inactive category and quietly vanish
// from the order screen (which only loads active categories/tabs) while
// still showing a blank category in the edit form.
func (s *Store) DeactivateCategory(id int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE categories SET active = 0 WHERE id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE products SET category_id = NULL WHERE category_id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}
