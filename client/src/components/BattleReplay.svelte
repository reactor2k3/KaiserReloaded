<script lang="ts">
  // Spielt eine Schlacht Schritt für Schritt ab (Daten aus dem Server-Ereignis „battle“).
  // Links das Schlachtfeld, rechts Stand der Truppen, Verlauf in Worten und Legende.
  // Einheiten bleiben bestehen; pro Schritt ändern sich nur Position und Zustand.
  import BattleField from './BattleField.svelte';
  import { UNIT_NAMES } from '../lib/format';
  import { sfxCannon, sfxCrash, sfxHit } from '../lib/audio.svelte';
  import type { Field, FieldUnit } from '../lib/types';
  import type { Snippet } from 'svelte';

  interface Step { unit: number; action: string; row: number; col: number; target?: number; what?: string }

  let { field, units: startUnits, steps, names, done = $bindable(false), result }: {
    field: Field; units: FieldUnit[]; steps: Step[]; names: [string, string]; done?: boolean; result?: Snippet;
  } = $props();

  const SPEEDS = [{ label: 'Langsam', ms: 220 }, { label: 'Normal', ms: 110 }, { label: 'Schnell', ms: 45 }];
  const BUILDINGS: Record<string, string> = { M: 'einen Markt', L: 'eine Mühle', P: 'den Palast', K: 'die Kathedrale' };

  // Ausgangslage aus den Props kopieren; eine neue Schlacht montiert die Komponente neu ({#key}).
  // svelte-ignore state_referenced_locally
  let units = $state<FieldUnit[]>(startUnits.map((u) => ({ ...u, dead: false })));
  // svelte-ignore state_referenced_locally
  let cells = $state(field.cells.map((r) => r.split('')));
  let idx = $state(0);
  let playing = $state(true);
  let speed = $state(1);
  let flash = $state<{ row: number; col: number; kind: string } | null>(null);
  let log = $state<{ id: number; text: string; kind: string }[]>([]);
  let lastSide = -1;
  let lastMover = -1;
  let logId = 0;

  const byId = $derived(new Map(units.map((u) => [u.id, u])));
  // Truppenstand je Seite und Gattung
  const tally = $derived([0, 1].map((side) => {
    const own = units.filter((u) => u.side === side);
    return {
      alive: own.filter((u) => !u.dead).length,
      total: own.length,
      kinds: ['cavalry', 'infantry', 'artillery', 'militia'].map((k) => ({
        kind: k, alive: own.filter((u) => u.kind === k && !u.dead).length, total: own.filter((u) => u.kind === k).length,
      })).filter((k) => k.total > 0),
    };
  }));

  const label = (u?: FieldUnit) => (u ? `${UNIT_NAMES[u.kind]} von ${names[u.side]}` : 'eine Einheit');

  function say(text: string, kind = 'info') {
    log.unshift({ id: logId++, text, kind });
    if (log.length > 40) log.length = 40;
  }

  // Einen Schritt anwenden und in Worten beschreiben.
  function apply(s: Step, quiet = false) {
    const u = byId.get(s.unit);
    const t = s.target ? byId.get(s.target) : undefined;
    flash = null;
    if (u && u.side !== lastSide) {
      lastSide = u.side;
      say(u.side === 0 ? `Angriff! ${names[0]} rückt vor.` : `Gegenangriff! ${names[1]} schlägt zurück.`, 'phase');
    }
    switch (s.action) {
      case 'move':
        if (u) { u.row = s.row; u.col = s.col; }
        if (s.unit !== lastMover) say(`${label(u)} rückt vor.`, 'move');
        break;
      case 'win':
        if (t) t.dead = true;
        flash = { row: s.row, col: s.col, kind: 'hit' };
        say(`${label(u)} besiegt ${label(t)}.`, u?.side === 0 ? 'good' : 'bad');
        if (!quiet) sfxHit();
        break;
      case 'lose':
        if (u) u.dead = true;
        flash = { row: s.row, col: s.col, kind: 'hit' };
        say(`${label(u)} fällt im Kampf gegen ${label(t)}.`, u?.side === 0 ? 'bad' : 'good');
        if (!quiet) sfxHit();
        break;
      case 'destroy':
        cells[s.row][s.col] = 'x';
        flash = { row: s.row, col: s.col, kind: 'fire' };
        say(`${label(u)} zerstört ${BUILDINGS[s.what ?? ''] ?? 'ein Gebäude'}.`, 'fire');
        if (!quiet) sfxCrash();
        break;
      case 'shot':
        if (t) t.dead = true;
        if (s.what) cells[s.row][s.col] = 'x';
        flash = { row: s.row, col: s.col, kind: 'shot' };
        if (!quiet) sfxCannon();
        say(t ? `Artillerie von ${names[u?.side ?? 0]} trifft ${label(t)}.`
          : s.what ? `Artillerie von ${names[u?.side ?? 0]} trifft ${BUILDINGS[s.what] ?? 'ein Gebäude'}.`
          : `Artillerie von ${names[u?.side ?? 0]} feuert – daneben.`, 'shot');
        break;
    }
    lastMover = s.action === 'move' ? s.unit : -1;
  }

  // Taktgeber für die Wiedergabe
  $effect(() => {
    if (!playing) return;
    const t = setInterval(() => {
      if (idx >= steps.length) {
        playing = false;
        flash = null;
        done = true;
        return;
      }
      apply(steps[idx]);
      idx++;
    }, SPEEDS[speed].ms);
    return () => clearInterval(t);
  });

  function skip() {
    while (idx < steps.length) apply(steps[idx++], true);
    playing = false;
    flash = null;
    done = true;
  }

  function restart() {
    units = startUnits.map((u) => ({ ...u, dead: false }));
    cells = field.cells.map((r) => r.split(''));
    log = [];
    lastSide = lastMover = -1;
    idx = 0;
    done = false;
    playing = true;
  }
