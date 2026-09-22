<script lang="ts">
  import {
    api,
    ApiError,
    type Category,
    type Product,
    type CashRegisterSummary,
    type OrderSource,
    type PaymentMethod,
  } from '$lib/api.js';
  import { toast } from '$lib/toast.svelte.js';
  import Toaster from '$lib/components/Toaster.svelte';
  import Nav from '$lib/components/Nav.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
  import PaymentDialogContent from '$lib/components/PaymentDialogContent.svelte';
  import { cn } from '$lib/utils.js';
  import Flame from '@lucide/svelte/icons/flame';

  type CartItem = { productId: number; name: string; unitPrice: number; quantity: number; note: string };

  function money(value: number) {
    return value.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
  }

  function errorMessage(err: unknown, fallback: string) {
    return err instanceof ApiError ? err.message : fallback;
  }

  function parseSearch(text: string) {
    const match = text.trim().match(/^(\d+)\s+(.*)$/);
    if (match && match[2]) {
      const quantity = Number(match[1]);
      return { quantity: quantity > 0 ? quantity : 1, term: match[2] };
    }
    return { quantity: 1, term: text.trim() };
  }

  let categories = $state<Category[]>([]);
  let products = $state<Product[]>([]);
  let registerInfo = $state<CashRegisterSummary | null>(null);
  let loading = $state(true);

  let currentCategory = $state<'all' | number>('all');
  let searchValue = $state('');
  let selectedIndex = $state(0);
  let searchInputEl = $state<HTMLInputElement | null>(null);
  let productRefs = $state<(HTMLButtonElement | null)[]>([]);

  let cart = $state<CartItem[]>([]);
  let lastItemIndex = $state(-1);
  let noteInputs = $state<(HTMLInputElement | null)[]>([]);
  let orderNote = $state('');
  let reference = $state<{ source: OrderSource; value: string | null }>({ source: 'counter', value: null });
  let submitting = $state(false);

  let referenceModalOpen = $state(false);
  let modalSource = $state<OrderSource>('counter');
  let modalValue = $state('');
  let referenceExtraInputEl = $state<HTMLInputElement | null>(null);

  let payingOpen = $state(false);
  let pendingClear = $state(false);

  let topProductNames = $state<Set<string>>(new Set());
  let lastOrder = $state<{ id: number; dailyNumber: number; paid: boolean } | null>(null);
  let reprinting = $state(false);

  const tabs = $derived.by(() => [
    { id: 'all' as const, name: 'Todas' },
    ...categories.map((c) => ({ id: c.id, name: c.name })),
  ]);

  const filteredProducts = $derived.by(() => {
    const { term } = parseSearch(searchValue);
    const termLower = term.toLowerCase();
    return products.filter((p) => {
      const matchesCategory = currentCategory === 'all' || p.categoryId === currentCategory;
      const matchesTerm = !termLower || p.name.toLowerCase().includes(termLower);
      return matchesCategory && matchesTerm;
    });
  });

  $effect(() => {
    if (selectedIndex >= filteredProducts.length) selectedIndex = 0;
  });

  $effect(() => {
    productRefs[selectedIndex]?.scrollIntoView({ block: 'nearest' });
  });

  $effect(() => {
    searchInputEl?.focus();
  });

  const cartTotal = $derived(cart.reduce((sum, item) => sum + item.unitPrice * item.quantity, 0));

  const referenceLabel = $derived.by(() => {
    if (reference.source === 'table') return `Mesa ${reference.value || '?'}`;
    if (reference.value) return reference.value;
    return 'Balcão';
  });

  function focusSearch() {
    searchInputEl?.focus();
  }

  async function loadData() {
    loading = true;
    try {
      const [categoriesResp, productsResp, registerResp] = await Promise.all([
        api.listCategories(),
        api.listProducts({ availableToday: true }),
        api.getCurrentRegister(),
      ]);
      categories = categoriesResp;
      products = productsResp;
      registerInfo = registerResp;
    } catch {
      toast('Não foi possível carregar os dados do servidor.', 'error');
    } finally {
      loading = false;
    }
    // Não crítico — se o relatório do dia falhar, só não destaca os mais pedidos.
    try {
      const report = await api.getDayReport();
      topProductNames = new Set(report.topProducts.slice(0, 5).map((p) => p.name));
    } catch {
      topProductNames = new Set();
    }
  }

  function selectCategory(id: 'all' | number) {
    currentCategory = id;
    focusSearch();
  }

  function onSearchInput() {
    selectedIndex = 0;
  }

  function addToCart(product: Product, quantity: number) {
    const existing = cart.find((i) => i.productId === product.id && !i.note);
    if (existing) {
      existing.quantity += quantity;
      lastItemIndex = cart.indexOf(existing);
    } else {
      cart.push({ productId: product.id, name: product.name, unitPrice: product.price, quantity, note: '' });
      // noteInputs precisa ter uma entrada (null, não undefined) pra cada
      // item ANTES do each-block renderizar — bind:ref num Input recém-criado
      // lendo `undefined` quebra a renderização inteira (Svelte exige um
      // valor compatível com o fallback do prop bindable).
      noteInputs.push(null);
      lastItemIndex = cart.length - 1;
    }
  }

  function changeQuantity(index: number, delta: number) {
    const item = cart[index];
    if (!item) return;
    item.quantity += delta;
    if (item.quantity <= 0) {
      cart.splice(index, 1);
      noteInputs.splice(index, 1);
      lastItemIndex = cart.length - 1;
    }
  }

  function focusLastItemNote() {
    if (lastItemIndex < 0) return;
    noteInputs[lastItemIndex]?.focus();
  }

  function clearOrder() {
    // AlertDialog.Action (ao contrário do Cancel) não fecha o diálogo
    // sozinho — quem chama decide se/quando fechar. Isso também é usado
    // pelo atalho de carrinho já vazio, onde o diálogo nem chegou a abrir.
    pendingClear = false;
    cart = [];
    noteInputs = [];
    lastItemIndex = -1;
    reference = { source: 'counter', value: null };
    searchValue = '';
    orderNote = '';
    selectedIndex = 0;
    focusSearch();
  }

  function handleClear() {
    if (cart.length === 0) {
      clearOrder();
      return;
    }
    pendingClear = true;
  }

  // --- Reference modal (F2) ---

  function selectModalSource(source: OrderSource) {
    modalSource = source;
    modalValue = reference.source === source ? reference.value ?? '' : '';
  }

  function openReferenceModal() {
    selectModalSource(reference.source);
    referenceModalOpen = true;
  }

  function confirmReference() {
    const value = modalValue.trim();
    if (modalSource === 'table' && !value) {
      toast('Informe o número da mesa.', 'error', 3000);
      return;
    }
    reference = { source: modalSource, value: value || null };
    referenceModalOpen = false;
    focusSearch();
  }

  // --- Payment (F4) ---

  function handlePaymentStart() {
    if (cart.length === 0) return;
    if (!registerInfo) {
      toast('Abra o caixa antes de registrar um pagamento.', 'error', 4000);
      return;
    }
    payingOpen = true;
  }

  function paymentConfirmed(result: { paymentMethod: PaymentMethod; amountReceived?: number }) {
    payingOpen = false;
    const payload: { paymentMethod: PaymentMethod; amountReceived?: number } = { paymentMethod: result.paymentMethod };
    if (result.paymentMethod === 'cash') payload.amountReceived = result.amountReceived;
    submitOrder(payload);
  }

  // --- Order submission ---

  async function submitOrder(payment: { paymentMethod: PaymentMethod; amountReceived?: number } | null) {
    if (submitting || cart.length === 0) return;
    submitting = true;
    try {
      const result = await api.createOrder({
        source: reference.source,
        reference: reference.value,
        items: cart.map((item) => ({ productId: item.productId, quantity: item.quantity, note: item.note || null })),
        note: orderNote.trim() || null,
        payment,
      });
      const { order, printing, receipt } = result;
      const documentLabels = { kitchen: 'ticket da cozinha', label: 'etiqueta', receipt: 'recibo' };
      const failures = (['kitchen', 'label', 'receipt'] as const)
        .map((doc) => ({ doc, r: doc === 'receipt' ? receipt : printing[doc] }))
        .filter((f): f is { doc: 'kitchen' | 'label' | 'receipt'; r: { success: boolean; reason?: string } } => !!f.r && !f.r.success);

      if (failures.length > 0) {
        const detail = failures.map(({ doc, r }) => `${documentLabels[doc]}: ${r.reason}`).join(' · ');
        toast(`Pedido #${order.dailyNumber} salvo, mas não foi possível imprimir — ${detail}`, 'warning', 7000);
      } else {
        toast(`Pedido #${order.dailyNumber} enviado para a cozinha!`, 'success', 4000);
      }

      lastOrder = { id: order.id, dailyNumber: order.dailyNumber, paid: payment !== null };
      clearOrder();
      await loadData();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao enviar pedido.'), 'error', 5000);
    } finally {
      submitting = false;
    }
  }

  // --- Reimpressão rápida do último pedido enviado ---

  async function reprintLast(document: 'kitchen' | 'label' | 'receipt') {
    // Sem essa trava, um duplo clique (ou o F6 repetindo ao segurar a tecla)
    // manda dois jobs pra impressora física — dois tickets de cozinha, duas
    // etiquetas.
    if (!lastOrder || reprinting) return;
    reprinting = true;
    try {
      const result = await api.reprint(lastOrder.id, document);
      if (result.success) toast('Reimpressão enviada.', 'success', 3000);
      else toast(`Não foi possível reimprimir: ${result.reason}`, 'error', 5000);
    } catch (err) {
      toast(errorMessage(err, 'Erro ao reimprimir.'), 'error');
    } finally {
      reprinting = false;
    }
  }

  // --- Keyboard shortcuts ---

  function handleGlobalKeydown(ev: KeyboardEvent) {
    if (payingOpen || pendingClear) return;

    if (referenceModalOpen) {
      const activeTag = (document.activeElement as HTMLElement | null)?.tagName;
      if (ev.key === '1' && activeTag !== 'INPUT') {
        ev.preventDefault();
        selectModalSource('counter');
      } else if (ev.key === '2' && activeTag !== 'INPUT') {
        ev.preventDefault();
        selectModalSource('table');
      } else if (ev.key === 'Enter' && activeTag !== 'INPUT') {
        ev.preventDefault();
        confirmReference();
      }
      return;
    }

    if (ev.key === 'F2') {
      ev.preventDefault();
      openReferenceModal();
      return;
    }
    if (ev.key === 'F4') {
      ev.preventDefault();
      handlePaymentStart();
      return;
    }
    if (ev.key === 'F8') {
      ev.preventDefault();
      if (cart.length > 0) submitOrder(null);
      return;
    }
    if (ev.key === 'F9') {
      ev.preventDefault();
      handleClear();
      return;
    }
    if (ev.key === 'F6') {
      ev.preventDefault();
      reprintLast('kitchen');
      return;
    }

    if (ev.altKey && /^Digit[1-9]$/.test(ev.code)) {
      ev.preventDefault();
      const idx = Number(ev.code.slice(5)) - 1;
      if (tabs[idx]) selectCategory(tabs[idx].id);
      return;
    }

    if (ev.target === searchInputEl) {
      if (ev.key === 'ArrowDown') {
        ev.preventDefault();
        if (selectedIndex < filteredProducts.length - 1) selectedIndex += 1;
      } else if (ev.key === 'ArrowUp') {
        ev.preventDefault();
        if (selectedIndex > 0) selectedIndex -= 1;
      } else if (ev.key === 'Enter') {
        ev.preventDefault();
        const { quantity } = parseSearch(searchValue);
        const product = filteredProducts[selectedIndex];
        if (product) {
          addToCart(product, quantity);
          searchValue = '';
          selectedIndex = 0;
        }
      } else if (ev.key === 'Escape') {
        ev.preventDefault();
        searchValue = '';
        selectedIndex = 0;
      } else if (ev.key === '/' && searchValue === '') {
        ev.preventDefault();
        focusLastItemNote();
      }
    }
  }

  function handleGlobalClick(ev: MouseEvent) {
    if (referenceModalOpen || payingOpen || pendingClear) return;
    const tag = (ev.target as HTMLElement).tagName;
    if (tag === 'INPUT' || tag === 'BUTTON' || tag === 'TEXTAREA') return;
    focusSearch();
  }

  loadData();
