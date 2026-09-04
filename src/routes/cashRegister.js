const express = require('express');
const cashRegisterService = require('../services/cashRegister');
const { ValidationError } = require('../services/errors');

const router = express.Router();

function handleError(res, err) {
  if (err instanceof ValidationError) {
    return res.status(400).json({ error: err.message });
  }
  console.error(err);
  return res.status(500).json({ error: 'Erro interno do servidor.' });
}

router.get('/current', (req, res) => {
  res.json(cashRegisterService.currentSummary());
});

router.post('/open', (req, res) => {
  try {
    const register = cashRegisterService.open(req.body.openingAmount);
    res.status(201).json(register);
  } catch (err) {
    handleError(res, err);
  }
});

router.post('/movement', (req, res) => {
  try {
    const { type, amount, reason } = req.body;
    const summary = cashRegisterService.registerMovement(type, amount, reason);
    res.status(201).json(summary);
  } catch (err) {
    handleError(res, err);
  }
});

router.post('/close', (req, res) => {
  try {
    const result = cashRegisterService.close(req.body.countedAmount);
    res.json(result);
  } catch (err) {
    handleError(res, err);
  }
});

router.get('/closures', (req, res) => {
  res.json(cashRegisterService.listClosures());
});

module.exports = router;
