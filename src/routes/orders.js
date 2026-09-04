const express = require('express');
const ordersService = require('../services/orders');
const { ValidationError } = require('../services/errors');

const router = express.Router();

function handleError(res, err) {
  if (err instanceof ValidationError) {
    return res.status(400).json({ error: err.message });
  }
  console.error(err);
  return res.status(500).json({ error: 'Erro interno do servidor.' });
}

router.post('/', async (req, res) => {
  try {
    const { source, reference, items, note, payment } = req.body;
    const result = await ordersService.createOrderWithPrinting({ source, reference, items, note, payment });
    res.status(201).json(result);
  } catch (err) {
    handleError(res, err);
  }
});

router.get('/today', (req, res) => {
  const { date, status } = req.query;
  res.json(ordersService.listForDay(date, status));
});

router.get('/:id', (req, res) => {
  const order = ordersService.loadOrder(req.params.id);
  if (!order) return res.status(404).json({ error: 'Pedido não encontrado.' });
  res.json(order);
});

router.post('/:id/pay', async (req, res) => {
  try {
    const result = await ordersService.registerPayment(req.params.id, req.body);
    res.json(result);
  } catch (err) {
    handleError(res, err);
  }
});

router.post('/:id/cancel', (req, res) => {
  try {
    const order = ordersService.cancelOrder(req.params.id, req.body.reason);
    res.json(order);
  } catch (err) {
    handleError(res, err);
  }
});

router.post('/:id/reprint', async (req, res) => {
  try {
    const { document } = req.body;
    const result = await ordersService.reprint(req.params.id, document);
    res.json(result);
  } catch (err) {
    handleError(res, err);
  }
});

module.exports = router;
