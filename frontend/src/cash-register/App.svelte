<script lang="ts">
  import { api, ApiError, type CashRegisterSummary, type CashRegisterClosure } from '$lib/api.js';
  import { toast } from '$lib/toast.svelte.js';
  import { parseAmount } from '$lib/utils.js';
  import Toaster from '$lib/components/Toaster.svelte';
  import Nav from '$lib/components/Nav.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';

  function money(value: number) {
    return value.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
  }

  function errorMessage(err: unknown, fallback: string) {
    return err instanceof ApiError ? err.message : fallback;
  }

  // "2026-09-22 08:00:00" -> "22/09 08:00"
  function formatDateTime(raw: string) {
    const [datePart, timePart] = raw.split(' ');
    const [, month, day] = datePart.split('-');
    const hm = timePart ? timePart.slice(0, 5) : '';
    return `${day}/${month}${hm ? ` ${hm}` : ''}`;
  }

  let loading = $state(true);
  let register = $state<CashRegisterSummary | null>(null);
  let closures = $state<CashRegisterClosure[]>([]);
  let busy = $state(false);

  let openingAmount = $state('');
  let movementType = $state<'cash_out' | 'cash_in'>('cash_out');
  let movementAmount = $state('');
  let movementReason = $state('');
  let countedAmount = $state('');
  let pendingClose = $state<number | null>(null);

  const pendingCloseDiff = $derived.by(() => {
    if (pendingClose === null || !register) return 0;
    return Math.round((pendingClose - register.expectedInDrawer) * 100) / 100;
  });

  async function loadRegister() {
    loading = true;
    try {
      register = await api.getCurrentRegister();
    } catch {
      toast('Erro ao carregar o caixa.', 'error');
    } finally {
      loading = false;
    }
  }

  async function loadHistory() {
    try {
      closures = await api.listClosures();
    } catch {
      closures = [];
    }
  }

  async function openRegister() {
    if (busy) return;
    const amount = parseAmount(openingAmount);
    if (amount === null || amount < 0) {
      toast('Informe um valor inicial válido.', 'error');
      return;
    }
    busy = true;
    try {
      await api.openRegister(amount);
      openingAmount = '';
      toast('Caixa aberto.', 'success', 3000);
      await loadRegister();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao abrir caixa.'), 'error');
    } finally {
      busy = false;
    }
  }

  async function submitMovement() {
    if (busy) return;
    const amount = parseAmount(movementAmount);
    if (amount === null || amount <= 0) {
      toast('Informe um valor válido.', 'error');
      return;
    }
    if (!movementReason.trim()) {
      toast('Informe o motivo do movimento.', 'error');
      return;
    }
    busy = true;
    try {
      await api.registerMovement({
        type: movementType,
        amount,
        reason: movementReason.trim(),
      });
      movementAmount = '';
      movementReason = '';
      toast('Movimento registrado.', 'success', 3000);
      await loadRegister();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao registrar movimento.'), 'error');
    } finally {
      busy = false;
    }
  }

  function startCloseRegister() {
    const amount = parseAmount(countedAmount);
    if (amount === null || amount < 0) {
      toast('Informe o valor contado na gaveta.', 'error');
      return;
    }
    pendingClose = amount;
  }

  async function confirmCloseRegister() {
    if (busy || pendingClose === null) return;
    const amount = pendingClose;
    pendingClose = null;
    busy = true;
    try {
      const result = await api.closeRegister(amount);
      const sign = result.difference >= 0 ? 'sobra' : 'falta';
      toast(
        `Caixa fechado. Esperado: ${money(result.summary.expectedInDrawer)} | Contado: ${money(result.countedAmount)} | Diferença: ${money(Math.abs(result.difference))} (${sign})`,
        Math.abs(result.difference) < 0.01 ? 'success' : 'warning',
        7000
      );
      countedAmount = '';
      await loadRegister();
      await loadHistory();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao fechar caixa.'), 'error');
    } finally {
      busy = false;
    }
  }

  async function init() {
    await loadRegister();
    await loadHistory();
  }
  init();
</script>

<Toaster />
<Nav active="cash-register" />

