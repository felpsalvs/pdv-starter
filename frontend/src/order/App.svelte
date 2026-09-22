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
      lastItemIndex = cart.length - 1;
    }
  }

  function changeQuantity(index: number, delta: number) {
    const item = cart[index];
    if (!item) return;
    item.quantity += delta;
    if (item.quantity <= 0) {
      cart.splice(index, 1);
      lastItemIndex = cart.length - 1;
    }
  }

  function focusLastItemNote() {
    if (lastItemIndex < 0) return;
    noteInputs[lastItemIndex]?.focus();
  }

  function clearOrder() {
    cart = [];
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

      clearOrder();
      await loadData();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao enviar pedido.'), 'error', 5000);
    } finally {
      submitting = false;
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

<div class="flex flex-wrap items-center gap-3 border-b bg-muted/30 px-4 py-2 text-sm" role="status">
  {#if !registerInfo}
    <span class="font-semibold text-destructive">Caixa fechado — abra o caixa na tela "Caixa" antes de vender.</span>
  {:else}
    <span>Caixa aberto às {registerInfo.openedAt.slice(11, 16)}</span>
    {#if registerInfo.openTabs.count > 0}
      <Badge variant="warning">Contas abertas: {registerInfo.openTabs.count} ({money(registerInfo.openTabs.total)})</Badge>
    {/if}
    <span>Vendas hoje: {money(registerInfo.totalSales)}</span>
  {/if}
</div>

<main id="main-content" class="mx-auto grid max-w-6xl gap-4 p-4 md:grid-cols-[1fr_360px]">
  <section>
    <Label for="search-input" class="sr-only">Buscar produto</Label>
    <Input
      id="search-input"
      bind:ref={searchInputEl}
      bind:value={searchValue}
      oninput={onSearchInput}
      placeholder="Digite o nome da sopa… (ex: 2 caldo verde)"
      autocomplete="off"
      class="mb-3 h-11 text-base"
    />

    <div class="mb-3 flex flex-wrap gap-2" role="tablist" aria-label="Categorias">
      {#each tabs as tab (tab.id)}
        <button
          type="button"
          role="tab"
          aria-selected={tab.id === currentCategory}
          class={`rounded-full border px-4 py-1.5 text-sm ${tab.id === currentCategory ? 'bg-primary text-primary-foreground' : 'bg-secondary'}`}
          onclick={() => selectCategory(tab.id)}
        >
          {tab.name}
        </button>
      {/each}
    </div>

    {#if loading}
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
        {#each Array(6) as _}
          <div class="h-16 animate-pulse rounded-lg bg-muted"></div>
        {/each}
      </div>
    {:else if products.length === 0}
      <p class="text-muted-foreground">
        Nenhum produto disponível hoje. Vá em "Cardápio" para marcar itens como disponíveis.
      </p>
    {:else if filteredProducts.length === 0}
      <p class="text-muted-foreground">Nada encontrado para essa busca.</p>
    {:else}
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
        {#each filteredProducts as product, index (product.id)}
          <button
            type="button"
            bind:this={productRefs[index]}
            class={`flex flex-col items-start gap-1 rounded-lg border p-3 text-left transition-colors hover:bg-accent ${index === selectedIndex ? 'border-primary bg-primary/5 ring-1 ring-primary' : 'bg-card'}`}
            onclick={() => {
              addToCart(product, 1);
              focusSearch();
            }}
          >
            <span class="font-semibold">{product.name}</span>
            <span class="text-sm tabular-nums text-muted-foreground">{money(product.price)}</span>
          </button>
        {/each}
      </div>
    {/if}
  </section>

  <section class="sticky top-4 flex h-fit flex-col gap-3 rounded-lg border bg-card p-4 shadow-sm">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold">Pedido</h2>
      <Button variant="secondary" size="sm" onclick={openReferenceModal}>
        {referenceLabel} <kbd class="ml-1.5 opacity-70">F2</kbd>
      </Button>
    </div>

    <Label for="order-note-input" class="sr-only">Nota do pedido</Label>
    <Input id="order-note-input" bind:value={orderNote} placeholder="Nota do pedido (opcional, ex: para viagem)" autocomplete="off" />

    {#if cart.length === 0}
      <p class="text-sm text-muted-foreground">Carrinho vazio. Digite o nome de uma sopa e aperte Enter.</p>
    {:else}
      <div class="flex flex-col gap-2">
        {#each cart as item, i (i)}
          <div class="rounded-md border p-2">
            <div class="flex items-center justify-between text-sm">
              <span>{item.quantity}x {item.name}</span>
              <span class="tabular-nums">{money(item.unitPrice * item.quantity)}</span>
            </div>
            <div class="mt-1 flex items-center gap-1.5">
              <Label for={`item-note-${i}`} class="sr-only">Observação para {item.name}</Label>
              <Input
                id={`item-note-${i}`}
                bind:ref={noteInputs[i]}
                bind:value={item.note}
                placeholder="observação (ex: sem cebola)"
                class="h-8 text-xs"
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

    <div class="flex justify-between border-t pt-2 text-lg font-bold">
      <span>Total</span>
      <span class="tabular-nums">{money(cartTotal)}</span>
    </div>

    <div class="flex flex-col gap-2">
      <Button disabled={cart.length === 0 || submitting} onclick={handlePaymentStart}>
        Pagar <kbd class="ml-1.5 opacity-70">(F4)</kbd>
      </Button>
      <Button variant="secondary" disabled={cart.length === 0 || submitting} onclick={() => submitOrder(null)}>
        Enviar sem pagar <kbd class="ml-1.5 opacity-70">(F8)</kbd>
      </Button>
      <Button variant="secondary" onclick={handleClear}>Limpar <kbd class="ml-1.5 opacity-70">(F9)</kbd></Button>
    </div>
  </section>
</main>

<footer class="flex flex-wrap gap-x-4 gap-y-1 border-t bg-muted/30 px-4 py-2 text-xs text-muted-foreground">
  <span><b>↑↓</b> navegar</span>
  <span><b>Enter</b> adicionar</span>
  <span><b>3 nome</b> quantidade</span>
  <span><b>/</b> observação do item</span>
  <span><b>F2</b> identificação</span>
  <span><b>F4</b> pagar</span>
  <span><b>F8</b> enviar sem pagar</span>
  <span><b>F9</b> limpar</span>
  <span><b>Esc</b> limpar busca</span>
</footer>

<Dialog.Root open={referenceModalOpen} onOpenChange={(open) => (referenceModalOpen = open)}>
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
        Balcão <kbd class="ml-1.5 rounded border px-1 text-xs opacity-70">1</kbd>
      </Button>
      <Button
        variant={modalSource === 'table' ? 'default' : 'secondary'}
        aria-pressed={modalSource === 'table'}
        onclick={() => selectModalSource('table')}
      >
        Mesa <kbd class="ml-1.5 rounded border px-1 text-xs opacity-70">2</kbd>
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
      <Button onclick={confirmReference}>Confirmar <kbd class="ml-1.5 opacity-70">(Enter)</kbd></Button>
      <Button variant="secondary" onclick={() => (referenceModalOpen = false)}>Cancelar <kbd class="ml-1.5 opacity-70">(Esc)</kbd></Button>
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
      <AlertDialog.Action class="bg-destructive text-destructive-foreground hover:bg-destructive/90" onclick={clearOrder}>
        Limpar
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
