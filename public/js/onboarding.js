(function () {
  const currentScript = document.currentScript;

  const PAGE_TOURS = {
    order: [
      {
        title: 'Bem-vindo ao Soparia PDV',
        body: 'Esse é o Balcão — a tela onde você lança os pedidos do dia a dia. Vamos te mostrar rapidinho onde fica cada coisa.',
      },
      {
        target: '#search-input',
        title: 'Comece digitando',
        body: 'O cursor já fica aqui. Digite o nome de uma sopa e aperte Enter pra adicionar. Pra pedir mais de uma, digite o número antes: "2 caldo verde".',
      },
      {
        target: '#category-tabs',
        title: 'Filtre por categoria',
        body: 'Toque numa aba para ver só as sopas, só as bebidas, e assim por diante.',
      },
      {
        target: '#cart-panel',
        title: 'O pedido atual',
        body: 'Aqui aparece o que já foi adicionado. Use os botões + e − pra ajustar a quantidade, ou digite uma observação em cada item.',
      },
      {
        target: '#reference-button',
        title: 'Diga pra quem é',
        body: 'Aperte F2 (ou toque aqui) para marcar Balcão ou Mesa — isso vai no ticket da cozinha e na etiqueta.',
      },
      {
        target: '#pay-button',
        title: 'Cobrar ou deixar em aberto',
        body: 'F4 abre o pagamento (dinheiro, pix, débito ou crédito). F8 envia para a cozinha sem cobrar — vira uma conta em aberto, pra fechar depois.',
      },
      {
        target: '.shortcuts-bar',
        title: 'Atalhos sempre à mão',
        body: 'Esse rodapé lembra todos os atalhos de teclado. Pronto — é só isso pra começar a vender!',
      },
    ],
    day: [
      {
        title: 'A tela Dia',
        body: 'Aqui fica a fila de tudo que aconteceu hoje: contas de mesa ainda em aberto, pedidos pagos e cancelamentos.',
      },
      {
        target: '#day-summary',
        title: 'O resumo do dia',
        body: 'Quantos pedidos foram pagos, quantos cancelados, e quantas contas ainda estão em aberto.',
      },
      {
        target: '#filters',
        title: 'Filtre a lista',
        body: 'Separe contas abertas, pagos, cancelados, ou veja todos de uma vez.',
      },
      {
        target: '#order-list',
        title: 'Cada pedido, com suas ações',
        body: 'Receba o pagamento de uma conta de mesa, reimprima o ticket da cozinha ou a etiqueta, ou cancele um pedido errado — sempre informando um motivo.',
      },
    ],
    'cash-register': [
      {
        title: 'A tela Caixa',
        body: 'Controla o dinheiro físico da gaveta. Sem caixa aberto, o Balcão não deixa registrar nenhum pagamento.',
      },
      {
        target: '#content',
        title: 'Abra, acompanhe e feche',
        body: 'Abra o caixa informando o valor que está na gaveta. Durante o dia, acompanhe dinheiro, pix e cartão separados. No fim, feche conferindo o valor contado com o esperado.',
      },
      {
        target: '#history',
        title: 'Histórico de fechamentos',
        body: 'Todo fechamento já feito fica registrado aqui, com a diferença encontrada em cada um.',
      },
    ],
    menu: [
      {
        title: 'A tela Cardápio',
        body: 'Aqui você organiza categorias e produtos, e escolhe todo dia o que está disponível — o primeiro passo antes de abrir.',
      },
      {
        target: '#category-form',
        title: 'Organize por categoria',
        body: 'Crie categorias como Sopas ou Bebidas para separar o cardápio.',
      },
      {
        target: '#product-form',
        title: 'Cadastre um produto',
        body: 'Nome, preço e categoria — pronto, já aparece na lista abaixo.',
      },
      {
        target: '#product-list',
        title: 'Marque a sopa do dia',
        body: 'Todo dia, ligue "Disponível hoje" só no que você vai vender — é só isso que aparece no Balcão.',
      },
    ],
  };

  function storageKey(pageKey) {
    return `pdv-tour-seen-${pageKey}`;
  }

  function hasSeenTour(pageKey) {
    try {
      return localStorage.getItem(storageKey(pageKey)) === '1';
    } catch (err) {
      return false;
    }
  }

  function markTourSeen(pageKey) {
    try {
      localStorage.setItem(storageKey(pageKey), '1');
    } catch (err) {
      // Armazenamento indisponível (ex: navegação privada) — sem problema,
      // o tour só vai aparecer de novo na próxima visita.
    }
  }

  let overlayEl = null;
  let spotlightEl = null;
  let cardEl = null;
  let liveRegionEl = null;
  let steps = [];
  let stepIndex = 0;
  let previouslyFocused = null;
  let keydownHandler = null;
  let resizeHandler = null;

  function buildDom() {
    overlayEl = document.createElement('div');
    overlayEl.className = 'tour-overlay';

    spotlightEl = document.createElement('div');
    spotlightEl.className = 'tour-spotlight';

    cardEl = document.createElement('div');
    cardEl.className = 'tour-card';
    cardEl.setAttribute('role', 'dialog');
    cardEl.setAttribute('aria-modal', 'true');
    cardEl.setAttribute('aria-labelledby', 'tour-title');
    cardEl.setAttribute('aria-describedby', 'tour-body');
    cardEl.setAttribute('tabindex', '-1');

    liveRegionEl = document.createElement('div');
    liveRegionEl.className = 'sr-only';
    liveRegionEl.setAttribute('aria-live', 'polite');

    document.body.appendChild(overlayEl);
    document.body.appendChild(spotlightEl);
    document.body.appendChild(cardEl);
    document.body.appendChild(liveRegionEl);
  }

  function removeDom() {
    [overlayEl, spotlightEl, cardEl, liveRegionEl].forEach((el) => el && el.remove());
    overlayEl = spotlightEl = cardEl = liveRegionEl = null;
  }

  function positionSpotlight(target) {
    if (!target) {
      spotlightEl.classList.add('no-target');
      return;
    }
    spotlightEl.classList.remove('no-target');
    const rect = target.getBoundingClientRect();
    const pad = 8;
    spotlightEl.style.top = `${rect.top - pad}px`;
    spotlightEl.style.left = `${rect.left - pad}px`;
    spotlightEl.style.width = `${rect.width + pad * 2}px`;
    spotlightEl.style.height = `${rect.height + pad * 2}px`;
  }

  function positionCard(target) {
    const cardRect = cardEl.getBoundingClientRect();
    const margin = 16;
    let top;
    let left;

    if (!target) {
      top = (window.innerHeight - cardRect.height) / 2;
      left = (window.innerWidth - cardRect.width) / 2;
    } else {
      const rect = target.getBoundingClientRect();
      const spaceBelow = window.innerHeight - rect.bottom;
      top = spaceBelow > cardRect.height + 24 ? rect.bottom + 16 : rect.top - cardRect.height - 16;
      left = rect.left;
    }

    top = Math.max(margin, Math.min(top, window.innerHeight - cardRect.height - margin));
    left = Math.max(margin, Math.min(left, window.innerWidth - cardRect.width - margin));

    cardEl.style.top = `${top}px`;
    cardEl.style.left = `${left}px`;
  }

  function reposition() {
    const step = steps[stepIndex];
    const target = step.target ? document.querySelector(step.target) : null;
    positionSpotlight(target);
    positionCard(target);
  }

  function renderStep() {
    const step = steps[stepIndex];
    const isFirst = stepIndex === 0;
    const isLast = stepIndex === steps.length - 1;

    cardEl.innerHTML = `
      <div class="tour-eyebrow">Passo ${stepIndex + 1} de ${steps.length}</div>
      <h2 id="tour-title">${step.title}</h2>
      <p id="tour-body">${step.body}</p>
      <div class="tour-actions">
        <button type="button" class="tour-skip" data-tour-action="skip">Pular tutorial</button>
        <div class="tour-nav-buttons">
          ${isFirst ? '' : '<button type="button" class="secondary" data-tour-action="back">Voltar</button>'}
          <button type="button" class="primary" data-tour-action="next">${isLast ? 'Concluir' : 'Próximo'}</button>
        </div>
      </div>
    `;

    reposition();

    liveRegionEl.textContent = `Passo ${stepIndex + 1} de ${steps.length}: ${step.title}`;

    const nextButton = cardEl.querySelector('[data-tour-action="next"]');
    nextButton.focus();

    cardEl.querySelectorAll('[data-tour-action]').forEach((button) => {
      button.addEventListener('click', () => {
        const action = button.dataset.tourAction;
        if (action === 'next') goNext();
        else if (action === 'back') goBack();
        else if (action === 'skip') endTour();
      });
    });
  }

  function goNext() {
    if (stepIndex < steps.length - 1) {
      stepIndex += 1;
      renderStep();
    } else {
      endTour();
    }
  }

  function goBack() {
    if (stepIndex > 0) {
      stepIndex -= 1;
      renderStep();
    }
  }

  function endTour() {
    markTourSeen(currentPageKey);
    window.pdvTourActive = false;
    document.removeEventListener('keydown', keydownHandler);
    window.removeEventListener('resize', resizeHandler);
    removeDom();
    if (previouslyFocused && typeof previouslyFocused.focus === 'function') {
      previouslyFocused.focus();
    }
  }

  let currentPageKey = null;

  function trapFocus(ev) {
    if (ev.key === 'Escape') {
      ev.preventDefault();
      endTour();
      return;
    }
    if (ev.key === 'ArrowRight') {
      ev.preventDefault();
      goNext();
      return;
    }
    if (ev.key === 'ArrowLeft') {
      ev.preventDefault();
      goBack();
      return;
    }
    if (ev.key !== 'Tab') return;

    const focusable = Array.from(cardEl.querySelectorAll('button'));
    if (focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];

    if (ev.shiftKey && document.activeElement === first) {
      ev.preventDefault();
      last.focus();
    } else if (!ev.shiftKey && document.activeElement === last) {
      ev.preventDefault();
      first.focus();
    }
  }

  function startTour(pageKey) {
    const tourSteps = PAGE_TOURS[pageKey];
    if (!tourSteps || tourSteps.length === 0) return;

    currentPageKey = pageKey;
    steps = tourSteps;
    stepIndex = 0;
    previouslyFocused = document.activeElement;
    window.pdvTourActive = true;

    buildDom();
    renderStep();

    keydownHandler = trapFocus;
    resizeHandler = () => reposition();
    document.addEventListener('keydown', keydownHandler);
    window.addEventListener('resize', resizeHandler);
  }

  window.pdvTour = {
    start(pageKey) {
      startTour(pageKey || (currentScript && currentScript.dataset.page));
    },
  };

  const pageKey = currentScript && currentScript.dataset.page;
  if (pageKey && !hasSeenTour(pageKey)) {
    window.setTimeout(() => startTour(pageKey), 400);
  }
})();
