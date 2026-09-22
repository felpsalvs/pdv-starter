const fs = require('fs');
const os = require('os');
const path = require('path');
const crypto = require('crypto');
const { execFile } = require('child_process');
const { ThermalPrinter, PrinterTypes, CharacterSet } = require('node-thermal-printer');

const CONFIG_PATH = path.join(__dirname, '..', 'printer.config.json');
const NOT_CONFIGURED_ERROR =
  'Impressora não configurada. Copie printer.config.example.json para printer.config.json e informe o nome da impressora.';

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

// Envia os bytes ESC/POS pro sistema operacional imprimir, sem depender de
// nenhum pacote nativo (nada de compilar C++ no computador do cliente):
// no macOS/Linux via CUPS (`lp -o raw`), no Windows via uma impressora
// compartilhada localmente (`copy /b`).
function printRawBytes(printerName, buffer) {
  return new Promise((resolve, reject) => {
    const tempFile = path.join(os.tmpdir(), `pdv-print-${crypto.randomBytes(6).toString('hex')}.bin`);
    fs.writeFile(tempFile, buffer, (writeErr) => {
      if (writeErr) return reject(writeErr);

      const cleanup = () => fs.unlink(tempFile, () => {});
      const isWindows = process.platform === 'win32';
      const command = isWindows ? 'cmd' : 'lp';
      const args = isWindows
        ? ['/c', 'copy', '/b', tempFile, `\\\\localhost\\${printerName}`]
        : ['-d', printerName, '-o', 'raw', tempFile];

      execFile(command, args, (err, stdout, stderr) => {
        cleanup();
        if (err) {
          reject(new Error((stderr && stderr.toString().trim()) || err.message));
        } else {
          resolve();
        }
      });
    });
  });
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

async function printBatch(jobs) {
  const config = readConfig();
  if (!config) {
    return jobs.map(() => ({ success: false, reason: NOT_CONFIGURED_ERROR }));
  }

  const results = [];
  for (const { type, order } of jobs) {
    const build = BUILDERS[type];
    if (!build) {
      results.push({ success: false, reason: `Unknown document type: ${type}` });
      continue;
    }
    try {
      const printer = new ThermalPrinter({
        type: PrinterTypes.EPSON,
        width: 32,
        characterSet: CharacterSet.PC860_PORTUGUESE,
      });
      build(printer, order);
      await printRawBytes(config.printerName, printer.getBuffer());
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