<div class="pl-52">
  <main id="main-content" class="mx-auto max-w-3xl p-6">
  <h1 class="mb-4 text-2xl font-bold">Caixa</h1>

  {#if loading}
    <div class="flex flex-col gap-2">
      {#each Array(6) as _}
        <div class="h-6 animate-pulse rounded bg-muted"></div>
      {/each}
    </div>
  {:else if !register}
    <div class="max-w-md rounded-lg border bg-card p-4 shadow-sm">
      <h2 class="mb-3 text-lg font-semibold">Abrir caixa</h2>
      <Label for="opening-amount-input" class="sr-only">Valor inicial da gaveta</Label>
      <Input
        id="opening-amount-input"
        type="number"
        step="0.01"
        min="0"
        placeholder="Valor inicial (ex: 50.00)"
        bind:value={openingAmount}
        class="mb-3"
      />
      <Button onclick={openRegister} disabled={busy}>Abrir caixa</Button>
    </div>
  {:else}
    <div class="mb-6 flex flex-col gap-1 rounded-lg border bg-card p-4 shadow-sm">
      <div class="flex justify-between py-1 text-sm"><span>Aberto em</span><span class="tabular-nums">{formatDateTime(register.openedAt)}</span></div>
      <div class="flex justify-between py-1 text-sm"><span>Valor de abertura</span><span class="tabular-nums">{money(register.openingAmount)}</span></div>
      <div class="flex justify-between py-1 text-sm"><span>Dinheiro</span><span class="tabular-nums">{money(register.sales.cash)}</span></div>
      <div class="flex justify-between py-1 text-sm"><span>Pix</span><span class="tabular-nums">{money(register.sales.pix)}</span></div>
      <div class="flex justify-between py-1 text-sm"><span>Cartão débito</span><span class="tabular-nums">{money(register.sales.debit)}</span></div>
      <div class="flex justify-between py-1 text-sm"><span>Cartão crédito</span><span class="tabular-nums">{money(register.sales.credit)}</span></div>
      <div class="flex justify-between py-1 text-sm"><span>Sangrias</span><span class="tabular-nums">-{money(register.movements.cashOut)}</span></div>
      <div class="flex justify-between py-1 text-sm"><span>Suprimentos</span><span class="tabular-nums">+{money(register.movements.cashIn)}</span></div>
      <div class="mt-1 flex justify-between border-t pt-2 text-sm font-bold">
        <span>Esperado na gaveta</span><span class="tabular-nums">{money(register.expectedInDrawer)}</span>
      </div>
      {#if register.openTabs.count > 0}
        <div class="flex justify-between py-1 text-sm font-semibold text-warning">
          <span>Contas de mesa abertas ({register.openTabs.count})</span>
          <span class="tabular-nums">{money(register.openTabs.total)}</span>
        </div>
      {/if}
    </div>

    <div class="mb-6 max-w-md rounded-lg border bg-card p-4 shadow-sm">
      <h2 class="mb-3 text-lg font-semibold">Movimento de caixa</h2>
      <Label for="movement-type-select" class="sr-only">Tipo de movimento</Label>
      <select
        id="movement-type-select"
        bind:value={movementType}
        class="border-input mb-3 flex h-9 w-full rounded-md border bg-transparent px-3 py-1 text-sm shadow-sm"
      >
        <option value="cash_out">Sangria (retirada)</option>
        <option value="cash_in">Suprimento (reforço)</option>
      </select>
      <Label for="movement-amount-input" class="sr-only">Valor do movimento</Label>
      <Input id="movement-amount-input" type="number" step="0.01" min="0" placeholder="Valor" bind:value={movementAmount} class="mb-3" />
      <Label for="movement-reason-input" class="sr-only">Motivo do movimento</Label>
      <Input id="movement-reason-input" placeholder="Motivo" bind:value={movementReason} class="mb-3" />
      <Button variant="secondary" onclick={submitMovement} disabled={busy}>Registrar</Button>
    </div>

    <div class="max-w-md rounded-lg border bg-card p-4 shadow-sm">
      <h2 class="mb-3 text-lg font-semibold">Fechar caixa</h2>
      <Label for="counted-amount-input" class="sr-only">Valor contado na gaveta</Label>
      <Input
        id="counted-amount-input"
        type="number"
        step="0.01"
        min="0"
        placeholder="Valor contado na gaveta"
        bind:value={countedAmount}
        class="mb-3"
      />
      <Button onclick={startCloseRegister} disabled={busy}>Fechar caixa</Button>
    </div>
  {/if}

  <h2 class="mt-8 mb-3 text-lg font-semibold">Fechamentos anteriores</h2>
  {#if closures.length === 0}
    <p class="text-muted-foreground">Nenhum fechamento registrado ainda.</p>
  {:else}
    <div class="flex flex-col gap-2">
      {#each closures as closure (closure.id)}
        <div class="flex justify-between rounded-lg border bg-card p-4 text-sm shadow-sm">
          <span>{formatDateTime(closure.openedAt)} → {closure.closedAt ? formatDateTime(closure.closedAt) : '—'}</span>
          <span class="tabular-nums">
            {Math.abs(closure.difference) < 0.01
              ? 'bateu'
              : closure.difference > 0
                ? `sobrou ${money(closure.difference)}`
                : `faltou ${money(Math.abs(closure.difference))}`}
          </span>
        </div>
      {/each}
    </div>
  {/if}
</main>
</div>

<AlertDialog.Root open={pendingClose !== null} onOpenChange={(open) => !open && (pendingClose = null)}>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>Fechar caixa?</AlertDialog.Title>
      <AlertDialog.Description>
        {#if pendingClose !== null && register}
          Esperado na gaveta: {money(register.expectedInDrawer)} · Contado: {money(pendingClose)} · Diferença: {money(Math.abs(pendingCloseDiff))}
          {pendingCloseDiff >= 0 ? '(sobra)' : '(falta)'}
        {/if}
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>Voltar</AlertDialog.Cancel>
      <AlertDialog.Action disabled={busy} onclick={confirmCloseRegister}>Fechar caixa</AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
