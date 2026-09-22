<script lang="ts">
  import { Button } from '$lib/components/ui/button/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { PAYMENT_METHOD_LABELS } from '$lib/constants.js';
  import { toast } from '$lib/toast.svelte.js';
  import type { PaymentMethod } from '$lib/api.js';

  let {
    total,
    items,
    onConfirm,
    onCancel,
  }: {
    total: number;
    items: { quantity: number; name: string }[];
    onConfirm: (result: { paymentMethod: PaymentMethod; amountReceived?: number; changeDue?: number }) => void;
    onCancel: () => void;
  } = $props();

  const METHOD_KEYS: Record<string, PaymentMethod> = { '1': 'cash', '2': 'pix', '3': 'debit', '4': 'credit' };
  const METHOD_HINTS: Record<PaymentMethod, string> = { cash: '1', pix: '2', debit: '3', credit: '4' };

  function money(value: number) {
    return value.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
  }

  // Compara em centavos — em ponto flutuante, somas como 0.1 + 0.2 não batem
  // exatamente com 0.3, então "recebido === total" podia dar falso negativo.
  function cents(value: number) {
    return Math.round(value * 100);
  }

  let stage = $state<'select' | 'summary'>('select');
  let method = $state<PaymentMethod | null>(null);
  let amountReceivedInput = $state('');
  let amountReceived = $state(0);
  let changeDue = $state(0);

  let changeDueDisplay = $derived.by(() => {
    const received = Number(amountReceivedInput);
    if (!Number.isFinite(received) || cents(received) < cents(total)) return '';
    return `Troco: ${money(received - total)}`;
  });

  function tryConfirm() {
    if (!method) {
      toast('Escolha a forma de pagamento.', 'error', 3000);
      return;
    }
    if (method === 'cash') {
      const received = Number(amountReceivedInput);
      if (!Number.isFinite(received) || cents(received) < cents(total)) {
        toast('Informe um valor recebido válido (maior ou igual ao total).', 'error', 3000);
        return;
      }
      amountReceived = received;
      changeDue = Math.round((received - total) * 100) / 100;
    }
    stage = 'summary';
  }

  function finish() {
    if (!method) return;
    onConfirm({
      paymentMethod: method,
      amountReceived: method === 'cash' ? amountReceived : undefined,
      changeDue: method === 'cash' ? changeDue : undefined,
    });
  }

  function handleKeydown(ev: KeyboardEvent) {
    if (stage === 'summary') {
      if (ev.key === 'Escape') {
        ev.preventDefault();
        stage = 'select';
      } else if (ev.key === 'Enter') {
        ev.preventDefault();
        finish();
      }
      return;
    }
    if (ev.key === 'Escape') {
      ev.preventDefault();
      onCancel();
      return;
    }
    const activeTag = (document.activeElement as HTMLElement | null)?.tagName;
    if (activeTag === 'INPUT') {
      // Só o Enter é tratado aqui (único lugar) — evita que o Enter
      // avance para "summary" e, no mesmo evento, já confirme o pagamento.
      if (ev.key === 'Enter') {
        ev.preventDefault();
        tryConfirm();
      }
      return;
    }
    if (ev.key in METHOD_KEYS) {
      ev.preventDefault();
      method = METHOD_KEYS[ev.key];
    } else if (ev.key === 'Enter') {
      ev.preventDefault();
      tryConfirm();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if stage === 'select'}
  <Dialog.Header>
    <Dialog.Title>Pagamento — {money(total)}</Dialog.Title>
  </Dialog.Header>
  <div class="grid grid-cols-2 gap-2">
    <Button variant={method === 'cash' ? 'default' : 'secondary'} onclick={() => (method = 'cash')} class="justify-between px-4">
      Dinheiro <kbd class="opacity-70">{METHOD_HINTS.cash}</kbd>
    </Button>
    <Button variant={method === 'pix' ? 'default' : 'secondary'} onclick={() => (method = 'pix')} class="justify-between px-4">
      Pix <kbd class="opacity-70">{METHOD_HINTS.pix}</kbd>
    </Button>
    <Button variant={method === 'debit' ? 'default' : 'secondary'} onclick={() => (method = 'debit')} class="justify-between px-4">
      Débito <kbd class="opacity-70">{METHOD_HINTS.debit}</kbd>
    </Button>
    <Button variant={method === 'credit' ? 'default' : 'secondary'} onclick={() => (method = 'credit')} class="justify-between px-4">
      Crédito <kbd class="opacity-70">{METHOD_HINTS.credit}</kbd>
    </Button>
  </div>
  {#if method === 'cash'}
    <div class="flex flex-col gap-2">
      <Input
        type="number"
        step="0.01"
        min="0"
        placeholder="Valor recebido"
        bind:value={amountReceivedInput}
      />
      <div class="flex gap-2">
        <Button variant="outline" size="sm" onclick={() => (amountReceivedInput = String(total))}>Exato</Button>
        <Button variant="outline" size="sm" onclick={() => (amountReceivedInput = '20')}>20</Button>
        <Button variant="outline" size="sm" onclick={() => (amountReceivedInput = '50')}>50</Button>
        <Button variant="outline" size="sm" onclick={() => (amountReceivedInput = '100')}>100</Button>
      </div>
      {#if changeDueDisplay}
        <p class="font-bold text-success" role="status" aria-live="polite">{changeDueDisplay}</p>
      {/if}
    </div>
  {/if}
  <Dialog.Footer>
    <Button onclick={tryConfirm} class="justify-between px-4">Confirmar <kbd class="opacity-70">Enter</kbd></Button>
    <Button variant="secondary" onclick={onCancel} class="justify-between px-4">Cancelar <kbd class="opacity-70">Esc</kbd></Button>
  </Dialog.Footer>
{:else}
  <Dialog.Header>
    <Dialog.Title>Pagamento confirmado</Dialog.Title>
  </Dialog.Header>
  <div class="flex flex-col gap-1.5 border border-dashed border-border bg-muted/30 p-3 font-mono text-[13px]">
    {#each items as item, i (i)}
      <div class="flex justify-between"><span class="uppercase">{item.quantity}x {item.name}</span></div>
    {/each}
    <div class="mt-1 flex justify-between border-t-2 border-foreground pt-2 font-bold"><span>Total</span><span class="tabular-nums">{money(total)}</span></div>
    <div class="flex justify-between text-muted-foreground"><span>Forma de pagamento</span><span>{method ? PAYMENT_METHOD_LABELS[method] : ''}</span></div>
    {#if method === 'cash'}
      <div class="flex justify-between text-muted-foreground"><span>Recebido</span><span class="tabular-nums">{money(amountReceived)}</span></div>
      <div class="flex justify-between font-bold text-success"><span>Troco</span><span class="tabular-nums">{money(changeDue)}</span></div>
    {/if}
  </div>
  <Dialog.Footer>
    <Button onclick={finish} class="justify-between px-4">OK <kbd class="opacity-70">Enter</kbd></Button>
  </Dialog.Footer>
{/if}
