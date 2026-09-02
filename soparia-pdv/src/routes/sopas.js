const express = require('express');
const db = require('../db');

const router = express.Router();

router.get('/', (req, res) => {
  const sopas = db.prepare('SELECT * FROM sopas WHERE ativo = 1 ORDER BY nome').all();
  res.json(sopas);
});

router.post('/', (req, res) => {
  const { nome, preco } = req.body;

  if (!nome || typeof nome !== 'string' || !nome.trim()) {
    return res.status(400).json({ erro: 'Informe o nome da sopa.' });
  }
  const precoNum = Number(preco);
  if (!Number.isFinite(precoNum) || precoNum <= 0) {
    return res.status(400).json({ erro: 'Informe um preço válido.' });
  }

  const result = db
    .prepare('INSERT INTO sopas (nome, preco, ativo) VALUES (?, ?, 1)')
    .run(nome.trim(), precoNum);

  const sopa = db.prepare('SELECT * FROM sopas WHERE id = ?').get(result.lastInsertRowid);
  res.status(201).json(sopa);
});

router.put('/:id', (req, res) => {
  const { id } = req.params;
  const { nome, preco } = req.body;

  const sopa = db.prepare('SELECT * FROM sopas WHERE id = ?').get(id);
  if (!sopa) return res.status(404).json({ erro: 'Sopa não encontrada.' });

  const novoNome = nome !== undefined ? String(nome).trim() : sopa.nome;
  const novoPreco = preco !== undefined ? Number(preco) : sopa.preco;

  if (!novoNome) return res.status(400).json({ erro: 'Informe o nome da sopa.' });
  if (!Number.isFinite(novoPreco) || novoPreco <= 0) {
    return res.status(400).json({ erro: 'Informe um preço válido.' });
  }

  db.prepare('UPDATE sopas SET nome = ?, preco = ? WHERE id = ?').run(novoNome, novoPreco, id);
  res.json(db.prepare('SELECT * FROM sopas WHERE id = ?').get(id));
});

router.delete('/:id', (req, res) => {
  const { id } = req.params;
  const sopa = db.prepare('SELECT * FROM sopas WHERE id = ?').get(id);
  if (!sopa) return res.status(404).json({ erro: 'Sopa não encontrada.' });

  db.prepare('UPDATE sopas SET ativo = 0 WHERE id = ?').run(id);
  res.status(204).end();
});

module.exports = router;
