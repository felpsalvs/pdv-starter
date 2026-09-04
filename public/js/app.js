const PAYMENT_METHOD_LABELS = { cash: 'Dinheiro', pix: 'Pix', debit: 'Débito', credit: 'Crédito' };
const DEFAULT_TOAST_DURATION_MS = 4000;

function formatCurrency(value) {
  return Number(value || 0).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

function escapeHtml(text) {
  return String(text ?? '').replace(/[&<>"']/g, (c) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  }[c]));
}

// --- Toasts: one small notification per action, stacked top-right. ---

let toastRegion = null;

function getToastRegion() {
  if (toastRegion) return toastRegion;
  toastRegion = document.createElement('div');
  toastRegion.className = 'toast-region';
  toastRegion.setAttribute('role', 'status');
  toastRegion.setAttribute('aria-live', 'polite');
  document.body.appendChild(toastRegion);
  return toastRegion;
}

function showToast(message, type, durationMs) {
  const region = getToastRegion();
  const toast = document.createElement('div');
  toast.className = `toast ${type || 'success'}`;

  const text = document.createElement('span');
  text.className = 'toast-message';
  text.textContent = message;

  const closeButton = document.createElement('button');
  closeButton.type = 'button';
  closeButton.className = 'toast-close';
  closeButton.setAttribute('aria-label', 'Fechar aviso');
  closeButton.textContent = '×';

  const dismiss = () => {
    toast.classList.add('leaving');
    toast.addEventListener('animationend', () => toast.remove(), { once: true });
    // Sem animação (prefers-reduced-motion), garante a remoção mesmo assim.
    setTimeout(() => toast.remove(), 200);
  };
  closeButton.addEventListener('click', dismiss);

  toast.appendChild(text);
  toast.appendChild(closeButton);
  region.prepend(toast);

  setTimeout(dismiss, durationMs || DEFAULT_TOAST_DURATION_MS);
}

// --- Skeletons: loading placeholders shown while a page's first fetch is in flight. ---

function skeletonCards(count) {
  return Array.from({ length: count }, () => '<div class="skeleton skeleton-card"></div>').join('');
}

function skeletonRows(count) {
  return Array.from({ length: count }, () => '<div class="skeleton skeleton-row"></div>').join('');
}

function skeletonLines(count) {
  return Array.from({ length: count }, () => '<div class="skeleton skeleton-line"></div>').join('');
}

async function fetchJSON(url, options) {
  const response = await fetch(url, options);
  let data = null;
  try {
    data = await response.json();
  } catch (err) {
    data = null;
  }
  if (!response.ok) {
    const error = new Error((data && data.error) || `Erro ${response.status}`);
    error.data = data;
    throw error;
  }
  return data;
}
