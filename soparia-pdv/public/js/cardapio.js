const nomeEl = document.getElementById('nome');
const precoEl = document.getElementById('preco');
const btnAdicionar = document.getElementById('btn-adicionar');
const listaEl = document.getElementById('lista-sopas');
const avisoEl = document.getElementById('aviso');

function formatarPreco(valor) {
  return valor.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

function mostrarAviso(mensagem, tipo) {
  avisoEl.innerHTML = `<div class="aviso ${tipo}">${mensagem}</div>`;
  setTimeout(() => (avisoEl.innerHTML = ''), 4000);
}

async function carregarSopas() {
  const resposta = await fetch('/api/sopas');
  const sopas = await resposta.json();
  renderizarLista(sopas);
}

function renderizarLista(sopas) {
  if (sopas.length === 0) {
    listaEl.innerHTML = '<p>Nenhuma sopa cadastrada ainda.</p>';
    return;
  }

  listaEl.innerHTML = sopas
    .map(
      (sopa) => `
      <li>
        <span>${sopa.nome} — ${formatarPreco(sopa.preco)}</span>
        <button class="secundario" data-id="${sopa.id}">Remover</button>
      </li>
    `
    )
    .join('');

  listaEl.querySelectorAll('button').forEach((botao) => {
    botao.addEventListener('click', () => removerSopa(Number(botao.dataset.id)));
  });
}

async function adicionarSopa() {
  const nome = nomeEl.value.trim();
  const preco = Number(precoEl.value);

  if (!nome || !Number.isFinite(preco) || preco <= 0) {
    mostrarAviso('Preencha nome e preço válidos.', 'erro');
    return;
  }

  const resposta = await fetch('/api/sopas', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ nome, preco }),
  });

  if (!resposta.ok) {
    const dados = await resposta.json();
    mostrarAviso(dados.erro || 'Erro ao adicionar sopa.', 'erro');
    return;
  }

  nomeEl.value = '';
  precoEl.value = '';
  mostrarAviso('Sopa adicionada!', 'sucesso');
  carregarSopas();
}

async function removerSopa(id) {
  if (!confirm('Remover esta sopa do cardápio?')) return;

  const resposta = await fetch(`/api/sopas/${id}`, { method: 'DELETE' });
  if (!resposta.ok) {
    mostrarAviso('Erro ao remover sopa.', 'erro');
    return;
  }
  carregarSopas();
}

btnAdicionar.addEventListener('click', adicionarSopa);

carregarSopas();
