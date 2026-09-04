const searchInput = document.getElementById('search-input');
const categoryTabsEl = document.getElementById('category-tabs');
const productListEl = document.getElementById('product-list');
const cartItemsEl = document.getElementById('cart-items');
const cartTotalEl = document.getElementById('cart-total');
const statusBarEl = document.getElementById('status-bar');
const referenceButton = document.getElementById('reference-button');
const payButton = document.getElementById('pay-button');
const sendUnpaidButton = document.getElementById('send-unpaid-button');
const clearButton = document.getElementById('clear-button');

const referenceModal = document.getElementById('reference-modal');
const referenceModalExtra = document.getElementById('reference-modal-extra');
const paymentModal = document.getElementById('payment-modal');
const paymentModalTotal = document.getElementById('payment-modal-total');
const paymentModalExtra = document.getElementById('payment-modal-extra');

let categories = [];
let products = [];
let currentCategory = 'all';
let filteredProducts = [];
let selectedIndex = 0;

let cart = [];
let lastItemIndex = -1;
let reference = { source: 'counter', value: null };
let registerInfo = null;

let activeModal = null;
let modalSource = 'counter';
let selectedPaymentMethod = null;
let submitting = false;

function focusSearch() {
  searchInput.focus();
}

function renderLoadingSkeleton() {
  productListEl.innerHTML = skeletonCards(6);
}

async function loadData() {
  renderLoadingSkeleton();
  try {
    const [categoriesResp, productsResp, registerResp] = await Promise.all([
      fetchJSON('/api/categories'),
      fetchJSON('/api/products?availableToday=1'),
      fetchJSON('/api/cash-register/current'),
    ]);
    categories = categoriesResp;
    products = productsResp;
    registerInfo = registerResp;
  } catch (err) {
    showToast('Não foi possível carregar os dados do servidor.', 'error');
  }
  renderStatusBar();
  renderTabs();
  filterAndRenderProducts();
}

function renderStatusBar() {
  if (!registerInfo) {
    statusBarEl.innerHTML = `<span class="register-warning">Caixa fechado — abra o caixa na tela "Caixa" antes de vender.</span>`;
    return;
  }
  statusBarEl.innerHTML = `
    <span>Caixa aberto às ${escapeHtml(registerInfo.openedAt.slice(11, 16))}</span>
    <span>Vendas hoje: ${formatCurrency(registerInfo.totalSales)}</span>
  `;
}

function renderTabs() {
  const tabs = [{ id: 'all', name: 'Todas' }, ...categories.map((c) => ({ id: c.id, name: c.name }))];
  categoryTabsEl.innerHTML = tabs
    .map((tab) => {
      const isActive = tab.id === currentCategory;
      return `<button type="button" role="tab" aria-selected="${isActive}" class="category-tab ${isActive ? 'active' : ''}" data-category="${tab.id}">${escapeHtml(tab.name)}</button>`;
    })
    .join('');
  categoryTabsEl.querySelectorAll('.category-tab').forEach((button) => {
    button.addEventListener('click', () => {
      const category = button.dataset.category;
      currentCategory = category === 'all' ? 'all' : Number(category);
      renderTabs();
      filterAndRenderProducts();
      focusSearch();
    });
  });
}

function parseSearch(text) {
  const match = text.trim().match(/^(\d+)\s+(.*)$/);
  if (match && match[2]) {
    const quantity = Number(match[1]);
    return { quantity: quantity > 0 ? quantity : 1, term: match[2] };
  }
  return { quantity: 1, term: text.trim() };
}

function filterAndRenderProducts() {
  const { term } = parseSearch(searchInput.value);
  const termLower = term.toLowerCase();

  filteredProducts = products.filter((p) => {
    const matchesCategory = currentCategory === 'all' || p.categoryId === currentCategory;
    const matchesTerm = !termLower || p.name.toLowerCase().includes(termLower);
    return matchesCategory && matchesTerm;
  });

  if (selectedIndex >= filteredProducts.length) selectedIndex = 0;
  renderProducts();
}

