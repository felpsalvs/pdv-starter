const fs = require('fs');
const path = require('path');
const { ThermalPrinter, PrinterTypes } = require('node-thermal-printer');

const CONFIG_PATH = path.join(__dirname, '..', 'printer.config.json');

const PAYMENT_METHOD_LABELS = {
  cash: 'Dinheiro',
  pix: 'Pix',
  debit: 'Cartão débito',
  credit: 'Cartão crédito',
};

function readConfig() {
  try {
    const raw = fs.readFileSync(CONFIG_PATH, 'utf8');
    const config = JSON.parse(raw);
    if (!config.printerName) return null;
    return config;
  } catch (err) {
    return null;
  }
}

async function openPrinter() {
  const config = readConfig();
  if (!config) {
    return {
      error:
        'Impressora não configurada. Copie printer.config.example.json para printer.config.json e informe o nome da impressora.',
    };
  }

  const printer = new ThermalPrinter({
    type: PrinterTypes.EPSON,
    interface: `printer:${config.printerName}`,
    width: 32,
  });

  const connected = await printer.isPrinterConnected().catch(() => false);
  if (!connected) {
    return { error: 'Impressora não respondeu (desligada ou não encontrada).' };
  }

  return { printer };
}

function referenceLine(order) {
  if (order.source === 'table') return `Mesa ${order.reference || '?'}`;
  return order.reference ? order.reference : 'Balcão';
}

function buildKitchenTicket(printer, order) {
  printer.alignCenter();
  printer.setTextSize(1, 1);
  printer.bold(true);
  printer.println(`SENHA ${order.dailyNumber}`);
  printer.setTextNormal();
  printer.println('PEDIDO PARA A COZINHA');
  printer.bold(false);
  printer.drawLine();

  printer.alignLeft();
  const timestamp = new Date(order.createdAt.replace(' ', 'T')).toLocaleString('pt-BR');
  printer.println(`${referenceLine(order)} · ${timestamp}`);
  printer.newLine();

  for (const item of order.items) {
    printer.bold(true);
    printer.println(`${item.quantity}x ${item.name}`);
    printer.bold(false);
    if (item.note) {
      printer.println(`   Obs: ${item.note}`);
    }
  }

  if (order.note) {
    printer.newLine();
    printer.println(`Obs geral: ${order.note}`);
  }

  printer.drawLine();
  printer.cut();
}

function buildLabel(printer, order, item) {
  printer.alignCenter();
  printer.bold(true);
  printer.setTextSize(1, 1);
  printer.println(`${order.dailyNumber}`);
  printer.setTextNormal();
  printer.println(referenceLine(order));
  printer.bold(false);
  printer.drawLine();
  printer.bold(true);
  printer.println(item.name);
  printer.bold(false);
  if (item.note) {
    printer.println(item.note);
  }
  printer.cut();
}

function buildReceipt(printer, order) {
  printer.alignCenter();
  printer.bold(true);
  printer.println('RECIBO');
  printer.bold(false);
  printer.drawLine();

  printer.alignLeft();
  printer.println(`Pedido #${order.dailyNumber} · ${referenceLine(order)}`);
  printer.newLine();

  for (const item of order.items) {
    printer.println(`${item.quantity}x ${item.name}`);
    printer.println(`   ${(item.unitPrice * item.quantity).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}`);
  }

  printer.drawLine();
  printer.bold(true);
  printer.println(`Total: ${order.total.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}`);
  printer.bold(false);
  printer.println(`Pagamento: ${PAYMENT_METHOD_LABELS[order.paymentMethod] || order.paymentMethod}`);
  if (order.paymentMethod === 'cash' && order.amountReceived != null) {
    printer.println(`Recebido: ${order.amountReceived.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}`);
    printer.println(`Troco: ${(order.changeDue || 0).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}`);
  }

  printer.drawLine();
  printer.cut();
}

const BUILDERS = {
  kitchen: (printer, order) => buildKitchenTicket(printer, order),
  label: (printer, order) => {
    for (const item of order.items) {
      for (let unit = 0; unit < item.quantity; unit += 1) {
        buildLabel(printer, order, item);
      }
    }
  },
  receipt: (printer, order) => buildReceipt(printer, order),
};

// Prints a list of documents reusing a single printer connection (a paid
// order prints up to 3 documents — kitchen ticket, label, receipt —
// reconnecting for each would triple checkout latency and the chance of a
// false-negative connectivity check).
async function printBatch(jobs) {
  const { printer, error } = await openPrinter();
  if (error) {
    return jobs.map(() => ({ success: false, reason: error }));
  }

  const results = [];
  for (const { type, order } of jobs) {
    const build = BUILDERS[type];
    if (!build) {
      results.push({ success: false, reason: `Unknown document type: ${type}` });
      continue;
    }
    try {
      printer.clear();
      build(printer, order);
      await printer.execute();
      results.push({ success: true });
    } catch (err) {
      results.push({ success: false, reason: `Falha ao imprimir: ${err.message}` });
    }
  }
  return results;
}

async function printDocument(type, order) {
  const [result] = await printBatch([{ type, order }]);
  return result;
}

async function printNewOrder(order) {
  const jobs = [
    { type: 'kitchen', order },
    { type: 'label', order },
  ];
  if (order.status === 'paid') jobs.push({ type: 'receipt', order });

  const [kitchen, label, receipt] = await printBatch(jobs);
  return {
    success: kitchen.success && label.success && (!receipt || receipt.success),
    kitchen,
    label,
    receipt: receipt || null,
  };
}

module.exports = { printDocument, printNewOrder };
