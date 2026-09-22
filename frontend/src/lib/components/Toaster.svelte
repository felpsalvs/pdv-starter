<script lang="ts">
  import { toasts, dismiss } from '$lib/toast.svelte.js';
  import X from '@lucide/svelte/icons/x';

  const typeClasses: Record<string, string> = {
    success: 'border-l-success',
    error: 'border-l-destructive',
    warning: 'border-l-warning',
  };
</script>

<div class="fixed top-4 right-4 z-[100] flex w-full max-w-sm flex-col gap-2" role="status" aria-live="polite">
  {#each toasts as item (item.id)}
    <div
      class={`flex items-start gap-2 rounded-md border border-l-4 bg-card px-4 py-3 text-sm shadow-lg ${typeClasses[item.type]}`}
    >
      <span class="flex-1 pt-px text-card-foreground">{item.message}</span>
      <button
        type="button"
        class="shrink-0 text-muted-foreground hover:text-foreground"
        aria-label="Fechar aviso"
        onclick={() => dismiss(item.id)}
      >
        <X class="size-4" />
      </button>
    </div>
  {/each}
</div>
