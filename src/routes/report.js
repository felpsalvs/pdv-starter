const express = require('express');
const db = require('../db');
const { todayLocal } = require('../util');
const { toCamelCase } = require('../services/serialize');

const router = express.Router();

router.get('/day', (req, res) => {
  const date = req.query.date || todayLocal();

  const byPaymentMethod = db
    .prepare(
      `SELECT payment_method, COUNT(*) AS count, COALESCE(SUM(total), 0) AS total
       FROM orders WHERE date = ? AND status = 'paid' GROUP BY payment_method`
    )
    .all(date);

  const topProducts = db
    .prepare(
      `SELECT oi.name, SUM(oi.quantity) AS quantity, SUM(oi.quantity * oi.unit_price) AS total
       FROM order_items oi
       JOIN orders o ON o.id = oi.order_id
       WHERE o.date = ? AND o.status = 'paid'
       GROUP BY oi.name
       ORDER BY quantity DESC
       LIMIT 10`
    )
    .all(date);

  const summary = db
    .prepare(
      `SELECT
         COUNT(*) FILTER (WHERE status = 'paid') AS paidOrders,
         COUNT(*) FILTER (WHERE status = 'canceled') AS canceledOrders,
         COUNT(*) FILTER (WHERE status = 'open') AS openTabs,
         COALESCE(SUM(total) FILTER (WHERE status = 'paid'), 0) AS totalSold
       FROM orders WHERE date = ?`
    )
    .get(date);

  res.json({
    date,
    summary,
    byPaymentMethod: byPaymentMethod.map(toCamelCase),
    topProducts: topProducts.map(toCamelCase),
  });
});

module.exports = router;
