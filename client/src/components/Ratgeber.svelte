<script lang="ts">
  // „Der Kanzler rät“: Hinweise aus lib/ratgeber.ts. compact = nur der wichtigste Rat (Fußleiste),
  // all = alle Ratschläge (Modal), sonst die Liste ohne die Phasenaufgabe.
  import { app } from '../lib/state.svelte';
  import { advise } from '../lib/ratgeber';

  let { compact = false, all = false }: { compact?: boolean; all?: boolean } = $props();

  const tips = $derived(app.self ? advise(app.self, app.year, app.players) : []);
  // Die erste info-Zeile ist die Phasenaufgabe
  const todo = $derived(tips.findIndex((t) => t.level === 'info'));
  const shown = $derived(compact ? tips.slice(0, 1) : all ? tips : tips.filter((_, i) => i !== todo));
</script>

{#if shown.length}
  <div class="ratgeber" class:compact role="note" aria-label="Der Kanzler rät">
    {#if !compact && !all}<span class="label head">Der Kanzler rät</span>{/if}
    <ul>
      {#each shown as t (t.text)}
        <li class={t.level}>
          <span class="mark" aria-hidden="true">{t.level === 'muss' ? '!' : t.level === 'rat' ? '❧' : '·'}</span>
          <span><span class="sr">{t.level === 'muss' ? 'Wichtig: ' : t.level === 'rat' ? 'Rat: ' : ''}</span>{t.text}</span>
        </li>
      {/each}
    </ul>
  </div>
{/if}

<style>
  .ratgeber { display: flex; flex-direction: column; gap: 4px; padding: 6px 10px; border-left: 3px solid var(--gold); background: #f3e7cc; border-radius: 0 4px 4px 0; }
  .ratgeber.compact { padding: 0; border: none; background: none; }
  .head { color: var(--sepia); }
  ul { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
  /* relative: hält die versteckten Vorlese-Texte (.sr) im Eintrag, sonst ragen sie aus der Seite */
  li { position: relative; display: flex; gap: 8px; line-height: 1.35; font-size: 16px; }
  .mark { width: 16px; flex-shrink: 0; text-align: center; font-weight: 700; color: var(--gold); }
  li.muss .mark { color: #fff; background: var(--blood); border-radius: 50%; height: 18px; line-height: 18px; font-size: 13px; margin-top: 1px; }
  li.muss { color: var(--blood); font-weight: 600; }
  .sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
</style>