</script>

<div class="replay">
  <div class="left">
    <BattleField {field} {cells} {units} {names} {flash} maxHeight="calc(100vh - 340px)" />
    <div class="controls">
      <progress max={steps.length} value={idx} aria-label="Fortschritt der Schlacht"></progress>
      {#if idx < steps.length}
        <button class="btn" type="button" onclick={() => (playing = !playing)}>{playing ? 'Pause' : 'Weiter'}</button>
        <div class="speed" role="radiogroup" aria-label="Tempo">
          {#each SPEEDS as s, i}
            <button type="button" role="radio" aria-checked={speed === i} class:on={speed === i} onclick={() => (speed = i)}>{s.label}</button>
          {/each}
        </div>
        <button class="btn" type="button" onclick={skip}>Zum Ergebnis</button>
      {:else}
        <button class="btn" type="button" onclick={restart}>Nochmal ansehen</button>
      {/if}
    </div>
  </div>

  <aside class="right">
    <section class="tally" aria-label="Truppenstand">
      {#each tally as t, side}
        <div class="side s{side}">
          <div class="head"><span class="dot"></span>{names[side]}<span class="num count">{t.alive} / {t.total}</span></div>
          <div class="kinds">
            {#each t.kinds as k}<span class="num">{UNIT_NAMES[k.kind]} {k.alive}/{k.total}</span>{/each}
          </div>
        </div>
      {/each}
    </section>

    {#if done && result}
      <section class="result">{@render result()}</section>
    {:else}
      <section class="log" aria-label="Verlauf der Schlacht" aria-live="polite">
        <h3 class="label">Verlauf</h3>
        <ol>
          {#each log as l (l.id)}<li class={l.kind}>{l.text}</li>{/each}
        </ol>
      </section>
    {/if}

    <section class="legend" aria-label="Legende">
      <span><i class="blk a"></i>Angreifer</span><span><i class="blk d"></i>Verteidiger</span>
      <span><i class="blk a cav"></i>Kavallerie</span><span><i class="blk a inf"></i>Infanterie</span>
      <span><i class="blk a art"></i>Artillerie</span><span><i class="blk mil"></i>Miliz</span>
      <span><i class="line"></i>Grenze</span><span><i class="sym x">✕</i>gefallen</span>
    </section>
  </aside>
</div>

<style>
  .replay { display: grid; grid-template-columns: minmax(0, 1fr) 290px; gap: 16px; align-items: start; }
  .left { display: flex; flex-direction: column; gap: 8px; min-width: 0; }
  .controls { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
  progress { flex: 1; min-width: 100px; accent-color: var(--blood); }
  .speed { display: flex; border: 1px solid var(--line-strong); border-radius: 4px; overflow: hidden; }
  .speed button { min-height: 44px; padding: 0 10px; border: none; background: transparent; font-family: var(--font-label); font-size: 12px; font-weight: 600; cursor: pointer; }
  .speed button.on { background: var(--ink); color: var(--parchment); }

  /* Rechte Spalte nie höher als das Feld plus Bedienleiste, damit das Modal nicht scrollt */
  .right { display: flex; flex-direction: column; gap: 8px; min-height: 0; max-height: calc(100vh - 280px); }
  .tally { display: flex; flex-direction: column; gap: 6px; }
  .side { border: 1px solid var(--line); border-radius: 4px; padding: 6px 8px; }
  .head { display: flex; align-items: center; gap: 6px; font-weight: 600; }
  .count { margin-left: auto; }
  .dot { width: 12px; height: 12px; border: 1px solid var(--ink); }
  .s0 .dot { background: #4e6b33; }
  .s1 .dot { background: #3f6a99; }
  .kinds { display: flex; flex-wrap: wrap; gap: 2px 10px; font-size: 14px; color: var(--sepia); }

  .log { flex: 1; min-height: 120px; overflow-y: auto; }
  .log h3 { margin: 0 0 4px; }
  .log ol { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; font-size: 15px; line-height: 1.3; }
  .log li { padding-left: 8px; border-left: 3px solid transparent; }
  .log li:first-child { font-weight: 600; }
  .log .phase { font-family: var(--font-label); font-size: 13px; letter-spacing: .05em; color: var(--blood); border-left-color: var(--blood); }
  .log .good { border-left-color: #4e6b33; }
  .log .bad { border-left-color: #a3282b; }
  .log .fire, .log .shot { border-left-color: #d9822b; }
  .log .move { color: var(--sepia); }
  .result { flex: 1; min-height: 0; overflow-y: auto; }

  .legend { display: grid; grid-template-columns: 1fr 1fr; gap: 4px 10px; font-size: 13px; color: var(--sepia); }
  .legend span { display: flex; align-items: center; gap: 6px; }
  .blk { display: inline-block; width: 20px; height: 10px; border: 1.3px solid #1e1710; flex-shrink: 0; }
  .blk.a { background: #4e6b33; }
  .blk.d { background: #3f6a99; }
  .blk.cav { background: linear-gradient(155deg, #4e6b33 42%, #f6eedb 42% 54%, #4e6b33 54%); }
  .blk.inf { background: linear-gradient(#4e6b33 45%, #f6eedb 45% 58%, #4e6b33 58%); }
  .blk.art { background: radial-gradient(circle at 40% 55%, #4e6b33 2px, #f6eedb 2.5px 3.5px, #4e6b33 4px); }
  .blk.mil { background: #eae0c6; border-style: dashed; }
  .line { display: inline-block; width: 20px; border-top: 3px dashed #7a1f1f; }
  .sym.x { font-style: normal; color: #7a1f1f; font-weight: 700; width: 20px; text-align: center; }

  @media (max-width: 900px) {
    .replay { grid-template-columns: 1fr; }
    .right { max-height: none; }
    .log { max-height: 200px; }
  }
</style>
