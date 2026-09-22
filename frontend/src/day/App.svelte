<script lang="ts">
  import { api, ApiError, type Order, type OrderStatus, type DayReport } from '$lib/api.js';
  import { PAYMENT_METHOD_LABELS, CANCEL_REASONS } from '$lib/constants.js';
  import { toast } from '$lib/toast.svelte.js';
  import Toaster from '$lib/components/Toaster.svelte';
  import Nav from '$lib/components/Nav.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
  import ReasonDialogContent from '$lib/components/ReasonDialogContent.svelte';
  import PaymentDialogContent from '$lib/components/PaymentDialogContent.svelte';

  type FilterId = 'open' | 'paid' | 'canceled' | 'all';
  const FILTERS: { id: FilterId; label: string; status: OrderStatus | null }[] = [
    { id: 'open', label: 'Contas abertas', status: 'open' },
    { id: 'paid', label: 'Pagos', status: 'paid' },
    { id: 'canceled', label: 'Cancelados', status: 'canceled' },
    { id: 'all', label: 'Todos', status: null },
  ];

  const STATUS_LABELS: Record<OrderStatus, string> = { open: 'Aberto', paid: 'Pago', canceled: 'Cancelado' };
  const STATUS_BADGE_VARIANT: Record<OrderStatus, 'warning' | 'success' | 'destructive'> = {
    open: 'warning',
    paid: 'success',
    canceled: 'destructive',
  };
  const STATUS_BORDER: Record<OrderStatus, string> = {
    open: 'border-l-warning',
    paid: 'border-l-success',
    canceled: 'border-l-destructive opacity-70',
  };

  function money(value: number) {
    return value.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
  }

  function errorMessage(err: unknown, fallback: string) {
    return err instanceof ApiError ? err.message : fallback;
  }

  function sourceLabel(order: Order) {
    if (order.source === 'table') return `Mesa ${order.reference || '?'}`;
    return order.reference || 'Balcão';
  }

  let currentFilter = $state<FilterId>('open');
  let orders = $state<Order[]>([]);
  let report = $state<DayReport | null>(null);
  let loading = $state(true);

  let cancelingOrder = $state<Order | null>(null);
  let pendingCancel = $state<{ order: Order; reason: string } | null>(null);
  let payingOrder = $state<Order | null>(null);

  async function load() {
    loading = true;
    try {
      const filter = FILTERS.find((f) => f.id === currentFilter);
      orders = await api.listOrdersToday(filter?.status ?? undefined);
    } catch {
      toast('Erro ao carregar pedidos do dia.', 'error');
    } finally {
      loading = false;
    }
    await loadReport();
  }

  async function loadReport() {
    try {
      report = await api.getDayReport();
    } catch {
      report = null;
    }
  }

  async function handleReprint(order: Order, document: 'kitchen' | 'label' | 'receipt') {
    try {
      const result = await api.reprint(order.id, document);
      if (result.success) toast('Reimpressão enviada.', 'success', 3000);
      else toast(`Não foi possível reimprimir: ${result.reason}`, 'error', 5000);
    } catch (err) {
      toast(errorMessage(err, 'Erro ao reimprimir.'), 'error');
    }
  }

  function startCancel(order: Order) {
    cancelingOrder = order;
  }

  async function reasonSelected(reason: string) {
    const order = cancelingOrder;
    cancelingOrder = null;
    if (!order) return;

    if (order.status === 'paid') {
      pendingCancel = { order, reason };
      return;
    }
    await doCancel(order.id, reason);
  }

  async function confirmPendingCancel() {
    if (!pendingCancel) return;
    const { order, reason } = pendingCancel;
    pendingCancel = null;
    await doCancel(order.id, reason);
  }

  async function doCancel(id: number, reason: string) {
    try {
      await api.cancelOrder(id, reason);
      toast('Pedido cancelado.', 'success', 3000);
      await load();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao cancelar pedido.'), 'error');
    }
  }

  function startPay(order: Order) {
    payingOrder = order;
  }

  async function paymentConfirmed(result: { paymentMethod: 'cash' | 'pix' | 'debit' | 'credit'; amountReceived?: number }) {
    const order = payingOrder;
    payingOrder = null;
    if (!order) return;
    try {
      const response = await api.payOrder(order.id, result);
      if (response.receipt && !response.receipt.success) {
        toast(`Pagamento registrado, mas não foi possível imprimir o recibo: ${response.receipt.reason}`, 'warning', 6000);
      } else {
        toast('Pagamento registrado.', 'success', 3000);
      }
      await load();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao registrar pagamento.'), 'error');
    }
  }

  load();
</script>

<Toaster />
<Nav active="day" />

