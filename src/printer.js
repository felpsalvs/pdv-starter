const fs = require('fs');
const path = require('path');
const { ThermalPrinter, PrinterTypes } = require('node-thermal-printer');

const CONFIG_PATH = path.join(__dirname, '..', 'printer.config.json');
const CONFIG_EXAMPLE_PATH = path.join(__dirname, '..', 'printer.config.example.json');

function lerConfig() {
  try {
    const raw = fs.readFileSync(CONFIG_PATH, 'utf8');
    const config = JSON.parse(raw);
    if (!config.nomeImpressora) return null;
    return config;
  } catch (err) {
    return null;
  }
}

function montarTicket(printer, pedido) {
  printer.alignCenter();
  printer.bold(true);
  printer.println('PEDIDO PARA A COZINHA');
  printer.bold(false);
  printer.drawLine();

  printer.alignLeft();
  const dataHora = new Date(pedido.criado_em).toLocaleString('pt-BR');
  printer.println(`Pedido #${pedido.id} - ${dataHora}`);
  printer.newLine();

  for (const item of pedido.itens) {
    printer.println(`${item.quantidade}x ${item.nome}`);
  }

  if (pedido.observacao) {
    printer.newLine();
    printer.println(`Obs: ${pedido.observacao}`);
  }

  printer.drawLine();
  printer.bold(true);
  printer.println(`Total: R$ ${pedido.total.toFixed(2)}`);
  printer.bold(false);

  printer.cut();
}

async function imprimirPedido(pedido) {
  const config = lerConfig();

  if (!config) {
    return {
      sucesso: false,
      motivo:
        'Impressora não configurada. Copie printer.config.example.json para printer.config.json e informe o nome da impressora.',
    };
  }

  try {
    const printer = new ThermalPrinter({
      type: PrinterTypes.EPSON,
      interface: `printer:${config.nomeImpressora}`,
      width: 32,
    });

    const conectada = await printer.isPrinterConnected().catch(() => false);
    if (!conectada) {
      return { sucesso: false, motivo: 'Impressora não respondeu (desligada ou não encontrada).' };
    }

    montarTicket(printer, pedido);
    await printer.execute();

    return { sucesso: true };
  } catch (err) {
    return { sucesso: false, motivo: `Falha ao imprimir: ${err.message}` };
  }
}

module.exports = { imprimirPedido };