</script>

<svelte:window onkeydown={handleGlobalKeydown} onclick={handleGlobalClick} />

<Toaster />
<Nav active="order" />

<div class="pl-52">
<div
  class="flex flex-wrap items-center gap-4 border-b border-dashed bg-card px-6 py-2 font-mono text-xs uppercase text-muted-foreground"
  role="status"
>
  {#if !registerInfo}
    <span class="font-semibold normal-case text-destructive">Caixa fechado — abra o caixa na tela "Caixa" antes de vender.</span>
  {:else}
    <span>Caixa aberto às {registerInfo.openedAt.slice(11, 16)}</span>
    {#if registerInfo.openTabs.count > 0}
      <Badge variant="warning">Mesas abertas: {registerInfo.openTabs.count} · {money(registerInfo.openTabs.total)}</Badge>
    {/if}
    <span>Vendas hoje: {money(registerInfo.totalSales)}</span>
  {/if}
  {#if lastOrder}
    <span class="ml-auto flex items-center gap-2 normal-case">
      <span>Último: #{lastOrder.dailyNumber}</span>
      <Button variant="ghost" size="sm" class="h-6 px-2 text-[11px]" disabled={reprinting} onclick={() => reprintLast('kitchen')}>
        Cozinha <kbd class="opacity-70">F6</kbd>
      </Button>
      <Button variant="ghost" size="sm" class="h-6 px-2 text-[11px]" disabled={reprinting} onclick={() => reprintLast('label')}>Etiqueta</Button>
      {#if lastOrder.paid}
        <Button variant="ghost" size="sm" class="h-6 px-2 text-[11px]" disabled={reprinting} onclick={() => reprintLast('receipt')}>Recibo</Button>
      {/if}
    </span>
  {/if}
</div>

<main id="main-content" class="mx-auto grid max-w-6xl gap-0 p-6 md:grid-cols-[1fr_320px]">
  <section class="pr-0 md:pr-6">
    <Label for="search-input" class="sr-only">Buscar produto</Label>
    <Input
      id="search-input"
      bind:ref={searchInputEl}
      bind:value={searchValue}
      oninput={onSearchInput}
      placeholder="&gt; digite o nome da sopa… (ex: 2 caldo verde)"
      autocomplete="off"
      class="mb-3 h-11 rounded-[3px] border-2 border-foreground font-mono text-[15px]"
    />

    <div class="mb-4 flex flex-wrap gap-1.5" role="tablist" aria-label="Categorias">
      {#each tabs as tab (tab.id)}
        <button
          type="button"
          role="tab"
          aria-selected={tab.id === currentCategory}
          class={cn(
            'rounded-[3px] border px-3.5 py-1.5 font-mono text-xs tracking-wide uppercase transition-colors outline-none focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:border-ring active:translate-y-px',
            tab.id === currentCategory
              ? 'border-foreground bg-foreground text-background'
              : 'border-border bg-card text-muted-foreground hover:bg-muted hover:text-foreground'
          )}
          onclick={() => selectCategory(tab.id)}
        >
          {tab.name}
        </button>
      {/each}
    </div>

    {#if loading}
      <div class="grid grid-cols-2 gap-px sm:grid-cols-3">
        {#each Array(6) as _}
          <div class="h-16 animate-pulse border border-border bg-muted"></div>
        {/each}
      </div>
    {:else if products.length === 0}
      <p class="text-muted-foreground">
        Nenhum produto disponível hoje. Vá em "Cardápio" para marcar itens como disponíveis.
      </p>
    {:else if filteredProducts.length === 0}
      <p class="text-muted-foreground">Nada encontrado para essa busca.</p>
    {:else}
      <div class="grid grid-cols-2 gap-px sm:grid-cols-3">
        {#each filteredProducts as product, index (product.id)}
          <button
            type="button"
            bind:this={productRefs[index]}
            class={cn(
              'relative flex flex-col items-start gap-1.5 border border-border bg-card p-3.5 text-left transition-colors outline-none hover:bg-accent focus-visible:z-10 focus-visible:ring-3 focus-visible:ring-ring/50 active:translate-y-px',
              index === selectedIndex && 'bg-accent shadow-[inset_3px_0_0_var(--color-primary)]'
            )}
            onclick={() => {
              addToCart(product, 1);
              focusSearch();
            }}
          >
            {#if topProductNames.has(product.name)}
              <Flame class="absolute top-2 right-2 size-3.5 text-primary" aria-label="Um dos mais pedidos hoje" />
            {/if}
            <span class="text-sm font-semibold">{product.name}</span>
            <span class="font-mono text-sm font-semibold tabular-nums text-muted-foreground">{money(product.price)}</span>
          </button>
        {/each}
      </div>
    {/if}
  </section>

  <section class="ticket-notch relative flex h-fit flex-col gap-3 border border-border bg-card p-4 md:[border-left-style:dashed]">
    <div class="flex items-center justify-between pt-1">
      <h2 class="font-mono text-sm font-bold tracking-wide uppercase">Pedido</h2>
      <Button variant="outline" size="sm" class="border-primary text-primary hover:bg-primary/5" onclick={openReferenceModal}>
        {referenceLabel} <kbd class="ml-1.5 opacity-70">F2</kbd>
      </Button>
    </div>

    <Label for="order-note-input" class="sr-only">Nota do pedido</Label>
    <Input
      id="order-note-input"
      bind:value={orderNote}
      placeholder="nota do pedido (opcional, ex: para viagem)"
      autocomplete="off"
      class="rounded-[3px] border-dashed font-mono text-xs"
    />

    {#if cart.length === 0}
      <p class="text-sm text-muted-foreground">Carrinho vazio. Digite o nome de uma sopa e aperte Enter.</p>
    {:else}
      <div class="flex flex-col gap-2.5">
        {#each cart as item, i (i)}
          <div class="border-b border-dotted border-border pb-2.5 last:border-0 last:pb-0">
            <div class="flex items-center justify-between font-mono text-[13px]">
              <span class="uppercase">{item.quantity}x {item.name}</span>
              <span class="font-bold tabular-nums">{money(item.unitPrice * item.quantity)}</span>
            </div>
            <div class="mt-1.5 flex items-center gap-1.5">
              <Label for={`item-note-${i}`} class="sr-only">Observação para {item.name}</Label>
              <Input
                id={`item-note-${i}`}
                bind:ref={noteInputs[i]}
                bind:value={item.note}
                placeholder="observação (ex: sem cebola)"
                class="h-8 rounded-[3px] border-dashed font-sans text-xs italic"
                onkeydown={(ev) => {
                  if (ev.key === 'Enter' || ev.key === 'Escape') {
                    (ev.target as HTMLInputElement).blur();
                    focusSearch();
                  }
                }}
              />
              <Button
                variant="outline"
                size="icon"
                class="h-8 w-8 shrink-0"
                aria-label={`Diminuir quantidade de ${item.name}`}
                onclick={() => changeQuantity(i, -1)}
              >
                −
              </Button>
              <Button
                variant="outline"
                size="icon"
                class="h-8 w-8 shrink-0"
                aria-label={`Aumentar quantidade de ${item.name}`}
                onclick={() => changeQuantity(i, 1)}
              >
                +
              </Button>
            </div>
          </div>
        {/each}
      </div>
    {/if}

    <div class="flex items-baseline justify-between border-t-2 border-foreground pt-2.5">
      <span class="font-mono text-xs uppercase tracking-wide text-muted-foreground">Total</span>
      <span class="font-mono text-2xl font-bold tabular-nums">{money(cartTotal)}</span>
    </div>

    <div class="flex flex-col gap-1.5 pt-1">
      <Button disabled={cart.length === 0 || submitting} onclick={handlePaymentStart} class="justify-between px-4">
        Pagar <kbd class="opacity-70">F4</kbd>
      </Button>
      <Button
        variant="outline"
        disabled={cart.length === 0 || submitting}
        onclick={() => submitOrder(null)}
        class="justify-between px-4"
      >
        Enviar sem pagar <kbd class="opacity-70">F8</kbd>
      </Button>
      <Button variant="ghost" onclick={handleClear} class="justify-between px-4">
        Limpar <kbd class="opacity-70">F9</kbd>
      </Button>
    </div>
  </section>
</main>

<footer class="flex flex-wrap gap-x-4 gap-y-1 border-t border-dashed bg-card px-6 py-2 font-mono text-[11px] uppercase tracking-wide text-muted-foreground">
  <span><b>↑↓</b> navegar</span>
  <span><b>Enter</b> adicionar</span>
  <span><b>3 nome</b> quantidade</span>
  <span><b>/</b> observação do item</span>
  <span><b>Alt+1-9</b> categoria</span>
  <span><b>F2</b> identificação</span>
  <span><b>F4</b> pagar</span>
  <span><b>F6</b> reimprimir último</span>
  <span><b>F8</b> enviar sem pagar</span>
  <span><b>F9</b> limpar</span>
  <span><b>Esc</b> limpar busca</span>
</footer>
</div>

<Dialog.Root open={referenceModalOpen} onOpenChange={(open) => { referenceModalOpen = open; if (!open) focusSearch(); }}>
  <Dialog.Content
    onOpenAutoFocus={(ev) => {
      ev.preventDefault();
      referenceExtraInputEl?.focus();
    }}
  >
    <Dialog.Header>
      <Dialog.Title>Identificação do pedido</Dialog.Title>
    </Dialog.Header>
    <div class="grid grid-cols-2 gap-2">
      <Button
        variant={modalSource === 'counter' ? 'default' : 'secondary'}
        aria-pressed={modalSource === 'counter'}
        onclick={() => selectModalSource('counter')}
      >
        Balcão <kbd class="ml-1.5 opacity-70">1</kbd>
      </Button>
      <Button
        variant={modalSource === 'table' ? 'default' : 'secondary'}
        aria-pressed={modalSource === 'table'}
        onclick={() => selectModalSource('table')}
      >
        Mesa <kbd class="ml-1.5 opacity-70">2</kbd>
      </Button>
    </div>
    {#if modalSource === 'table'}
      <Label for="table-number-input" class="sr-only">Número da mesa</Label>
      <Input
        id="table-number-input"
        bind:ref={referenceExtraInputEl}
        bind:value={modalValue}
        placeholder="Número da mesa"
        onkeydown={(ev) => {
          if (ev.key === 'Enter') {
            ev.preventDefault();
            confirmReference();
          }
        }}
      />
    {:else}
      <Label for="customer-name-input" class="sr-only">Nome do cliente</Label>
      <Input
        id="customer-name-input"
        bind:ref={referenceExtraInputEl}
        bind:value={modalValue}
        placeholder="Nome do cliente (opcional)"
        onkeydown={(ev) => {
          if (ev.key === 'Enter') {
            ev.preventDefault();
            confirmReference();
          }
        }}
      />
    {/if}
    <Dialog.Footer>
      <Button onclick={confirmReference}>Confirmar <kbd class="ml-1.5 opacity-70">Enter</kbd></Button>
      <Button variant="secondary" onclick={() => { referenceModalOpen = false; focusSearch(); }}>Cancelar <kbd class="ml-1.5 opacity-70">Esc</kbd></Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root open={payingOpen} onOpenChange={(open) => { payingOpen = open; if (!open) focusSearch(); }}>
  <Dialog.Content escapeKeydownBehavior="ignore">
    {#if payingOpen}
      <PaymentDialogContent
        total={cartTotal}
        items={cart.map((i) => ({ quantity: i.quantity, name: i.name }))}
        onConfirm={paymentConfirmed}
        onCancel={() => { payingOpen = false; focusSearch(); }}
      />
    {/if}
  </Dialog.Content>
</Dialog.Root>

<AlertDialog.Root open={pendingClear} onOpenChange={(open) => { pendingClear = open; if (!open) focusSearch(); }}>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>Limpar pedido?</AlertDialog.Title>
      <AlertDialog.Description>O pedido atual será apagado, incluindo os itens já adicionados.</AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>Cancelar</AlertDialog.Cancel>
      <AlertDialog.Action variant="destructive" onclick={clearOrder}>
        Limpar
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>

<style>
  .ticket-notch::before {
    content: '';
    position: absolute;
    top: -1px;
    left: 0;
    right: 0;
    height: 8px;
    background-image: radial-gradient(circle at 8px 0, transparent 4px, var(--background) 4.5px);
    background-size: 16px 8px;
    background-repeat: repeat-x;
  }
</style>
