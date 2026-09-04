const express = require('express');
const db = require('../db');
const { toCamelCase } = require('../services/serialize');

const router = express.Router();

router.get('/', (req, res) => {
  const { category, availableToday, includeInactive } = req.query;

  const conditions = [];
  const params = [];

  if (!includeInactive) {
    conditions.push('active = 1');
  }
  if (category) {
    conditions.push('category_id = ?');
    params.push(category);
  }
  if (availableToday === '1') {
    conditions.push('available_today = 1');
  }

  const where = conditions.length ? `WHERE ${conditions.join(' AND ')}` : '';
  const products = db.prepare(`SELECT * FROM products ${where} ORDER BY sort_order, name`).all(...params);
  res.json(products.map(toCamelCase));
});

router.post('/', (req, res) => {
  const { name, price, categoryId } = req.body;

  if (!name || typeof name !== 'string' || !name.trim()) {
    return res.status(400).json({ error: 'Informe o nome do produto.' });
  }
  const priceNumber = Number(price);
  if (!Number.isFinite(priceNumber) || priceNumber <= 0) {
    return res.status(400).json({ error: 'Informe um preço válido.' });
  }

  const highestOrder = db.prepare('SELECT COALESCE(MAX(sort_order), -1) AS m FROM products').get().m;

  const result = db
    .prepare(
      'INSERT INTO products (name, price, category_id, active, available_today, sort_order) VALUES (?, ?, ?, 1, 1, ?)'
    )
    .run(name.trim(), priceNumber, categoryId || null, highestOrder + 1);

  const product = db.prepare('SELECT * FROM products WHERE id = ?').get(result.lastInsertRowid);
  res.status(201).json(toCamelCase(product));
});

router.put('/:id', (req, res) => {
  const { id } = req.params;
  const { name, price, categoryId, active } = req.body;

  const product = db.prepare('SELECT * FROM products WHERE id = ?').get(id);
  if (!product) return res.status(404).json({ error: 'Produto não encontrado.' });

  const newName = name !== undefined ? String(name).trim() : product.name;
  const newPrice = price !== undefined ? Number(price) : product.price;
  const newCategory = categoryId !== undefined ? categoryId : product.category_id;
  const newActive = active !== undefined ? (active ? 1 : 0) : product.active;

  if (!newName) return res.status(400).json({ error: 'Informe o nome do produto.' });
  if (!Number.isFinite(newPrice) || newPrice <= 0) {
    return res.status(400).json({ error: 'Informe um preço válido.' });
  }

  db.prepare('UPDATE products SET name = ?, price = ?, category_id = ?, active = ? WHERE id = ?').run(
    newName,
    newPrice,
    newCategory || null,
    newActive,
    id
  );
  res.json(toCamelCase(db.prepare('SELECT * FROM products WHERE id = ?').get(id)));
});

router.patch('/:id/availability', (req, res) => {
  const { id } = req.params;
  const product = db.prepare('SELECT * FROM products WHERE id = ?').get(id);
  if (!product) return res.status(404).json({ error: 'Produto não encontrado.' });

  const availableToday = req.body.availableToday ? 1 : 0;
  db.prepare('UPDATE products SET available_today = ? WHERE id = ?').run(availableToday, id);
  res.json(toCamelCase(db.prepare('SELECT * FROM products WHERE id = ?').get(id)));
});

router.delete('/:id', (req, res) => {
  const { id } = req.params;
  const product = db.prepare('SELECT * FROM products WHERE id = ?').get(id);
  if (!product) return res.status(404).json({ error: 'Produto não encontrado.' });

  db.prepare('UPDATE products SET active = 0 WHERE id = ?').run(id);
  res.status(204).end();
});

module.exports = router;
