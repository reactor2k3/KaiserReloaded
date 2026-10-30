<script lang="ts">
  // Taktische Aufstellung auf der Gefechtskarte: Gattung in der Reserve wählen, dann ein hell
  // markiertes Feld der eigenen Seite anklicken. Klick auf eine gesetzte Einheit nimmt sie zurück.
  // Fehlende Einheiten stellt der Feldherr beim Bestätigen automatisch auf.
  import BattleField from './BattleField.svelte';
  import type { Field, Placement } from '../lib/types';
  import { UNIT_NAMES } from '../lib/format';

  let { field, side, counts, names, placements = $bindable([]) }: {
    field: Field; side: number; counts: Record<string, number>; names: [string, string]; placements?: Placement[];
  } = $props();

  const KINDS = ['cavalry', 'infantry', 'artillery', 'militia'];
  const HINTS: Record<string, string> = {
    cavalry: 'stark im Angriff, reicht weit (19 Felder)',
    infantry: 'solide Linie, reicht 13 Felder',
    artillery: 'schießt 16–21 Felder weit über die Linien',
    militia: 'verteidigt, greift nicht an',
  };
  let selected = $state('cavalry');

  const used = $derived(Object.fromEntries(KINDS.map((k) => [k, placements.filter((p) => p.kind === k).length])));
  const left = (k: string) => (counts[k] ?? 0) - used[k];
  const units = $derived(placements.map((p, i) => ({ id: i + 1, side, kind: p.kind, row: p.row, col: p.col })));
  const total = $derived(KINDS.reduce((s, k) => s + (counts[k] ?? 0), 0));

  // Gattung mit Rest automatisch weiterschalten, wenn die gewählte aufgebraucht ist
  $effect(() => {
    if (left(selected) <= 0) {
      const next = KINDS.find((k) => left(k) > 0);
      if (next) selected = next;
    }
  });

  function pick(r: number, c: number) {
    const i = placements.findIndex((p) => p.row === r && p.col === c);
    if (i >= 0) {
      placements.splice(i, 1);
      return;
    }
    if (left(selected) > 0) placements.push({ kind: selected, row: r, col: c });
  }
</script>

<div class="deploy">
  <div class="board">
    <BattleField {field} {units} {names} selectableSide={side} onpick={pick} maxHeight="calc(100vh - 350px)" />
  </div>
  <div class="reserve">
    <h3 class="label">Reserve · {placements.length} von {total} gesetzt</h3>
    <div role="radiogroup" aria-label="Gattung wählen" class="kinds">
      {#each KINDS as k}
        {#if (counts[k] ?? 0) > 0}
          <button type="button" role="radio" aria-checked={selected === k} class:sel={selected === k} disabled={left(k) <= 0} onclick={() => (selected = k)}>
            <span class="tok {k}"></span>
            <span class="txt"><strong>{UNIT_NAMES[k]}</strong><small>{HINTS[k]}</small></span>
            <span class="num n">{left(k)}</span>
          </button>
        {/if}
      {/each}
    </div>
    <p class="hint">Helle Felder auf Ihrer Seite der roten Linie sind frei. Klick auf eine gesetzte Einheit nimmt sie zurück. Was Sie nicht setzen, stellt der Feldherr auf.</p>
  </div>
</div>

<style>
  .deploy { display: grid; grid-template-columns: minmax(0, 1fr) 270px; gap: 16px; align-items: start; }
  .board { min-width: 0; }
  .reserve { display: flex; flex-direction: column; gap: 8px; }
  .reserve h3 { margin: 0; }
  .kinds { display: flex; flex-direction: column; gap: 6px; }
  .kinds button { display: flex; align-items: center; gap: 10px; min-height: 52px; padding: 4px 10px; border: 1px solid var(--line); border-radius: 6px; background: #fbf6ea; cursor: pointer; text-align: left; }
  .kinds button.sel { border: 2px solid var(--blood); background: #efe0c4; }
  .kinds button:disabled { opacity: .5; cursor: default; }
  .txt { display: flex; flex-direction: column; flex: 1; line-height: 1.2; }
  .txt small { color: var(--sepia); font-size: 13px; }
  .n { font-size: 20px; font-weight: 600; }
  .hint { color: var(--sepia); font-size: 14px; margin: 0; }
  .tok { display: inline-block; width: 28px; height: 14px; border: 1.5px solid #1e1710; flex-shrink: 0; }
  .cavalry { background: linear-gradient(155deg, #4e6b33 42%, #f6eedb 42% 54%, #4e6b33 54%); }
  .infantry { background: linear-gradient(#4e6b33 45%, #f6eedb 45% 58%, #4e6b33 58%); }
  .artillery { background: radial-gradient(circle at 40% 55%, #4e6b33 2px, #f6eedb 2.5px 3.5px, #4e6b33 4px); }
  .militia { background: #eae0c6; border-style: dashed; }
  @media (max-width: 900px) { .deploy { grid-template-columns: 1fr; } }
</style>