function renderProducts() {
  if (products.length === 0) {
    productListEl.innerHTML =
      '<p class="empty-state">Nenhum produto disponível hoje. Vá em "Cardápio" para marcar itens como disponíveis.</p>';
    return;
  }
  if (filteredProducts.length === 0) {
    productListEl.innerHTML = '<p class="empty-state">Nada encontrado para essa busca.</p>';
    return;
  }

  productListEl.innerHTML = filteredProducts
    .map(
      (product, index) => `
      <button type="button" class="product-card ${index === selectedIndex ? 'selected' : ''}" data-id="${product.id}">
        <span class="name">${escapeHtml(product.name)}</span>
        <span class="price">${formatCurrency(product.price)}</span>
      </button>
    `
    )
    .join('');

  productListEl.querySelectorAll('.product-card').forEach((button) => {
    button.addEventListener('click', () => {
      const product = products.find((p) => p.id === Number(button.dataset.id));
      addToCart(product, 1);
      focusSearch();
    });
  });

  const selected = productListEl.querySelector('.product-card.selected');
  if (selected) selected.scrollIntoView({ block: 'nearest' });
}

function addToCart(product, quantity) {
  if (!product) return;
  const existing = cart.find((i) => i.productId === product.id && !i.note);
  if (existing) {
    existing.quantity += quantity;
    lastItemIndex = cart.indexOf(existing);
  } else {
    cart.push({
      productId: product.id,
      name: product.name,
      unitPrice: product.price,
      quantity,
      note: '',
    });
    lastItemIndex = cart.length - 1;
  }
  renderCart();
}

function changeQuantity(index, delta) {
  const item = cart[index];
  if (!item) return;
  item.quantity += delta;
  if (item.quantity <= 0) {
    cart.splice(index, 1);
    lastItemIndex = cart.length - 1;
  }
  renderCart();
}

function calculateTotal() {
  return cart.reduce((sum, item) => sum + item.unitPrice * item.quantity, 0);
}

function renderCart() {
  if (cart.length === 0) {
    cartItemsEl.innerHTML = '<p class="empty-state">Carrinho vazio. Digite o nome de uma sopa e aperte Enter.</p>';
  } else {
    cartItemsEl.innerHTML = cart
      .map(
        (item, index) => `
        <div class="cart-item" data-index="${index}">
          <div class="cart-item-row">
            <span>${item.quantity}x ${escapeHtml(item.name)}</span>
            <span>${formatCurrency(item.unitPrice * item.quantity)}</span>
          </div>
          <div class="cart-item-controls">
            <label for="item-note-${index}" class="sr-only">Observação para ${escapeHtml(item.name)}</label>
            <input type="text" id="item-note-${index}" class="item-note-input" data-index="${index}" placeholder="observação (ex: sem cebola)" value="${escapeHtml(item.note)}" />
            <button type="button" data-index="${index}" data-delta="-1" aria-label="Diminuir quantidade de ${escapeHtml(item.name)}">−</button>
            <button type="button" data-index="${index}" data-delta="1" aria-label="Aumentar quantidade de ${escapeHtml(item.name)}">+</button>
          </div>
        </div>
      `
      )
      .join('');

    cartItemsEl.querySelectorAll('button[data-delta]').forEach((button) => {
      button.addEventListener('click', () => changeQuantity(Number(button.dataset.index), Number(button.dataset.delta)));
    });
    cartItemsEl.querySelectorAll('.item-note-input').forEach((input) => {
      input.addEventListener('change', () => {
        const item = cart[Number(input.dataset.index)];
        if (item) item.note = input.value.trim();
      });
      input.addEventListener('keydown', (ev) => {
        if (ev.key === 'Enter' || ev.key === 'Escape') {
          ev.target.blur();
          focusSearch();
        }
      });
    });
  }

  cartTotalEl.textContent = formatCurrency(calculateTotal());
  payButton.disabled = cart.length === 0;
  sendUnpaidButton.disabled = cart.length === 0;
}

function focusLastItemNote() {
  if (lastItemIndex < 0 || !cart[lastItemIndex]) return;
  const input = cartItemsEl.querySelector(`.item-note-input[data-index="${lastItemIndex}"]`);
  if (input) input.focus();
}

