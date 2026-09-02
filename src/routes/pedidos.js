const express = require('express');
const db = require('../db');
const { imprimirPedido } = require('../printer');
const { agoraLocal, hojeLocal } = require('../util');

const router = express.Router();

router.post('/', async (req, res) => {
  const { itens, observacao } = req.body;

  if (!Array.isArray(itens) || itens.length === 0) {
    return res.status(400).json({ erro: 'O pedido precisa ter pelo menos um item.' });
  }

  const itensPedido = [];
  for (const item of itens) {
    const quantidade = Number(item.quantidade);
    if (!item.sopaId || !Number.isInteger(quantidade) || quantidade <= 0) {
      return res.status(400).json({ erro: 'Item de pedido inválido.' });
    }
    const sopa = db.prepare('SELECT * FROM sopas WHERE id = ? AND ativo = 1').get(item.sopaId);
    if (!sopa) {
      return res.status(400).json({ erro: `Sopa ${item.sopaId} não encontrada ou inativa.` });
    }
    itensPedido.push({ sopaId: sopa.id, nome: sopa.nome, preco: sopa.preco, quantidade });
  }

  const total = itensPedido.reduce((soma, item) => soma + item.preco * item.quantidade, 0);
  const criadoEm = agoraLocal();

  const result = db
    .prepare('INSERT INTO pedidos (itens_json, total, observacao, criado_em) VALUES (?, ?, ?, ?)')
    .run(JSON.stringify(itensPedido), total, observacao || null, criadoEm);

  const pedido = {
    id: result.lastInsertRowid,
    itens: itensPedido,
    total,
    observacao: observacao || null,
    criado_em: criadoEm,
  };

  const impressao = await imprimirPedido(pedido);

  res.status(201).json({ pedido, impressao });
});

router.get('/hoje', (req, res) => {
  const hoje = hojeLocal();
  const pedidos = db
    .prepare("SELECT * FROM pedidos WHERE criado_em LIKE ? ORDER BY id DESC")
    .all(`${hoje}%`)
    .map((p) => ({ ...p, itens: JSON.parse(p.itens_json) }));

  res.json(pedidos);
});

module.exports = router;
