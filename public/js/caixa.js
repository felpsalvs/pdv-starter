const conteudoEl = document.getElementById('conteudo');
const avisoEl = document.getElementById('aviso');

function formatarPreco(valor) {
  return valor.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
}

function mostrarAviso(mensagem, tipo) {
  avisoEl.innerHTML = `<div class="aviso ${tipo}">${mensagem}</div>`;
}

function limparAviso() {
  avisoEl.innerHTML = '';
}

async function carregarCaixa() {
  const resposta = await fetch('/api/caixa/atual');
  const caixa = await resposta.json();

  if (!caixa) {
    renderizarAbertura();
  } else {
    renderizarAberto(caixa);
  }
}

function renderizarAbertura() {
  conteudoEl.innerHTML = `
    <div class="form-box">
      <h2>Abrir caixa</h2>
      <input type="number" id="valor-abertura" placeholder="Valor inicial (ex: 50.00)" step="0.01" min="0" />
      <button id="btn-abrir" class="principal">Abrir caixa</button>
    </div>
  `;

  document.getElementById('btn-abrir').addEventListener('click', abrirCaixa);
}

function renderizarAberto(caixa) {
  conteudoEl.innerHTML = `
    <div class="resumo-caixa">
      <div class="linha"><span>Aberto em</span><span>${caixa.aberto_em}</span></div>
      <div class="linha"><span>Valor de abertura</span><span>${formatarPreco(caixa.valor_abertura)}</span></div>
      <div class="linha"><span>Vendas até agora</span><span>${formatarPreco(caixa.total_vendas_parcial)}</span></div>
      <div class="linha destaque"><span>Total esperado agora</span><span>${formatarPreco(caixa.valor_abertura + caixa.total_vendas_parcial)}</span></div>
    </div>

    <div class="form-box">
      <h2>Fechar caixa</h2>
      <input type="number" id="valor-contado" placeholder="Valor contado na gaveta" step="0.01" min="0" />
      <button id="btn-fechar" class="principal">Fechar caixa</button>
    </div>
  `;

  document.getElementById('btn-fechar').addEventListener('click', fecharCaixa);
}

async function abrirCaixa() {
  limparAviso();
  const valor = Number(document.getElementById('valor-abertura').value);

  if (!Number.isFinite(valor) || valor < 0) {
    mostrarAviso('Informe um valor de abertura válido.', 'erro');
    return;
  }

  const resposta = await fetch('/api/caixa/abrir', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ valor_abertura: valor }),
  });

  const dados = await resposta.json();

  if (!resposta.ok) {
    mostrarAviso(dados.erro || 'Erro ao abrir caixa.', 'erro');
    return;
  }

  carregarCaixa();
}

async function fecharCaixa() {
  limparAviso();
  const valor = Number(document.getElementById('valor-contado').value);

  if (!Number.isFinite(valor) || valor < 0) {
    mostrarAviso('Informe um valor contado válido.', 'erro');
    return;
  }

  const resposta = await fetch('/api/caixa/fechar', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ valor_contado: valor }),
  });

  const dados = await resposta.json();

  if (!resposta.ok) {
    mostrarAviso(dados.erro || 'Erro ao fechar caixa.', 'erro');
    return;
  }

  const sinal = dados.diferenca >= 0 ? 'sobra' : 'falta';
  mostrarAviso(
    `Caixa fechado. Esperado: ${formatarPreco(dados.valor_esperado)} | Contado: ${formatarPreco(
      dados.valor_contado
    )} | Diferença: ${formatarPreco(Math.abs(dados.diferenca))} (${sinal})`,
    Math.abs(dados.diferenca) < 0.01 ? 'sucesso' : 'alerta'
  );

  renderizarAbertura();
}

carregarCaixa();