function clearOrder() {
  cart = [];
  lastItemIndex = -1;
  reference = { source: 'counter', value: null };
  updateReferenceChip();
  searchInput.value = '';
  renderCart();
  filterAndRenderProducts();
  focusSearch();
}

function updateReferenceChip() {
  if (reference.source === 'table') {
    referenceButton.innerHTML = `Mesa ${escapeHtml(reference.value || '?')} <span class="key-hint">F2</span>`;
  } else if (reference.value) {
    referenceButton.innerHTML = `${escapeHtml(reference.value)} <span class="key-hint">F2</span>`;
  } else {
    referenceButton.innerHTML = `Balcão <span class="key-hint">F2</span>`;
  }
}

// --- Reference modal ---

function openReferenceModal() {
  searchInput.blur();
  activeModal = 'reference';
  modalSource = reference.source;
  renderReferenceModalExtra();
  referenceModal.classList.remove('hidden');
  const extraInput = referenceModalExtra.querySelector('input');
  if (extraInput) extraInput.focus();
}

function renderReferenceModalExtra() {
  referenceModal.querySelectorAll('[data-source]').forEach((button) => {
    const isSelected = button.dataset.source === modalSource;
    button.classList.toggle('selected', isSelected);
    button.setAttribute('aria-pressed', String(isSelected));
  });
  if (modalSource === 'table') {
    referenceModalExtra.innerHTML = `<label for="table-number-input" class="sr-only">Número da mesa</label><input type="text" id="table-number-input" placeholder="Número da mesa" value="${escapeHtml(reference.source === 'table' ? reference.value || '' : '')}" />`;
  } else {
    referenceModalExtra.innerHTML = `<label for="customer-name-input" class="sr-only">Nome do cliente</label><input type="text" id="customer-name-input" placeholder="Nome do cliente (opcional)" value="${escapeHtml(reference.source === 'counter' ? reference.value || '' : '')}" />`;
  }
  const input = referenceModalExtra.querySelector('input');
  if (input) {
    input.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter') {
        ev.preventDefault();
        confirmReference();
      }
    });
  }
}

function confirmReference() {
  const input = referenceModalExtra.querySelector('input');
  const value = input ? input.value.trim() : '';

  if (modalSource === 'table' && !value) {
    showToast('Informe o número da mesa.', 'error', 3000);
    return;
  }

  reference = { source: modalSource, value: value || null };
  updateReferenceChip();
  closeModals();
}

function trapTabKey(ev, modalEl) {
  if (ev.key !== 'Tab') return false;
  const focusable = Array.from(modalEl.querySelectorAll('button, input, select, textarea')).filter(
    (el) => !el.disabled && el.offsetParent !== null
  );
  if (focusable.length === 0) return true;
  const first = focusable[0];
  const last = focusable[focusable.length - 1];

  if (ev.shiftKey && document.activeElement === first) {
    ev.preventDefault();
    last.focus();
  } else if (!ev.shiftKey && document.activeElement === last) {
    ev.preventDefault();
    first.focus();
  }
  return true;
}

function closeModals() {
  activeModal = null;
  referenceModal.classList.add('hidden');
  paymentModal.classList.add('hidden');
  selectedPaymentMethod = null;
  focusSearch();
}

// --- Payment modal ---

function openPaymentModal() {
  if (cart.length === 0) return;
  if (!registerInfo) {
    showToast('Abra o caixa antes de registrar um pagamento.', 'error', 4000);
    return;
  }
  searchInput.blur();
  activeModal = 'payment';
  selectedPaymentMethod = null;
  paymentModalTotal.textContent = formatCurrency(calculateTotal());
  paymentModalExtra.innerHTML = '';
  paymentModal.querySelectorAll('[data-method]').forEach((b) => b.classList.remove('selected'));
  paymentModal.classList.remove('hidden');
}

