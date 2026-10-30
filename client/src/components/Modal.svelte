<script lang="ts">
  // Zentrale Modal-Komponente auf Basis von <dialog>: Fokusfalle, Esc und
  // Fokus-Rückgabe liefert der Browser. Varianten: dialog, confirm, event.
  import type { Snippet } from 'svelte';

  let { open = $bindable(false), title, variant = 'dialog', wide = false, onclose, children, actions }: {
    open?: boolean;
    wide?: boolean;
    title: string;
    variant?: 'dialog' | 'confirm' | 'event';
    onclose?: () => void;
    children: Snippet;
    actions?: Snippet;
  } = $props();

  const uid = $props.id(); // eindeutige ID je Modal für aria-labelledby
  let el: HTMLDialogElement;

  // Das Element bleibt bestehen; nur open/close wird umgeschaltet.
  $effect(() => {
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  });
</script>

<dialog bind:this={el} class={variant} class:wide aria-labelledby="{uid}-title" onclose={() => { open = false; onclose?.(); }}>
  <header>
    <h2 id="{uid}-title" class="display">{title}</h2>
    <button type="button" class="x" aria-label="Schließen" onclick={() => (open = false)}>×</button>
  </header>
  <div class="body">{@render children()}</div>
  {#if actions}<footer>{@render actions()}</footer>{/if}
</dialog>

<style>
  dialog {
    width: min(620px, calc(100vw - 32px));
    max-height: calc(100vh - 48px);
    /* Rahmen und Pergament: Zierrahmen aus app.css (border-image); Schatten folgt der Rahmenform */
    color: var(--ink);
    padding: 14px 18px 12px; /* Platz für die großen Eckschnörkel des Rahmens */
    filter: drop-shadow(0 18px 30px #00000070);
  }
  /* Kriegsmeldungen: Rahmen rötlich getönt */
  dialog.confirm { filter: drop-shadow(0 0 2px #7a1f1f) drop-shadow(0 18px 30px #00000070); }
  dialog.wide { width: min(1100px, calc(100vw - 32px)); }
  dialog::backdrop { background: #2b2118aa; }
  header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 12px; }
  h2 { font-size: 34px; line-height: 1.05; }
  .x { min-width: 44px; min-height: 44px; border: 1px solid var(--line); border-radius: 4px; background: transparent; font-size: 22px; cursor: pointer; }
  .body { display: flex; flex-direction: column; gap: 14px; }
  footer { display: flex; justify-content: flex-end; gap: 12px; margin-top: 18px; flex-wrap: wrap; }
</style>
