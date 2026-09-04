const productNameInput = document.getElementById('product-name-input');
const productPriceInput = document.getElementById('product-price-input');
const categorySelectEl = document.getElementById('category-select');
const addProductButton = document.getElementById('add-product-button');
const categoryNameInput = document.getElementById('category-name-input');
const addCategoryButton = document.getElementById('add-category-button');
const toggleInactiveButton = document.getElementById('toggle-inactive-button');
const listEl = document.getElementById('product-list');

let categories = [];
let showingInactive = false;

async function loadCategories() {
  categories = await fetchJSON('/api/categories');
  categorySelectEl.innerHTML =
    '<option value="">Sem categoria</option>' +
    categories.map((c) => `<option value="${c.id}">${escapeHtml(c.name)}</option>`).join('');
}

async function loadProducts() {
  listEl.innerHTML = skeletonRows(4);
  const qs = showingInactive ? '?includeInactive=1' : '';
  const products = await fetchJSON(`/api/products${qs}`);
  renderList(products);
}

function categoryName(id) {
  const category = categories.find((c) => c.id === id);
  return category ? category.name : 'Sem categoria';
}

function renderList(products) {
  if (products.length === 0) {
    listEl.innerHTML = '<p class="empty-state">Nenhum produto cadastrado ainda.</p>';
    return;
  }

  listEl.innerHTML = products
    .map(
      (product) => `
      <li class="${product.active ? '' : 'inactive'}">
        <div class="product-info">
          <span class="product-name">${escapeHtml(product.name)}</span>
          <span class="product-category">${escapeHtml(categoryName(product.categoryId))}</span>
          <span class="product-price">${formatCurrency(product.price)}</span>
        </div>
        <div class="product-actions">
          ${
            product.active
              ? `<label class="availability-toggle">
                   <input type="checkbox" data-id="${product.id}" class="today-toggle" ${product.availableToday ? 'checked' : ''} />
                   Disponível hoje
                 </label>
                 <button class="secondary" data-action="edit" data-id="${product.id}">Editar</button>
                 <button class="secondary danger" data-action="remove" data-id="${product.id}">Remover</button>`
              : `<button class="secondary" data-action="reactivate" data-id="${product.id}">Reativar</button>`
          }
        </div>
      </li>
    `
    )
    .join('');

  listEl.querySelectorAll('.today-toggle').forEach((checkbox) => {
    checkbox.addEventListener('change', () => toggleAvailability(Number(checkbox.dataset.id), checkbox.checked));
  });
  listEl.querySelectorAll('button[data-action]').forEach((button) => {
    button.addEventListener('click', () => handleAction(button.dataset.action, Number(button.dataset.id)));
  });
}

async function addCategory() {
  const name = categoryNameInput.value.trim();
  if (!name) {
    showToast('Informe o nome da categoria.', 'error', 4000);
    return;
  }
  try {
    await fetchJSON('/api/categories', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });
    categoryNameInput.value = '';
    showToast('Categoria adicionada!', 'success', 3000);
    await loadCategories();
  } catch (err) {
    showToast(err.message || 'Erro ao adicionar categoria.', 'error', 4000);
  }
}

async function addProduct() {
  const name = productNameInput.value.trim();
  const price = Number(productPriceInput.value);
  const categoryId = categorySelectEl.value || null;

  if (!name || !Number.isFinite(price) || price <= 0) {
    showToast('Preencha nome e preço válidos.', 'error', 4000);
    return;
  }

  try {
    await fetchJSON('/api/products', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, price, categoryId }),
    });
    productNameInput.value = '';
    productPriceInput.value = '';
    showToast('Produto adicionado!', 'success', 3000);
    loadProducts();
  } catch (err) {
    showToast(err.message || 'Erro ao adicionar produto.', 'error', 4000);
  }
}

async function toggleAvailability(id, available) {
  try {
    await fetchJSON(`/api/products/${id}/availability`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ availableToday: available }),
    });
    showToast(available ? 'Marcado como disponível hoje.' : 'Marcado como indisponível hoje.', 'success', 2500);
  } catch (err) {
    showToast('Erro ao atualizar disponibilidade.', 'error', 4000);
    loadProducts();
  }
}

async function handleAction(action, id) {
  try {
    if (action === 'remove') {
      if (!confirm('Remover este produto do cardápio?')) return;
      await fetchJSON(`/api/products/${id}`, { method: 'DELETE' });
      showToast('Produto removido.', 'success', 3000);
    } else if (action === 'reactivate') {
      await fetchJSON(`/api/products/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ active: true }),
      });
      showToast('Produto reativado.', 'success', 3000);
    } else if (action === 'edit') {
      await editProduct(id);
    }
    loadProducts();
  } catch (err) {
    showToast(err.message || 'Erro ao executar ação.', 'error', 4000);
  }
}

async function editProduct(id) {
  const newName = prompt('Novo nome:');
  if (newName === null) return;
  const newPrice = prompt('Novo preço:');
  if (newPrice === null) return;

  await fetchJSON(`/api/products/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: newName, price: Number(newPrice) }),
  });
  showToast('Produto atualizado.', 'success', 3000);
}

addCategoryButton.addEventListener('click', addCategory);
addProductButton.addEventListener('click', addProduct);
toggleInactiveButton.addEventListener('click', () => {
  showingInactive = !showingInactive;
  toggleInactiveButton.textContent = showingInactive ? 'Ver só ativos' : 'Ver removidos';
  loadProducts();
});

(async function init() {
  await loadCategories();
  await loadProducts();
})();
