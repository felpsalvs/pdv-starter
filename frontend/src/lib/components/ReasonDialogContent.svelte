<script lang="ts">
  import { Button } from '$lib/components/ui/button/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';

  let {
    title,
    options,
    allowOther = false,
    onSelect,
  }: {
    title: string;
    options: string[];
    allowOther?: boolean;
    onSelect: (reason: string) => void;
  } = $props();

  let showOther = $state(false);
  let otherText = $state('');

  function submitOther() {
    const value = otherText.trim();
    if (value) onSelect(value);
  }
</script>

<Dialog.Header>
  <Dialog.Title>{title}</Dialog.Title>
</Dialog.Header>

{#if !showOther}
  <div class="flex flex-col gap-2">
    {#each options as option (option)}
      <Button variant="secondary" class="justify-start" onclick={() => onSelect(option)}>{option}</Button>
    {/each}
    {#if allowOther}
      <Button variant="secondary" class="justify-start" onclick={() => (showOther = true)}>Outro</Button>
    {/if}
  </div>
{:else}
  <div class="flex flex-col gap-3">
    <Input
      placeholder="Descreva o motivo"
      bind:value={otherText}
      onkeydown={(e) => e.key === 'Enter' && submitOther()}
    />
    <Button onclick={submitOther}>Confirmar</Button>
  </div>
{/if}
