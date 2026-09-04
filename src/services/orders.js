const db = require('../db');
const { printDocument, printNewOrder } = require('../printer');
const { nowLocal, todayLocal } = require('../util');
const { openRegister } = require('./cashRegister');
const { ValidationError } = require('./errors');
const { toCamelCase } = require('./serialize');

const SOURCES = new Set(['counter', 'table']);
const PAYMENT_METHODS = new Set(['cash', 'pix', 'debit', 'credit']);

function nextDailyNumber(date) {
  const row = db.prepare('SELECT COALESCE(MAX(daily_number), 0) AS highest FROM orders WHERE date = ?').get(date);
  return row.highest + 1;
}

function findActiveProduct(productId) {
  return db.prepare('SELECT * FROM products WHERE id = ? AND active = 1').get(productId);
}

function loadOrder(id) {
  const order = db.prepare('SELECT * FROM orders WHERE id = ?').get(id);
  if (!order) return null;
  const items = db.prepare('SELECT * FROM order_items WHERE order_id = ? ORDER BY id').all(id);
  return { ...toCamelCase(order), items: items.map(toCamelCase) };
}

function validateItems(items) {
  if (!Array.isArray(items) || items.length === 0) {
    throw new ValidationError('O pedido precisa ter pelo menos um item.');
  }

  const orderItems = [];
  for (const item of items) {
    const quantity = Number(item.quantity);
    if (!item.productId || !Number.isInteger(quantity) || quantity <= 0) {
      throw new ValidationError('Item de pedido inválido.');
    }
    const product = findActiveProduct(item.productId);
    if (!product) {
      throw new ValidationError(`Produto ${item.productId} não encontrado ou inativo.`);
    }
    orderItems.push({
      productId: product.id,
      name: product.name,
      unitPrice: product.price,
      quantity,
      note: item.note ? String(item.note).trim() || null : null,
    });
  }
  return orderItems;
}

function createOrder({ source, reference, items, note, payment }) {
  const finalSource = SOURCES.has(source) ? source : 'counter';
  const orderItems = validateItems(items);
  const total = orderItems.reduce((sum, item) => sum + item.unitPrice * item.quantity, 0);

  let paymentData = null;
  if (payment) {
    paymentData = validatePayment(payment, total);
  }

  const date = todayLocal();
  const createdAt = nowLocal();

  const run = db.transaction(() => {
    const dailyNumber = nextDailyNumber(date);

    const result = db
      .prepare(
        `INSERT INTO orders
          (daily_number, date, source, reference, status, total, payment_method,
           amount_received, change_due, cash_register_id, note, created_at, paid_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .run(
        dailyNumber,
        date,
        finalSource,
        reference ? String(reference).trim() || null : null,
        paymentData ? 'paid' : 'open',
        total,
        paymentData ? paymentData.paymentMethod : null,
        paymentData ? paymentData.amountReceived : null,
        paymentData ? paymentData.changeDue : null,
        paymentData ? paymentData.cashRegisterId : null,
        note ? String(note).trim() || null : null,
        createdAt,
        paymentData ? createdAt : null
      );

    const insertItem = db.prepare(
      `INSERT INTO order_items (order_id, product_id, name, unit_price, quantity, note)
       VALUES (?, ?, ?, ?, ?, ?)`
    );
    for (const item of orderItems) {
      insertItem.run(result.lastInsertRowid, item.productId, item.name, item.unitPrice, item.quantity, item.note);
    }

    return result.lastInsertRowid;
  });

  return run();
}

function validatePayment(payment, total) {
  const paymentMethod = payment.paymentMethod;
  if (!PAYMENT_METHODS.has(paymentMethod)) {
    throw new ValidationError('Forma de pagamento inválida.');
  }

  const register = openRegister();
  if (!register) {
    throw new ValidationError('Abra o caixa antes de registrar um pagamento.');
  }

  let amountReceived = null;
  let changeDue = 0;
  if (paymentMethod === 'cash') {
    amountReceived = Number(payment.amountReceived);
    if (!Number.isFinite(amountReceived) || amountReceived < total) {
      throw new ValidationError('Valor recebido insuficiente.');
    }
    changeDue = Math.round((amountReceived - total) * 100) / 100;
  }

  return { paymentMethod, amountReceived, changeDue, cashRegisterId: register.id };
}

async function createOrderWithPrinting(data) {
  const orderId = createOrder(data);
  const order = loadOrder(orderId);
  const printing = await printNewOrder(order);
  markPrinted(orderId, printing.success);

  return { order: loadOrder(orderId), printing, receipt: printing.receipt };
}

function markPrinted(orderId, success) {
  if (success) {
    db.prepare(
      'UPDATE orders SET printed_at = ?, print_attempts = print_attempts + 1 WHERE id = ?'
    ).run(nowLocal(), orderId);
  } else {
    db.prepare('UPDATE orders SET print_attempts = print_attempts + 1 WHERE id = ?').run(orderId);
  }
}

async function registerPayment(orderId, payment) {
  const order = loadOrder(orderId);
  if (!order) throw new ValidationError('Pedido não encontrado.');
  if (order.status !== 'open') {
    throw new ValidationError('Este pedido não está aguardando pagamento.');
  }

  const paymentData = validatePayment(payment, order.total);
  const paidAt = nowLocal();

  db.prepare(
    `UPDATE orders
     SET status = 'paid', payment_method = ?, amount_received = ?, change_due = ?,
         cash_register_id = ?, paid_at = ?
     WHERE id = ?`
  ).run(
    paymentData.paymentMethod,
    paymentData.amountReceived,
    paymentData.changeDue,
    paymentData.cashRegisterId,
    paidAt,
    orderId
  );

  const updatedOrder = loadOrder(orderId);
  const receipt = await printDocument('receipt', updatedOrder);
  return { order: updatedOrder, receipt };
}

function cancelOrder(orderId, reason) {
  const order = loadOrder(orderId);
  if (!order) throw new ValidationError('Pedido não encontrado.');
  if (order.status === 'canceled') {
    throw new ValidationError('Este pedido já está cancelado.');
  }
  if (!reason || !String(reason).trim()) {
    throw new ValidationError('Informe o motivo do cancelamento.');
  }

  db.prepare(
    "UPDATE orders SET status = 'canceled', canceled_at = ?, cancellation_reason = ? WHERE id = ?"
  ).run(nowLocal(), String(reason).trim(), orderId);

  return loadOrder(orderId);
}

async function reprint(orderId, documentType) {
  const order = loadOrder(orderId);
  if (!order) throw new ValidationError('Pedido não encontrado.');
  const result = await printDocument(documentType, order);
  if (result.success) {
    markPrinted(orderId, true);
  }
  return result;
}

function listForDay(date, status) {
  const dateFilter = date || todayLocal();
  const orders = status
    ? db.prepare('SELECT * FROM orders WHERE date = ? AND status = ? ORDER BY id DESC').all(dateFilter, status)
    : db.prepare('SELECT * FROM orders WHERE date = ? ORDER BY id DESC').all(dateFilter);

  const findItems = db.prepare('SELECT * FROM order_items WHERE order_id = ? ORDER BY id');
  return orders.map((order) => ({ ...toCamelCase(order), items: findItems.all(order.id).map(toCamelCase) }));
}

module.exports = {
  createOrderWithPrinting,
  registerPayment,
  cancelOrder,
  reprint,
  listForDay,
  loadOrder,
};
