const db = require('../db');
const { nowLocal } = require('../util');
const { ValidationError } = require('./errors');
const { toCamelCase } = require('./serialize');

function openRegister() {
  return db.prepare("SELECT * FROM cash_register WHERE status = 'open' ORDER BY id DESC LIMIT 1").get();
}

function open(openingAmount) {
  if (openRegister()) {
    throw new ValidationError('A cash register is already open.');
  }

  const amount = Number(openingAmount);
  if (!Number.isFinite(amount) || amount < 0) {
    throw new ValidationError('Enter a valid opening amount.');
  }

  const result = db
    .prepare("INSERT INTO cash_register (opening_amount, opened_at, status) VALUES (?, ?, 'open')")
    .run(amount, nowLocal());

  return toCamelCase(db.prepare('SELECT * FROM cash_register WHERE id = ?').get(result.lastInsertRowid));
}

function salesByPaymentMethod(cashRegisterId) {
  const rows = db
    .prepare(
      `SELECT payment_method, COALESCE(SUM(total), 0) AS sum
       FROM orders
       WHERE cash_register_id = ? AND status = 'paid'
       GROUP BY payment_method`
    )
    .all(cashRegisterId);

  const totals = { cash: 0, pix: 0, debit: 0, credit: 0 };
  for (const row of rows) {
    if (totals[row.payment_method] !== undefined) {
      totals[row.payment_method] = row.sum;
    }
  }
  return totals;
}

function movementTotals(cashRegisterId) {
  const rows = db
    .prepare(
      `SELECT type, COALESCE(SUM(amount), 0) AS sum
       FROM cash_register_movements
       WHERE cash_register_id = ?
       GROUP BY type`
    )
    .all(cashRegisterId);

  const totals = { cashOut: 0, cashIn: 0 };
  for (const row of rows) {
    if (row.type === 'cash_out') totals.cashOut = row.sum;
    if (row.type === 'cash_in') totals.cashIn = row.sum;
  }
  return totals;
}

function openTabsSummary() {
  const row = db
    .prepare("SELECT COUNT(*) AS count, COALESCE(SUM(total), 0) AS total FROM orders WHERE status = 'open'")
    .get();
  return { count: row.count, total: row.total };
}

function currentSummary() {
  const register = openRegister();
  if (!register) return null;

  const sales = salesByPaymentMethod(register.id);
  const movements = movementTotals(register.id);
  const totalSales = sales.cash + sales.pix + sales.debit + sales.credit;
  const expectedInDrawer = register.opening_amount + sales.cash + movements.cashIn - movements.cashOut;

  return {
    ...toCamelCase(register),
    sales,
    movements,
    totalSales,
    expectedInDrawer,
    openTabs: openTabsSummary(),
  };
}

function registerMovement(type, amount, reason) {
  if (type !== 'cash_out' && type !== 'cash_in') {
    throw new ValidationError('Invalid movement type.');
  }
  const register = openRegister();
  if (!register) {
    throw new ValidationError('There is no open cash register.');
  }
  const value = Number(amount);
  if (!Number.isFinite(value) || value <= 0) {
    throw new ValidationError('Enter a valid amount.');
  }
  if (!reason || !String(reason).trim()) {
    throw new ValidationError('Enter a reason for the movement.');
  }

  db.prepare(
    'INSERT INTO cash_register_movements (cash_register_id, type, amount, reason, created_at) VALUES (?, ?, ?, ?, ?)'
  ).run(register.id, type, value, String(reason).trim(), nowLocal());

  return currentSummary();
}

function close(countedAmount) {
  const register = openRegister();
  if (!register) {
    throw new ValidationError('There is no open cash register to close.');
  }

  const counted = Number(countedAmount);
  if (!Number.isFinite(counted) || counted < 0) {
    throw new ValidationError('Enter a valid counted amount.');
  }

  const summary = currentSummary();
  const difference = Math.round((counted - summary.expectedInDrawer) * 100) / 100;

  db.prepare(
    `UPDATE cash_register
     SET closed_at = ?, counted_amount = ?, total_cash = ?, total_pix = ?,
         total_debit = ?, total_credit = ?, total_cash_out = ?, total_cash_in = ?,
         expected_amount = ?, difference = ?, status = 'closed'
     WHERE id = ?`
  ).run(
    nowLocal(),
    counted,
    summary.sales.cash,
    summary.sales.pix,
    summary.sales.debit,
    summary.sales.credit,
    summary.movements.cashOut,
    summary.movements.cashIn,
    summary.expectedInDrawer,
    difference,
    register.id
  );

  return {
    register: toCamelCase(db.prepare('SELECT * FROM cash_register WHERE id = ?').get(register.id)),
    summary,
    countedAmount: counted,
    difference,
  };
}

function listClosures() {
  return db
    .prepare("SELECT * FROM cash_register WHERE status = 'closed' ORDER BY id DESC LIMIT 30")
    .all()
    .map(toCamelCase);
}

module.exports = {
  openRegister,
  open,
  currentSummary,
  registerMovement,
  close,
  listClosures,
};
