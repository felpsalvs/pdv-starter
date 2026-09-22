import type { PaymentMethod } from './api.js';

export const PAYMENT_METHOD_LABELS: Record<PaymentMethod, string> = {
  cash: 'Dinheiro',
  pix: 'Pix',
  debit: 'Débito',
  credit: 'Crédito',
};

export const CANCEL_REASONS = ['Pedido errado', 'Cliente desistiu', 'Item em falta', 'Troco/pagamento incorreto', 'Erro de lançamento'];