function selectPaymentMethod(method) {
  selectedPaymentMethod = method;
  paymentModal.querySelectorAll('[data-method]').forEach((b) => {
    const isSelected = b.dataset.method === method;
    b.classList.toggle('selected', isSelected);
    b.setAttribute('aria-pressed', String(isSelected));
  });

  if (method === 'cash') {
    const total = calculateTotal();
    paymentModalExtra.innerHTML = `
      <label for="amount-received-input" class="sr-only">Valor recebido</label>
      <input type="number" id="amount-received-input" placeholder="Valor recebido" step="0.01" min="0" />
      <div class="bill-shortcuts">
        <button type="button" data-amount="${total}">Exato</button>
        <button type="button" data-amount="20">20</button>
        <button type="button" data-amount="50">50</button>
        <button type="button" data-amount="100">100</button>
      </div>
      <div id="change-due-display" class="change-due-display" role="status" aria-live="polite"></div>
    `;
    const amountInput = document.getElementById('amount-received-input');
    const updateChangeDue = () => {
      const received = Number(amountInput.value);
      const changeDueEl = document.getElementById('change-due-display');
      if (!Number.isFinite(received) || received < total) {
        changeDueEl.textContent = '';
        return;
      }
      changeDueEl.textContent = `Troco: ${formatCurrency(received - total)}`;
    };
    amountInput.addEventListener('input', updateChangeDue);
    amountInput.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter') {
        ev.preventDefault();
        confirmPayment();
      }
    });
    paymentModalExtra.querySelectorAll('.bill-shortcuts button').forEach((button) => {
      button.addEventListener('click', () => {
        amountInput.value = button.dataset.amount;
        updateChangeDue();
        amountInput.focus();
      });
    });
    amountInput.focus();
  } else {
    paymentModalExtra.innerHTML = '';
  }
}

async function confirmPayment() {
  if (!selectedPaymentMethod) {
    showToast('Escolha a forma de pagamento.', 'error', 3000);
    return;
  }

  const payment = { paymentMethod: selectedPaymentMethod };
  if (selectedPaymentMethod === 'cash') {
    const amountInput = document.getElementById('amount-received-input');
    const received = Number(amountInput ? amountInput.value : NaN);
    if (!Number.isFinite(received) || received < calculateTotal()) {
      showToast('Informe um valor recebido válido (maior ou igual ao total).', 'error', 3000);
      return;
    }
    payment.amountReceived = received;
  }

  closeModals();
  await submitOrder(payment);
}

// --- Order submission ---

async function submitOrder(payment) {
  if (submitting) return;
  if (cart.length === 0) return;
  submitting = true;
  payButton.disabled = true;
  sendUnpaidButton.disabled = true;

  const items = cart.map((item) => ({
    productId: item.productId,
    quantity: item.quantity,
    note: item.note || null,
  }));

  try {
    const result = await fetchJSON('/api/orders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        source: reference.source,
        reference: reference.value,
        items,
        payment,
      }),
    });

    const { order, printing, receipt } = result;
    const documentLabels = { kitchen: 'ticket da cozinha', label: 'etiqueta', receipt: 'recibo' };
    const failures = ['kitchen', 'label', 'receipt']
      .map((doc) => ({ doc, result: doc === 'receipt' ? receipt : printing && printing[doc] }))
      .filter(({ result: r }) => r && !r.success);

    if (failures.length > 0) {
      const detail = failures.map(({ doc, result: r }) => `${documentLabels[doc]}: ${r.reason}`).join(' · ');
      showToast(
        `Pedido #${order.dailyNumber} salvo, mas não foi possível imprimir — ${detail}`,
        'warning',
        7000
      );
    } else {
      showToast(`Pedido #${order.dailyNumber} enviado para a cozinha!`, 'success', 4000);
    }

    clearOrder();
    loadData();
  } catch (err) {
    showToast(err.message || 'Erro ao enviar pedido.', 'error', 5000);
  } finally {
    submitting = false;
    renderCart();
  }
}

// --- Keyboard shortcuts ---

