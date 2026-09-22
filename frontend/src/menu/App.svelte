<script lang="ts">
  import { api, ApiError, type Category, type Product } from '$lib/api.js';
  import { toast } from '$lib/toast.svelte.js';
  import Toaster from '$lib/components/Toaster.svelte';
  import Nav from '$lib/components/Nav.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Switch } from '$lib/components/ui/switch/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';

  let categories = $state<Category[]>([]);
  let products = $state<Product[]>([]);
  let loading = $state(true);
  let showingInactive = $state(false);
  let busy = $state(false);

  let categoryName = $state('');
  let productName = $state('');
  let productPrice = $state('');
  let productCategoryId = $state<number | ''>('');

  let editingProduct = $state<Product | null>(null);
  let editName = $state('');
  let editPrice = $state('');
  let editCategoryId = $state<number | ''>('');

  let removingProduct = $state<Product | null>(null);

  let editingCategory = $state<Category | null>(null);
  let editCategoryName = $state('');

  let removingCategory = $state<Category | null>(null);

  let sortedCategories = $derived([...categories].sort((a, b) => a.sortOrder - b.sortOrder));

  function categoryName_(id: number | null) {
    const category = categories.find((c) => c.id === id);
    return category ? category.name : 'Sem categoria';
  }

  function errorMessage(err: unknown, fallback: string) {
    return err instanceof ApiError ? err.message : fallback;
  }

  async function loadCategories() {
    categories = await api.listCategories();
  }

  async function loadProducts() {
    loading = true;
    try {
      products = await api.listProducts({ includeInactive: showingInactive });
    } finally {
      loading = false;
    }
  }

  async function addCategory() {
    if (busy) return;
    const name = categoryName.trim();
    if (!name) {
      toast('Informe o nome da categoria.', 'error');
      return;
    }
    busy = true;
    try {
      await api.createCategory(name);
      categoryName = '';
      toast('Categoria adicionada!', 'success');
      await loadCategories();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao adicionar categoria.'), 'error');
    } finally {
      busy = false;
    }
  }

  function openEditCategory(category: Category) {
    editingCategory = category;
    editCategoryName = category.name;
  }

  async function submitEditCategory() {
    if (!editingCategory || busy) return;
    const name = editCategoryName.trim();
    if (!name) {
      toast('Informe o nome da categoria.', 'error');
      return;
    }
    busy = true;
    try {
      await api.updateCategory(editingCategory.id, { name });
      toast('Categoria atualizada.', 'success');
      editingCategory = null;
      await loadCategories();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao atualizar categoria.'), 'error');
    } finally {
      busy = false;
    }
  }

  async function moveCategory(category: Category, direction: -1 | 1) {
    const sorted = [...categories].sort((a, b) => a.sortOrder - b.sortOrder);
    const index = sorted.findIndex((c) => c.id === category.id);
    const swapIndex = index + direction;
    if (index === -1 || swapIndex < 0 || swapIndex >= sorted.length) return;
    // Reindexa tudo pela posição (0..n-1) em vez de trocar os valores de
    // sortOrder — categorias antigas podem ter o mesmo sortOrder (padrão da
    // migração), e nesse caso trocar os valores não muda nada.
    const reordered = [...sorted];
    [reordered[index], reordered[swapIndex]] = [reordered[swapIndex], reordered[index]];
    const updates = reordered
      .map((c, newIndex) => ({ id: c.id, sortOrder: newIndex, changed: c.sortOrder !== newIndex }))
      .filter((u) => u.changed);
    try {
      await Promise.all(updates.map((u) => api.updateCategory(u.id, { sortOrder: u.sortOrder })));
      await loadCategories();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao reordenar categoria.'), 'error');
    }
  }

  async function confirmRemoveCategory() {
    if (!removingCategory || busy) return;
    busy = true;
    try {
      await api.removeCategory(removingCategory.id);
      toast('Categoria removida.', 'success');
      removingCategory = null;
      await loadCategories();
      await loadProducts();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao remover categoria.'), 'error');
    } finally {
      busy = false;
    }
  }

  async function addProduct() {
    if (busy) return;
    const name = productName.trim();
    const price = Number(productPrice);
    if (!name || !Number.isFinite(price) || price <= 0) {
      toast('Preencha nome e preço válidos.', 'error');
      return;
    }
    busy = true;
    try {
      await api.createProduct({ name, price, categoryId: productCategoryId === '' ? null : productCategoryId });
      productName = '';
      productPrice = '';
      productCategoryId = '';
      toast('Produto adicionado!', 'success');
      await loadProducts();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao adicionar produto.'), 'error');
    } finally {
      busy = false;
    }
  }

  async function toggleAvailability(product: Product, available: boolean) {
    try {
      await api.setAvailability(product.id, available);
      product.availableToday = available;
      toast(available ? 'Marcado como disponível hoje.' : 'Marcado como indisponível hoje.', 'success', 2500);
    } catch {
      toast('Erro ao atualizar disponibilidade.', 'error');
      await loadProducts();
    }
  }

  function openEdit(product: Product) {
    editingProduct = product;
    editName = product.name;
    editPrice = String(product.price);
    editCategoryId = product.categoryId ?? '';
  }

  async function submitEdit() {
    if (!editingProduct || busy) return;
    const name = editName.trim();
    const price = Number(editPrice);
    if (!name || !Number.isFinite(price) || price <= 0) {
      toast('Preencha nome e preço válidos.', 'error');
      return;
    }
    busy = true;
    try {
      await api.updateProduct(editingProduct.id, {
        name,
        price,
        categoryId: editCategoryId === '' ? null : editCategoryId,
      });
      toast('Produto atualizado.', 'success');
      editingProduct = null;
      await loadProducts();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao atualizar produto.'), 'error');
    } finally {
      busy = false;
    }
  }

  async function confirmRemove() {
    if (!removingProduct || busy) return;
    busy = true;
    try {
      await api.removeProduct(removingProduct.id);
      toast('Produto removido.', 'success');
      removingProduct = null;
      await loadProducts();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao remover produto.'), 'error');
    } finally {
      busy = false;
    }
  }

  async function reactivate(product: Product) {
    try {
      await api.updateProduct(product.id, { active: true });
      toast('Produto reativado.', 'success');
      await loadProducts();
    } catch (err) {
      toast(errorMessage(err, 'Erro ao reativar produto.'), 'error');
    }
  }

  async function init() {
    await loadCategories();
    await loadProducts();
  }
  init();
