const summaryEl = document.getElementById('day-summary');
const filtersEl = document.getElementById('filters');
const listEl = document.getElementById('order-list');

const FILTERS = [
  { id: 'open', label: 'Contas abertas', status: 'open' },
  { id: 'paid', label: 'Pagos', status: 'paid' },
  { id: 'canceled', label: 'Cancelados', status: 'canceled' },
  { id: 'all', label: 'Todos', status: null },
];

// The cashier types this in Portuguese via prompt(); we translate it to the
// English enum the API expects.
const PAYMENT_METHOD_INPUT = { dinheiro: 'cash', pix: 'pix', debito: 'debit', débito: 'debit', credito: 'credit', crédito: 'credit' };

let currentFilter = 'open';
let orders = [];

function sourceLabel(order) {
  if (order.source === 'table') return `Mesa ${order.reference || '?'}`;
  return order.reference || 'Balcão';
}

const STATUS_LABELS = { open: 'Aberto', paid: 'Pago', canceled: 'Cancelado' };

function renderLoadingSkeleton() {
  summaryEl.innerHTML = skeletonLines(4);
  listEl.innerHTML = skeletonRows(4);
}

async function load() {
  renderLoadingSkeleton();
  try {
    const filter = FILTERS.find((f) => f.id === currentFilter);
    const qs = filter && filter.status ? `?status=${filter.status}` : '';
    orders = await fetchJSON(`/api/orders/today${qs}`);
  } catch (err) {
    showToast('Erro ao carregar pedidos do dia.', 'error');
    return;
  }
  renderSummary();
  renderFilters();
  renderList();
}

async function renderSummary() {
  try {
    const report = await fetchJSON('/api/report/day');
    const openTabs = report.summary.openTabs;
    summaryEl.innerHTML = `
      <div class="summary-row"><span>Pedidos pagos</span><span>${report.summary.paidOrders}</span></div>
      <div class="summary-row"><span>Cancelados</span><span>${report.summary.canceledOrders}</span></div>
      <div class="summary-row ${openTabs > 0 ? 'highlight' : ''}"><span>Contas abertas</span><span>${openTabs}</span></div>
      <div class="summary-row highlight"><span>Total vendido hoje</span><span>${formatCurrency(report.summary.totalSold)}</span></div>
    `;
  } catch (err) {
    summaryEl.innerHTML = '';
  }
}

function renderFilters() {
  filtersEl.innerHTML = FILTERS.map((f) => {
    const isActive = f.id === currentFilter;
    return `<button type="button" role="tab" aria-selected="${isActive}" class="category-tab ${isActive ? 'active' : ''}" data-id="${f.id}">${f.label}</button>`;
  }).join('');
  filtersEl.querySelectorAll('button').forEach((button) => {
    button.addEventListener('click', () => {
      currentFilter = button.dataset.id;
      load();
    });
  });
}

function renderList() {
  if (orders.length === 0) {
    listEl.innerHTML = '<p class="empty-state">Nenhum pedido nessa categoria hoje.</p>';
    return;
  }

  listEl.innerHTML = orders
    .map((order) => {
      const items = order.items
        .map((item) => `${item.quantity}x ${escapeHtml(item.name)}${item.note ? ` (${escapeHtml(item.note)})` : ''}`)
        .join(', ');

      const payment =
        order.status === 'paid'
          ? `<span class="payment-method">${PAYMENT_METHOD_LABELS[order.paymentMethod] || order.paymentMethod}</span>`
          : '';

      const actions = [];
      if (order.status === 'open') {
        actions.push(`<button class="secondary" data-action="pay" data-id="${order.id}">Receber pagamento</button>`);
      }
      if (order.status !== 'canceled') {
        actions.push(`<button class="secondary" data-action="reprint-kitchen" data-id="${order.id}">Reimprimir cozinha</button>`);
        actions.push(`<button class="secondary" data-action="reprint-label" data-id="${order.id}">Reimprimir etiqueta</button>`);
        actions.push(`<button class="secondary danger" data-action="cancel" data-id="${order.id}">Cancelar</button>`);
      }

      return `
        <li class="order-card status-${order.status}">
          <div class="order-card-header">
            <span class="order-number">#${order.dailyNumber}</span>
            <span>${escapeHtml(sourceLabel(order))}</span>
            <span class="status-badge">${STATUS_LABELS[order.status]}</span>
            ${payment}
            <span class="order-total">${formatCurrency(order.total)}</span>
          </div>
          <div class="order-card-items">${items}</div>
          ${order.cancellationReason ? `<div class="order-card-reason">Motivo: ${escapeHtml(order.cancellationReason)}</div>` : ''}
          <div class="order-card-actions">${actions.join('')}</div>
        </li>
      `;
    })
    .join('');

  listEl.querySelectorAll('button[data-action]').forEach((button) => {
    button.addEventListener('click', () => handleAction(button.dataset.action, Number(button.dataset.id)));
  });
}

async function handleAction(action, id) {
  try {
    if (action === 'cancel') {
      const reason = prompt('Motivo do cancelamento:');
      if (!reason || !reason.trim()) return;
      await fetchJSON(`/api/orders/${id}/cancel`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason }),
      });
      showToast('Pedido cancelado.', 'success', 3000);
    } else if (action === 'pay') {
      await receivePayment(id);
      return;
    } else if (action === 'reprint-kitchen' || action === 'reprint-label') {
      const documentType = action === 'reprint-kitchen' ? 'kitchen' : 'label';
      const result = await fetchJSON(`/api/orders/${id}/reprint`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ document: documentType }),
      });
      if (result.success) {
        showToast('Reimpressão enviada.', 'success', 3000);
      } else {
        showToast(`Não foi possível reimprimir: ${result.reason}`, 'error', 5000);
      }
    }
    load();
  } catch (err) {
    showToast(err.message || 'Erro ao executar ação.', 'error', 4000);
  }
}

async function receivePayment(id) {
  const typedMethod = prompt('Forma de pagamento (dinheiro, pix, debito, credito):', 'dinheiro');
  if (!typedMethod) return;
  const paymentMethod = PAYMENT_METHOD_INPUT[typedMethod.trim().toLowerCase()];
  if (!paymentMethod) {
    showToast('Forma de pagamento não reconhecida.', 'error', 4000);
    return;
  }
  const payment = { paymentMethod };

  if (paymentMethod === 'cash') {
    const amount = prompt('Valor recebido:');
    if (!amount) return;
    payment.amountReceived = Number(amount);
  }

  try {
    const result = await fetchJSON(`/api/orders/${id}/pay`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payment),
    });

    if (result.receipt && !result.receipt.success) {
      showToast(`Pagamento registrado, mas não foi possível imprimir o recibo: ${result.receipt.reason}`, 'warning', 6000);
    } else {
      showToast('Pagamento registrado.', 'success', 3000);
    }
    load();
  } catch (err) {
    showToast(err.message || 'Erro ao registrar pagamento.', 'error', 4000);
  }
}

load();
