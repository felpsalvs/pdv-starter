const express = require('express');
const db = require('../db');
const { agoraLocal } = require('../util');

const router = express.Router();

function caixaAberto() {
  return db.prepare("SELECT * FROM caixa WHERE status = 'aberto' ORDER BY id DESC LIMIT 1").get();
}

function totalVendasDesde(dataHora) {
  const row = db
    .prepare('SELECT COALESCE(SUM(total), 0) AS soma FROM pedidos WHERE criado_em >= ?')
    .get(dataHora);
  return row.soma;
}

router.get('/atual', (req, res) => {
  const caixa = caixaAberto();
  if (!caixa) return res.json(null);

  const totalVendasParcial = totalVendasDesde(caixa.aberto_em);
  res.json({ ...caixa, total_vendas_parcial: totalVendasParcial });
});

router.post('/abrir', (req, res) => {
  if (caixaAberto()) {
    return res.status(409).json({ erro: 'Já existe um caixa aberto.' });
  }

  const valorAbertura = Number(req.body.valor_abertura);
  if (!Number.isFinite(valorAbertura) || valorAbertura < 0) {
    return res.status(400).json({ erro: 'Informe um valor de abertura válido.' });
  }

  const abertoEm = agoraLocal();
  const result = db
    .prepare(
      "INSERT INTO caixa (valor_abertura, aberto_em, status) VALUES (?, ?, 'aberto')"
    )
    .run(valorAbertura, abertoEm);

  res.status(201).json(db.prepare('SELECT * FROM caixa WHERE id = ?').get(result.lastInsertRowid));
});

router.post('/fechar', (req, res) => {
  const caixa = caixaAberto();
  if (!caixa) {
    return res.status(400).json({ erro: 'Não há caixa aberto para fechar.' });
  }

  const valorContado = Number(req.body.valor_contado);
  if (!Number.isFinite(valorContado) || valorContado < 0) {
    return res.status(400).json({ erro: 'Informe um valor contado válido.' });
  }

  const totalVendas = totalVendasDesde(caixa.aberto_em);
  const valorEsperado = caixa.valor_abertura + totalVendas;
  const diferenca = valorContado - valorEsperado;
  const fechadoEm = agoraLocal();

  db.prepare(
    `UPDATE caixa
     SET valor_fechamento = ?, fechado_em = ?, total_vendas = ?, status = 'fechado'
     WHERE id = ?`
  ).run(valorContado, fechadoEm, totalVendas, caixa.id);

  res.json({
    caixa: db.prepare('SELECT * FROM caixa WHERE id = ?').get(caixa.id),
    valor_esperado: valorEsperado,
    valor_contado: valorContado,
    diferenca,
  });
});

module.exports = router;
