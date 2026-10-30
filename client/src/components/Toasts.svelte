<script lang="ts">
  // Hinweise oben rechts; schließen sich selbst (siehe toast() im Zustand).
  import { app } from '../lib/state.svelte';
  import { flip } from 'svelte/animate';
  import { fly } from 'svelte/transition';

  function close(id: number) {
    const i = app.toasts.findIndex((t) => t.id === id);
    if (i >= 0) app.toasts.splice(i, 1);
  }
</script>

<div class="toasts" aria-live="polite">
  {#each app.toasts as t (t.id)}
    <div class="toast {t.kind}" role="status" animate:flip={{ duration: 200 }} transition:fly={{ x: 40, duration: 200 }}>
      <span>{t.text}</span>
      <button type="button" aria-label="Hinweis schließen" onclick={() => close(t.id)}>×</button>
    </div>
  {/each}
</div>

<style>
  .toasts { position: fixed; top: 84px; right: 20px; display: flex; flex-direction: column; gap: 10px; z-index: 50; width: min(380px, calc(100vw - 32px)); }
  .toast {
    display: flex; gap: 10px; align-items: flex-start; background: var(--parchment);
    border: 1px solid var(--line); border-radius: 6px; padding: 12px 8px 12px 16px;
    box-shadow: 0 12px 24px #00000040; font-size: 16px; line-height: 1.35;
  }
  .toast.warn { border-left: 4px solid var(--warn); }
  .toast.error { border-left: 4px solid var(--blood); }
  .toast.info { border-left: 4px solid var(--gold); }
  .toast span { flex: 1; }
  button { min-width: 36px; min-height: 36px; border: none; background: transparent; font-size: 22px; cursor: pointer; }
</style>
