<script lang="ts">
  type Page = 'menu' | 'cash-register' | 'order' | 'day';

  let { active }: { active: Page } = $props();

  const links: { page: Page; href: string; label: string }[] = [
    { page: 'order', href: '/index.html', label: 'Balcão' },
    { page: 'day', href: '/day.html', label: 'Dia' },
    { page: 'cash-register', href: '/cash-register.html', label: 'Caixa' },
    { page: 'menu', href: '/menu.html', label: 'Cardápio' },
  ];
</script>

<a
  href="#main-content"
  class="focus:not-sr-only focus:bg-primary focus:text-primary-foreground sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:rounded-md focus:px-4 focus:py-2"
>
  Pular para o conteúdo
</a>

<aside aria-label="Navegação principal" class="fixed inset-y-0 left-0 z-40 flex w-52 flex-col bg-foreground px-4 py-5 text-background">
  <div class="mb-8 font-mono text-base font-bold tracking-tight">
    <span class="text-warning">»</span> Soparia
  </div>
  <nav class="flex flex-col gap-0.5">
    {#each links as link (link.page)}
      <a
        href={link.href}
        class={`border-l-2 px-3 py-2.5 font-mono text-xs tracking-wide uppercase ${link.page === active ? 'border-primary bg-background/10 opacity-100' : 'border-transparent opacity-60 hover:opacity-100'}`}
      >
        {link.label}
      </a>
    {/each}
  </nav>
</aside>
