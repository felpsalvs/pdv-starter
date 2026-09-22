<script lang="ts">
  type Page = 'menu' | 'cash-register' | 'order' | 'day';

  let { active }: { active: Page } = $props();

  const links: { page: Page; href: string; label: string }[] = [
    { page: 'menu', href: '/menu.html', label: 'Cardápio' },
    { page: 'cash-register', href: '/cash-register.html', label: 'Caixa' },
    { page: 'order', href: '/index.html', label: 'Balcão' },
    { page: 'day', href: '/day.html', label: 'Dia' },
  ];
</script>

<a
  href="#main-content"
  class="focus:not-sr-only focus:bg-primary focus:text-primary-foreground sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:rounded-md focus:px-4 focus:py-2"
>
  Pular para o conteúdo
</a>

<nav aria-label="Navegação principal" class="flex items-center gap-2 bg-primary px-4 py-3">
  {#each links as link (link.page)}
    <a
      href={link.href}
      class={`rounded-md px-4 py-2 font-semibold text-primary-foreground hover:bg-white/20 ${link.page === active ? 'bg-white/20' : ''}`}
    >
      {link.label}
    </a>
  {/each}
</nav>
