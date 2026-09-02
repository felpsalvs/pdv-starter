const sopasEl = document.getElementById('sopas');
const itensCarrinhoEl = document.getElementById('itens-carrinho');
const totalEl = document.getElementById('total');
const observacaoEl = document.getElementById('observacao');
const btnConfirmar = document.getElementById('btn-confirmar');
const avisoEl = document.getElementById('aviso');

let sopas = [];
let carrinho = [];

function formatarPreco(valor) {
  return valor.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

function mostrarAviso(mensagem, tipo) {
  avisoEl.innerHTML = `<div class="aviso ${tipo}">${mensagem}</div>`;
}

function limparAviso() {
  avisoEl.innerHTML = '';
}

async function carregarSopas() {
  const resposta = await fetch('/api/sopas');
  sopas = await resposta.json();
  renderizarSopas();
}

function renderizarSopas() {
  if (sopas.length === 0) {
    sopasEl.innerHTML = '<p>Nenhuma sopa cadastrada ainda. Vá em "Cardápio" para adicionar.</p>';
    return;
  }
  sopasEl.innerHTML = sopas
    .map(
      (sopa) => `
      <button class="cartao-sopa" data-id="${sopa.id}">
        <span class="nome">${sopa.nome}</span>
        <span class="preco">${formatarPreco(sopa.preco)}</span>
      </button>
    `
    )
    .join('');

  sopasEl.querySelectorAll('.cartao-sopa').forEach((botao) => {
    botao.addEventListener('click', () => adicionarAoCarrinho(Number(botao.dataset.id)));
  });
}

function adicionarAoCarrinho(sopaId) {
  const sopa = sopas.find((s) => s.id === sopaId);
  const item = carrinho.find((i) => i.sopaId === sopaId);
  if (item) {
    item.quantidade += 1;
  } else {
    carrinho.push({ sopaId: sopa.id, nome: sopa.nome, preco: sopa.preco, quantidade: 1 });
  }
  renderizarCarrinho();
}

function alterarQuantidade(sopaId, delta) {
  const item = carrinho.find((i) => i.sopaId === sopaId);
  if (!item) return;
  item.quantidade += delta;
  if (item.quantidade <= 0) {
    carrinho = carrinho.filter((i) => i.sopaId !== sopaId);
  }
  renderizarCarrinho();
}

function calcularTotal() {
  return carrinho.reduce((soma, item) => soma + item.preco * item.quantidade, 0);
}

function renderizarCarrinho() {
  if (carrinho.length === 0) {
    itensCarrinhoEl.innerHTML = '<p>Carrinho vazio. Toque numa sopa para adicionar.</p>';
  } else {
    itensCarrinhoEl.innerHTML = carrinho
      .map(
        (item) => `
        <div class="item-carrinho">
          <span>${item.quantidade}x ${item.nome} — ${formatarPreco(item.preco * item.quantidade)}</span>
          <span>
            <button data-id="${item.sopaId}" data-delta="-1">-</button>
            <button data-id="${item.sopaId}" data-delta="1">+</button>
          </span>
        </div>
      `
      )
      .join('');

    itensCarrinhoEl.querySelectorAll('button').forEach((botao) => {
      botao.addEventListener('click', () =>
        alterarQuantidade(Number(botao.dataset.id), Number(botao.dataset.delta))
      );
    });
  }

  totalEl.textContent = formatarPreco(calcularTotal());
  btnConfirmar.disabled = carrinho.length === 0;
}

async function confirmarPedido() {
  btnConfirmar.disabled = true;
  limparAviso();

  const itens = carrinho.map((item) => ({ sopaId: item.sopaId, quantidade: item.quantidade }));

  try {
    const resposta = await fetch('/api/pedidos', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ itens, observacao: observacaoEl.value.trim() || null }),
    });

    const dados = await resposta.json();

    if (!resposta.ok) {
      mostrarAviso(dados.erro || 'Erro ao enviar pedido.', 'erro');
      btnConfirmar.disabled = false;
      return;
    }

    if (dados.impressao.sucesso) {
      mostrarAviso(`Pedido #${dados.pedido.id} enviado para a cozinha!`, 'sucesso');
    } else {
      mostrarAviso(
        `Pedido #${dados.pedido.id} salvo, mas não foi possível imprimir: ${dados.impressao.motivo}`,
        'alerta'
      );
    }

    carrinho = [];
    observacaoEl.value = '';
    renderizarCarrinho();
  } catch (err) {
    mostrarAviso('Erro de conexão com o servidor. Tente novamente.', 'erro');
    btnConfirmar.disabled = false;
  }
}

btnConfirmar.addEventListener('click', confirmarPedido);

carregarSopas();
renderizarCarrinho();
