const contentEl = document.getElementById('content');
const historyEl = document.getElementById('history');

function renderLoadingSkeleton() {
  contentEl.innerHTML = skeletonLines(6);
  historyEl.innerHTML = skeletonRows(2);
}

async function loadRegister() {
  renderLoadingSkeleton();
  let register;
  try {
    register = await fetchJSON('/api/cash-register/current');
  } catch (err) {
    showToast('Erro ao carregar o caixa.', 'error');
    return;
  }

  if (!register) {
    renderOpenForm();
  } else {
    renderActiveRegister(register);
  }

  loadHistory();
}

function renderOpenForm() {
  contentEl.innerHTML = `
    <div class="form-box">
      <h2>Abrir caixa</h2>
      <label for="opening-amount-input" class="sr-only">Valor inicial da gaveta</label>
      <input type="number" id="opening-amount-input" placeholder="Valor inicial (ex: 50.00)" step="0.01" min="0" />
      <button id="open-button" class="primary">Abrir caixa</button>
    </div>
  `;
  document.getElementById('open-button').addEventListener('click', openRegister);
}

function renderActiveRegister(register) {
  const openTabs = register.openTabs;

  contentEl.innerHTML = `
    <div class="summary-box">
      <div class="summary-row"><span>Aberto em</span><span>${escapeHtml(register.openedAt)}</span></div>
      <div class="summary-row"><span>Valor de abertura</span><span>${formatCurrency(register.openingAmount)}</span></div>
      <div class="summary-row"><span>Dinheiro</span><span>${formatCurrency(register.sales.cash)}</span></div>
      <div class="summary-row"><span>Pix</span><span>${formatCurrency(register.sales.pix)}</span></div>
      <div class="summary-row"><span>Cartão débito</span><span>${formatCurrency(register.sales.debit)}</span></div>
      <div class="summary-row"><span>Cartão crédito</span><span>${formatCurrency(register.sales.credit)}</span></div>
      <div class="summary-row"><span>Sangrias</span><span>-${formatCurrency(register.movements.cashOut)}</span></div>
      <div class="summary-row"><span>Suprimentos</span><span>+${formatCurrency(register.movements.cashIn)}</span></div>
      <div class="summary-row highlight"><span>Esperado na gaveta</span><span>${formatCurrency(register.expectedInDrawer)}</span></div>
      ${openTabs.count > 0 ? `<div class="summary-row warning-row"><span>Contas de mesa abertas (${openTabs.count})</span><span>${formatCurrency(openTabs.total)}</span></div>` : ''}
    </div>

    <div class="form-box">
      <h2>Movimento de caixa</h2>
      <label for="movement-type-select" class="sr-only">Tipo de movimento</label>
      <select id="movement-type-select">
        <option value="cash_out">Sangria (retirada)</option>
        <option value="cash_in">Suprimento (reforço)</option>
      </select>
      <label for="movement-amount-input" class="sr-only">Valor do movimento</label>
      <input type="number" id="movement-amount-input" placeholder="Valor" step="0.01" min="0" />
      <label for="movement-reason-input" class="sr-only">Motivo do movimento</label>
      <input type="text" id="movement-reason-input" placeholder="Motivo" />
      <button id="register-movement-button" class="secondary">Registrar</button>
    </div>

    <div class="form-box">
      <h2>Fechar caixa</h2>
      <label for="counted-amount-input" class="sr-only">Valor contado na gaveta</label>
      <input type="number" id="counted-amount-input" placeholder="Valor contado na gaveta" step="0.01" min="0" />
      <button id="close-button" class="primary">Fechar caixa</button>
    </div>
  `;

  document.getElementById('register-movement-button').addEventListener('click', registerMovement);
  document.getElementById('close-button').addEventListener('click', closeRegister);
}

async function openRegister() {
  const openingAmount = Number(document.getElementById('opening-amount-input').value);

  try {
    await fetchJSON('/api/cash-register/open', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ openingAmount }),
    });
    showToast('Caixa aberto.', 'success', 3000);
    loadRegister();
  } catch (err) {
    showToast(err.message || 'Erro ao abrir caixa.', 'error');
  }
}

async function registerMovement() {
  const type = document.getElementById('movement-type-select').value;
  const amount = Number(document.getElementById('movement-amount-input').value);
  const reason = document.getElementById('movement-reason-input').value;

  try {
    await fetchJSON('/api/cash-register/movement', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type, amount, reason }),
    });
    showToast('Movimento registrado.', 'success', 3000);
    loadRegister();
  } catch (err) {
    showToast(err.message || 'Erro ao registrar movimento.', 'error');
  }
}

async function closeRegister() {
  const countedAmount = Number(document.getElementById('counted-amount-input').value);

  try {
    const result = await fetchJSON('/api/cash-register/close', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ countedAmount }),
    });

    const sign = result.difference >= 0 ? 'sobra' : 'falta';
    showToast(
      `Caixa fechado. Esperado: ${formatCurrency(result.summary.expectedInDrawer)} | Contado: ${formatCurrency(result.countedAmount)} | Diferença: ${formatCurrency(Math.abs(result.difference))} (${sign})`,
      Math.abs(result.difference) < 0.01 ? 'success' : 'warning',
      7000
    );

    renderOpenForm();
    loadHistory();
  } catch (err) {
    showToast(err.message || 'Erro ao fechar caixa.', 'error');
  }
}

async function loadHistory() {
  try {
    const closures = await fetchJSON('/api/cash-register/closures');
    if (closures.length === 0) {
      historyEl.innerHTML = '<p class="empty-state">Nenhum fechamento registrado ainda.</p>';
      return;
    }
    historyEl.innerHTML = closures
      .map(
        (c) => `
        <div class="summary-box">
          <div class="summary-row"><span>${escapeHtml(c.openedAt)} → ${escapeHtml(c.closedAt)}</span><span>${(Math.abs(c.difference) < 0.01 ? 'bateu' : c.difference > 0 ? 'sobrou' : 'faltou') + ' ' + formatCurrency(Math.abs(c.difference))}</span></div>
        </div>
      `
      )
      .join('');
  } catch (err) {
    historyEl.innerHTML = '';
  }
}

loadRegister();
