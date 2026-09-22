export type Category = {
  id: number;
  name: string;
  sortOrder: number;
  active: boolean;
};

export type Product = {
  id: number;
  name: string;
  price: number;
  categoryId: number | null;
  active: boolean;
  availableToday: boolean;
  sortOrder: number;
};

export type PaymentMethodSales = { cash: number; pix: number; debit: number; credit: number };
export type MovementTotals = { cashOut: number; cashIn: number };
export type OpenTabsSummary = { count: number; total: number };

export type CashRegister = {
  id: number;
  openingAmount: number;
  openedAt: string;
  closedAt: string | null;
  countedAmount: number | null;
  totalCash: number | null;
  totalPix: number | null;
  totalDebit: number | null;
  totalCredit: number | null;
  status: 'open' | 'closed';
};

export type CashRegisterSummary = CashRegister & {
  sales: PaymentMethodSales;
  movements: MovementTotals;
  totalSales: number;
  expectedInDrawer: number;
  openTabs: OpenTabsSummary;
};

export type CashRegisterClosure = CashRegister & {
  expectedAmount: number;
  difference: number;
};

export type CloseResult = {
  register: CashRegisterClosure;
  summary: CashRegisterSummary;
  countedAmount: number;
  difference: number;
};

export type PaymentMethod = 'cash' | 'pix' | 'debit' | 'credit';
export type OrderStatus = 'open' | 'paid' | 'canceled';
export type OrderSource = 'counter' | 'table';

export type OrderItem = {
  id: number;
  orderId: number;
  productId: number | null;
  name: string;
  unitPrice: number;
  quantity: number;
  note: string | null;
};

export type Order = {
  id: number;
  dailyNumber: number;
  date: string;
  source: OrderSource;
  reference: string | null;
  status: OrderStatus;
  total: number;
  paymentMethod: PaymentMethod | null;
  amountReceived: number | null;
  changeDue: number | null;
  note: string | null;
  createdAt: string;
  paidAt: string | null;
  canceledAt: string | null;
  cancellationReason: string | null;
  items: OrderItem[];
};

export type ReprintResult = { success: boolean; reason?: string };

export type CreateOrderResult = {
  order: Order;
  printing: { success: boolean; kitchen: ReprintResult; label: ReprintResult; receipt: ReprintResult | null };
  receipt: ReprintResult | null;
};

export type DayReport = {
  date: string;
  summary: { paidOrders: number; canceledOrders: number; openTabs: number; totalSold: number };
  byPaymentMethod: { paymentMethod: PaymentMethod; count: number; total: number }[];
  topProducts: { name: string; quantity: number; total: number }[];
};

export class ApiError extends Error {
  data: unknown;
  constructor(message: string, data: unknown) {
    super(message);
    this.data = data;
  }
}

export async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, options);
  let data: unknown = null;
  try {
    data = await response.json();
  } catch {
    data = null;
  }
  if (!response.ok) {
    const message =
      data && typeof data === 'object' && 'error' in data && typeof (data as { error: unknown }).error === 'string'
        ? (data as { error: string }).error
        : `Erro ${response.status}`;
    throw new ApiError(message, data);
  }
  return data as T;
}

const jsonHeaders = { 'Content-Type': 'application/json' };

export const api = {
  listCategories: () => fetchJSON<Category[]>('/api/categories'),
  createCategory: (name: string) =>
    fetchJSON<Category>('/api/categories', { method: 'POST', headers: jsonHeaders, body: JSON.stringify({ name }) }),

  listProducts: (opts?: { includeInactive?: boolean; availableToday?: boolean }) => {
    const params = new URLSearchParams();
    if (opts?.includeInactive) params.set('includeInactive', '1');
    if (opts?.availableToday) params.set('availableToday', '1');
    const qs = params.toString();
    return fetchJSON<Product[]>(`/api/products${qs ? `?${qs}` : ''}`);
  },
  createProduct: (input: { name: string; price: number; categoryId: number | null }) =>
    fetchJSON<Product>('/api/products', { method: 'POST', headers: jsonHeaders, body: JSON.stringify(input) }),
  updateProduct: (id: number, input: Partial<{ name: string; price: number; categoryId: number | null; active: boolean }>) =>
    fetchJSON<Product>(`/api/products/${id}`, { method: 'PUT', headers: jsonHeaders, body: JSON.stringify(input) }),
  setAvailability: (id: number, availableToday: boolean) =>
    fetchJSON<Product>(`/api/products/${id}/availability`, {
      method: 'PATCH',
      headers: jsonHeaders,
      body: JSON.stringify({ availableToday }),
    }),
  removeProduct: (id: number) => fetchJSON<void>(`/api/products/${id}`, { method: 'DELETE' }),

  getCurrentRegister: () => fetchJSON<CashRegisterSummary | null>('/api/cash-register/current'),
  openRegister: (openingAmount: number) =>
    fetchJSON<CashRegister>('/api/cash-register/open', {
      method: 'POST',
      headers: jsonHeaders,
      body: JSON.stringify({ openingAmount }),
    }),
  registerMovement: (input: { type: 'cash_out' | 'cash_in'; amount: number; reason: string }) =>
    fetchJSON<CashRegisterSummary>('/api/cash-register/movement', {
      method: 'POST',
      headers: jsonHeaders,
      body: JSON.stringify(input),
    }),
  closeRegister: (countedAmount: number) =>
    fetchJSON<CloseResult>('/api/cash-register/close', {
      method: 'POST',
      headers: jsonHeaders,
      body: JSON.stringify({ countedAmount }),
    }),
  listClosures: () => fetchJSON<CashRegisterClosure[]>('/api/cash-register/closures'),

  createOrder: (input: {
    source: OrderSource;
    reference: string | null;
    items: { productId: number; quantity: number; note: string | null }[];
    note: string | null;
    payment: { paymentMethod: PaymentMethod; amountReceived?: number } | null;
  }) => fetchJSON<CreateOrderResult>('/api/orders', { method: 'POST', headers: jsonHeaders, body: JSON.stringify(input) }),
  listOrdersToday: (status?: OrderStatus) => fetchJSON<Order[]>(`/api/orders/today${status ? `?status=${status}` : ''}`),
  cancelOrder: (id: number, reason: string) =>
    fetchJSON<Order>(`/api/orders/${id}/cancel`, { method: 'POST', headers: jsonHeaders, body: JSON.stringify({ reason }) }),
  payOrder: (id: number, payment: { paymentMethod: PaymentMethod; amountReceived?: number }) =>
    fetchJSON<{ order: Order; receipt: ReprintResult }>(`/api/orders/${id}/pay`, {
      method: 'POST',
      headers: jsonHeaders,
      body: JSON.stringify(payment),
    }),
  reprint: (id: number, document: 'kitchen' | 'label' | 'receipt') =>
    fetchJSON<ReprintResult>(`/api/orders/${id}/reprint`, {
      method: 'POST',
      headers: jsonHeaders,
      body: JSON.stringify({ document }),
    }),
  getDayReport: () => fetchJSON<DayReport>('/api/report/day'),
};