<main id="main-content" class="mx-auto max-w-3xl p-4">
  <h1 class="mb-4 text-2xl font-bold">Pedidos do dia</h1>

  {#if report}
    <div class="mb-4 flex flex-col gap-1 rounded-lg border bg-card p-4 shadow-sm">
      <div class="flex justify-between py-1 text-sm"><span>Pedidos pagos</span><span>{report.summary.paidOrders}</span></div>
      <div class="flex justify-between py-1 text-sm"><span>Cancelados</span><span>{report.summary.canceledOrders}</span></div>
      <div class={`flex justify-between py-1 text-sm ${report.summary.openTabs > 0 ? 'font-bold' : ''}`}>
        <span>Contas abertas</span><span>{report.summary.openTabs}</span>
      </div>
      <div class="flex justify-between border-t pt-2 text-sm font-bold"><span>Total vendido hoje</span><span class="tabular-nums">{money(report.summary.totalSold)}</span></div>
      {#if report.byPaymentMethod.length > 0}
        <div class="mt-2 text-sm font-bold">Por forma de pagamento</div>
        {#each report.byPaymentMethod as row (row.paymentMethod)}
          <div class="flex justify-between py-1 text-sm">
            <span>{PAYMENT_METHOD_LABELS[row.paymentMethod]}</span><span class="tabular-nums">{money(row.total)}</span>
          </div>
        {/each}
      {/if}
      {#if report.topProducts.length > 0}
        <div class="mt-2 text-sm font-bold">Mais vendidos</div>
        {#each report.topProducts.slice(0, 5) as row (row.name)}
          <div class="flex justify-between py-1 text-sm">
            <span>{row.name} ({row.quantity}x)</span><span class="tabular-nums">{money(row.total)}</span>
          </div>
        {/each}
      {/if}
    </div>
  {/if}

  <div class="mb-4 flex flex-wrap gap-2" role="tablist" aria-label="Filtrar pedidos">
    {#each FILTERS as filter (filter.id)}
      <button
        type="button"
        role="tab"
        aria-selected={currentFilter === filter.id}
        class={`rounded-full border px-4 py-1.5 text-sm ${currentFilter === filter.id ? 'bg-primary text-primary-foreground' : 'bg-secondary'}`}
        onclick={() => {
          currentFilter = filter.id;
          load();
        }}
      >
        {filter.label}
      </button>
    {/each}
  </div>

  {#if loading}
    <div class="flex flex-col gap-2">
      {#each Array(4) as _}
        <div class="h-20 animate-pulse rounded-lg bg-muted"></div>
      {/each}
    </div>
  {:else if orders.length === 0}
    <p class="text-muted-foreground">Nenhum pedido nessa categoria hoje.</p>
  {:else}
    <ul class="flex flex-col gap-2" role="list" aria-label="Pedidos do dia">
      {#each orders as order (order.id)}
        <li class={`rounded-lg border border-l-4 bg-card p-4 shadow-sm ${STATUS_BORDER[order.status]}`}>
          <div class="flex flex-wrap items-center gap-3 text-sm">
            <span class="text-lg font-bold">#{order.dailyNumber}</span>
            <span>{sourceLabel(order)}</span>
            <Badge variant={STATUS_BADGE_VARIANT[order.status]}>{STATUS_LABELS[order.status]}</Badge>
            {#if order.status === 'paid' && order.paymentMethod}
              <span class="text-sm text-muted-foreground">{PAYMENT_METHOD_LABELS[order.paymentMethod]}</span>
            {/if}
            <span class="ml-auto font-bold tabular-nums">{money(order.total)}</span>
          </div>
          <div class="mt-1 text-sm text-muted-foreground">
            {order.items.map((item) => `${item.quantity}x ${item.name}${item.note ? ` (${item.note})` : ''}`).join(', ')}
          </div>
          {#if order.cancellationReason}
            <div class="mt-1 text-sm text-destructive">Motivo: {order.cancellationReason}</div>
          {/if}
          <div class="mt-3 flex flex-wrap gap-2">
            {#if order.status === 'open'}
              <Button variant="secondary" size="sm" onclick={() => startPay(order)}>Receber pagamento</Button>
            {/if}
            {#if order.status !== 'canceled'}
              <Button variant="secondary" size="sm" onclick={() => handleReprint(order, 'kitchen')}>Reimprimir cozinha</Button>
              <Button variant="secondary" size="sm" onclick={() => handleReprint(order, 'label')}>Reimprimir etiqueta</Button>
              {#if order.status === 'paid'}
                <Button variant="secondary" size="sm" onclick={() => handleReprint(order, 'receipt')}>Reimprimir recibo</Button>
              {/if}
              <Button variant="outline" size="sm" class="text-destructive" onclick={() => startCancel(order)}>Cancelar</Button>
            {/if}
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</main>

<Dialog.Root open={cancelingOrder !== null} onOpenChange={(open) => !open && (cancelingOrder = null)}>
  <Dialog.Content>
    <ReasonDialogContent title="Motivo do cancelamento" options={CANCEL_REASONS} allowOther onSelect={reasonSelected} />
  </Dialog.Content>
</Dialog.Root>

<AlertDialog.Root open={pendingCancel !== null} onOpenChange={(open) => !open && (pendingCancel = null)}>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>Cancelar um pedido já pago?</AlertDialog.Title>
      <AlertDialog.Description>
        {#if pendingCancel}
          O pedido #{pendingCancel.order.dailyNumber} ({money(pendingCancel.order.total)}) já foi pago. Cancelar agora tira esse valor
          do fechamento do caixa.
        {/if}
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>Voltar</AlertDialog.Cancel>
      <AlertDialog.Action class="bg-destructive text-destructive-foreground hover:bg-destructive/90" onclick={confirmPendingCancel}>
        Cancelar mesmo assim
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>

<Dialog.Root open={payingOrder !== null} onOpenChange={(open) => !open && (payingOrder = null)}>
  <Dialog.Content escapeKeydownBehavior="ignore">
    {#if payingOrder}
      <PaymentDialogContent
        total={payingOrder.total}
        items={payingOrder.items.map((i) => ({ quantity: i.quantity, name: i.name }))}
        onConfirm={paymentConfirmed}
        onCancel={() => (payingOrder = null)}
      />
    {/if}
  </Dialog.Content>
</Dialog.Root>