</script>

<Toaster />
<Nav active="menu" />

<div class="pl-52">
  <main id="main-content" class="mx-auto max-w-3xl p-6">
  <h1 class="mb-4 text-2xl font-bold">Cardápio</h1>

  <div class="mb-6 max-w-md rounded-lg border bg-card p-4 shadow-sm">
    <h2 class="mb-3 text-lg font-semibold">Nova categoria</h2>
    <Label for="category-name-input" class="sr-only">Nome da categoria</Label>
    <Input id="category-name-input" placeholder="Ex: Sopas, Bebidas, Porções" bind:value={categoryName} class="mb-3" />
    <Button variant="secondary" onclick={addCategory} disabled={busy}>Adicionar categoria</Button>

    {#if sortedCategories.length > 0}
      <ul class="mt-4 flex flex-col gap-2">
        {#each sortedCategories as category, index (category.id)}
          <li class="flex items-center justify-between gap-2 rounded-md border bg-background px-3 py-2">
            <span class="text-sm font-medium">{category.name}</span>
            <div class="flex items-center gap-1">
              <Button
                variant="outline"
                size="sm"
                disabled={index === 0}
                onclick={() => moveCategory(category, -1)}
                aria-label="Mover para cima"
              >
                ↑
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={index === sortedCategories.length - 1}
                onclick={() => moveCategory(category, 1)}
                aria-label="Mover para baixo"
              >
                ↓
              </Button>
              <Button variant="secondary" size="sm" onclick={() => openEditCategory(category)}>Renomear</Button>
              <Button variant="outline" size="sm" class="text-destructive" onclick={() => (removingCategory = category)}>
                Remover
              </Button>
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  <div class="mb-6 max-w-md rounded-lg border bg-card p-4 shadow-sm">
    <h2 class="mb-3 text-lg font-semibold">Novo produto</h2>
    <Label for="product-name-input" class="sr-only">Nome do produto</Label>
    <Input id="product-name-input" placeholder="Nome do produto" bind:value={productName} class="mb-3" />
    <Label for="product-price-input" class="sr-only">Preço</Label>
    <Input
      id="product-price-input"
      type="number"
      step="0.01"
      min="0"
      placeholder="Preço (ex: 18.90)"
      bind:value={productPrice}
      class="mb-3"
    />
    <Label for="category-select" class="sr-only">Categoria do produto</Label>
    <select
      id="category-select"
      bind:value={productCategoryId}
      class="border-input mb-3 flex h-9 w-full rounded-md border bg-transparent px-3 py-1 text-sm shadow-sm"
    >
      <option value="">Sem categoria</option>
      {#each categories as category (category.id)}
        <option value={category.id}>{category.name}</option>
      {/each}
    </select>
    <Button onclick={addProduct} disabled={busy}>Adicionar ao cardápio</Button>
  </div>

  <div class="mb-3 flex items-center justify-between">
    <h2 class="text-lg font-semibold">Produtos</h2>
    <Button
      variant="secondary"
      onclick={() => {
        showingInactive = !showingInactive;
        loadProducts();
      }}
    >
      {showingInactive ? 'Ver só ativos' : 'Ver removidos'}
    </Button>
  </div>

  {#if loading}
    <div class="flex flex-col gap-2">
      {#each Array(4) as _}
        <div class="h-14 animate-pulse rounded-lg bg-muted"></div>
      {/each}
    </div>
  {:else if products.length === 0}
    <p class="text-muted-foreground">Nenhum produto cadastrado ainda.</p>
  {:else}
    <ul class="flex flex-col gap-2">
      {#each products as product (product.id)}
        <li class={`flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-card p-4 shadow-sm ${!product.active ? 'opacity-60' : ''}`}>
          <div class="flex items-baseline gap-3">
            <span class="font-semibold">{product.name}</span>
            {#if !product.active}
              <Badge variant="secondary">Removido</Badge>
            {/if}
            <span class="text-sm text-muted-foreground">{categoryName_(product.categoryId)}</span>
            <span class="tabular-nums text-muted-foreground">
              {product.price.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}
            </span>
          </div>
          <div class="flex items-center gap-3">
            {#if product.active}
              <label class="flex items-center gap-2 text-sm">
                <Switch
                  checked={product.availableToday}
                  onCheckedChange={(value: boolean) => toggleAvailability(product, value)}
                />
                Disponível hoje
              </label>
              <Button variant="secondary" size="sm" onclick={() => openEdit(product)}>Editar</Button>
              <Button variant="outline" size="sm" class="text-destructive" onclick={() => (removingProduct = product)}>
                Remover
              </Button>
            {:else}
              <Button variant="secondary" size="sm" onclick={() => reactivate(product)}>Reativar</Button>
            {/if}
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</main>
</div>

<Dialog.Root open={editingProduct !== null} onOpenChange={(open) => !open && (editingProduct = null)}>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Editar produto</Dialog.Title>
    </Dialog.Header>
    <div class="flex flex-col gap-3">
      <Label for="edit-name" class="sr-only">Nome do produto</Label>
      <Input id="edit-name" placeholder="Nome do produto" bind:value={editName} />
      <Label for="edit-price" class="sr-only">Preço</Label>
      <Input id="edit-price" type="number" step="0.01" min="0" placeholder="Preço" bind:value={editPrice} />
      <Label for="edit-category" class="sr-only">Categoria</Label>
      <select
        id="edit-category"
        bind:value={editCategoryId}
        class="border-input flex h-9 w-full rounded-md border bg-transparent px-3 py-1 text-sm shadow-sm"
      >
        <option value="">Sem categoria</option>
        {#each categories as category (category.id)}
          <option value={category.id}>{category.name}</option>
        {/each}
      </select>
    </div>
    <Dialog.Footer>
      <Button variant="secondary" onclick={() => (editingProduct = null)}>Cancelar</Button>
      <Button onclick={submitEdit} disabled={busy}>Salvar</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root open={editingCategory !== null} onOpenChange={(open) => !open && (editingCategory = null)}>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Renomear categoria</Dialog.Title>
    </Dialog.Header>
    <div class="flex flex-col gap-3">
      <Label for="edit-category-name" class="sr-only">Nome da categoria</Label>
      <Input id="edit-category-name" placeholder="Nome da categoria" bind:value={editCategoryName} />
    </div>
    <Dialog.Footer>
      <Button variant="secondary" onclick={() => (editingCategory = null)}>Cancelar</Button>
      <Button onclick={submitEditCategory} disabled={busy}>Salvar</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<AlertDialog.Root open={removingCategory !== null} onOpenChange={(open) => !open && (removingCategory = null)}>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>Remover categoria?</AlertDialog.Title>
      <AlertDialog.Description>
        A categoria sai do cardápio. Produtos que estavam nela passam a aparecer como "Sem categoria".
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>Cancelar</AlertDialog.Cancel>
      <AlertDialog.Action variant="destructive" disabled={busy} onclick={confirmRemoveCategory}>
        Remover
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>

<AlertDialog.Root open={removingProduct !== null} onOpenChange={(open) => !open && (removingProduct = null)}>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>Remover produto?</AlertDialog.Title>
      <AlertDialog.Description>
        O produto sai do cardápio, mas continua no histórico — dá pra reativar depois em "Ver removidos".
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>Cancelar</AlertDialog.Cancel>
      <AlertDialog.Action variant="destructive" disabled={busy} onclick={confirmRemove}>
        Remover
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
