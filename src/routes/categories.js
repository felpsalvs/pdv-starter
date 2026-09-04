const express = require('express');
const db = require('../db');
const { toCamelCase } = require('../services/serialize');

const router = express.Router();

router.get('/', (req, res) => {
  const categories = db.prepare('SELECT * FROM categories WHERE active = 1 ORDER BY sort_order, name').all();
  res.json(categories.map(toCamelCase));
});

router.post('/', (req, res) => {
  const { name } = req.body;
  if (!name || typeof name !== 'string' || !name.trim()) {
    return res.status(400).json({ error: 'Informe o nome da categoria.' });
  }

  const highestOrder = db.prepare('SELECT COALESCE(MAX(sort_order), -1) AS m FROM categories').get().m;
  const result = db
    .prepare('INSERT INTO categories (name, sort_order, active) VALUES (?, ?, 1)')
    .run(name.trim(), highestOrder + 1);

  res.status(201).json(toCamelCase(db.prepare('SELECT * FROM categories WHERE id = ?').get(result.lastInsertRowid)));
});

router.put('/:id', (req, res) => {
  const { id } = req.params;
  const { name, sortOrder } = req.body;

  const category = db.prepare('SELECT * FROM categories WHERE id = ?').get(id);
  if (!category) return res.status(404).json({ error: 'Categoria não encontrada.' });

  const newName = name !== undefined ? String(name).trim() : category.name;
  const newSortOrder = sortOrder !== undefined ? Number(sortOrder) : category.sort_order;

  if (!newName) return res.status(400).json({ error: 'Informe o nome da categoria.' });

  db.prepare('UPDATE categories SET name = ?, sort_order = ? WHERE id = ?').run(newName, newSortOrder, id);
  res.json(toCamelCase(db.prepare('SELECT * FROM categories WHERE id = ?').get(id)));
});

router.delete('/:id', (req, res) => {
  const { id } = req.params;
  const category = db.prepare('SELECT * FROM categories WHERE id = ?').get(id);
  if (!category) return res.status(404).json({ error: 'Categoria não encontrada.' });

  db.prepare('UPDATE categories SET active = 0 WHERE id = ?').run(id);
  res.status(204).end();
});

module.exports = router;