document.addEventListener('keydown', (ev) => {
  if (window.pdvTourActive) return;
  if (activeModal === 'reference') {
    if (trapTabKey(ev, referenceModal)) return;
    if (ev.key === 'Escape') {
      ev.preventDefault();
      closeModals();
    } else if (ev.key === '1') {
      const target = document.activeElement;
      if (target && target.tagName === 'INPUT') return;
      ev.preventDefault();
      modalSource = 'counter';
      renderReferenceModalExtra();
    } else if (ev.key === '2') {
      const target = document.activeElement;
      if (target && target.tagName === 'INPUT') return;
      ev.preventDefault();
      modalSource = 'table';
      renderReferenceModalExtra();
    } else if (ev.key === 'Enter' && ev.target.tagName !== 'INPUT') {
      ev.preventDefault();
      confirmReference();
    }
    return;
  }

  if (activeModal === 'payment') {
    if (trapTabKey(ev, paymentModal)) return;
    if (ev.key === 'Escape') {
      ev.preventDefault();
      closeModals();
    } else if (['1', '2', '3', '4'].includes(ev.key) && document.activeElement.tagName !== 'INPUT') {
      ev.preventDefault();
      const map = { 1: 'cash', 2: 'pix', 3: 'debit', 4: 'credit' };
      selectPaymentMethod(map[ev.key]);
    } else if (ev.key === 'Enter' && document.activeElement.tagName !== 'INPUT') {
      ev.preventDefault();
      confirmPayment();
    }
    return;
  }

  // No modal open
  if (ev.key === 'F2') {
    ev.preventDefault();
    openReferenceModal();
    return;
  }
  if (ev.key === 'F4') {
    ev.preventDefault();
    openPaymentModal();
    return;
  }
  if (ev.key === 'F8') {
    ev.preventDefault();
    if (cart.length > 0) submitOrder(null);
    return;
  }
  if (ev.key === 'F9') {
    ev.preventDefault();
    clearOrder();
    return;
  }

  if (ev.target === searchInput) {
    if (ev.key === 'ArrowDown') {
      ev.preventDefault();
      if (selectedIndex < filteredProducts.length - 1) selectedIndex += 1;
      renderProducts();
    } else if (ev.key === 'ArrowUp') {
      ev.preventDefault();
      if (selectedIndex > 0) selectedIndex -= 1;
      renderProducts();
    } else if (ev.key === 'Enter') {
      ev.preventDefault();
      const { quantity } = parseSearch(searchInput.value);
      const product = filteredProducts[selectedIndex];
      if (product) {
        addToCart(product, quantity);
        searchInput.value = '';
        filterAndRenderProducts();
      }
    } else if (ev.key === 'Escape') {
      ev.preventDefault();
      searchInput.value = '';
      filterAndRenderProducts();
    } else if (ev.key === '/' && searchInput.value === '') {
      ev.preventDefault();
      focusLastItemNote();
    }
  }
});

searchInput.addEventListener('input', () => {
  selectedIndex = 0;
  filterAndRenderProducts();
});

document.querySelectorAll('#reference-modal [data-source]').forEach((button) => {
  button.addEventListener('click', () => {
    modalSource = button.dataset.source;
    renderReferenceModalExtra();
  });
});
document.getElementById('confirm-reference-button').addEventListener('click', confirmReference);
document.getElementById('cancel-reference-button').addEventListener('click', closeModals);

document.querySelectorAll('#payment-modal [data-method]').forEach((button) => {
  button.addEventListener('click', () => selectPaymentMethod(button.dataset.method));
});
document.getElementById('confirm-payment-button').addEventListener('click', confirmPayment);
document.getElementById('cancel-payment-button').addEventListener('click', closeModals);

referenceButton.addEventListener('click', openReferenceModal);
payButton.addEventListener('click', openPaymentModal);
sendUnpaidButton.addEventListener('click', () => submitOrder(null));
clearButton.addEventListener('click', clearOrder);

// Keeps focus on the search box whenever possible (clicks outside text fields).
document.addEventListener('click', (ev) => {
  if (activeModal || window.pdvTourActive) return;
  const tag = ev.target.tagName;
  if (tag === 'INPUT' || tag === 'BUTTON' || tag === 'TEXTAREA') return;
  focusSearch();
});

loadData();
renderCart();
updateReferenceChip();
focusSearch();
