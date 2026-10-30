<script lang="ts">
  // Schlachtfeld im Stil einer barocken Gefechtskarte. Gemeinsam genutzt von Aufstellung
  // (klickbare Felder) und Wiedergabe (Einheiten bewegen sich, Treffer blitzen auf).
  // Gelände wird einmal gezeichnet; nur Einheiten und Markierungen ändern sich.
  import type { Field, FieldUnit } from '../lib/types';

  let { field, cells, units = [], names, flash = null, selectableSide = -1, onpick, maxHeight = '60vh' }: {
    field: Field;
    cells?: string[][];                // aktueller Zellstand (zerstörte Bauten); sonst field.cells
    units?: FieldUnit[];
    names: [string, string];           // Angreifer, Verteidiger
    flash?: { row: number; col: number; kind: string } | null;
    selectableSide?: number;           // -1 = nicht klickbar, sonst eigene Seite bei der Aufstellung
    onpick?: (row: number, col: number) => void;
    maxHeight?: string;
  } = $props();

  const C = 24; // Zellgröße in SVG-Einheiten
  const W = $derived(field.cols * C);
  const H = $derived(field.rows * C);
  const grid = $derived(cells ?? field.cells.map((r) => r.split('')));

  // Grenzlinie als Polyline durch die Mitte der Grenzzellen
  const borderLine = $derived(field.border.map((b, r) => `${b * C + C / 2},${r * C} ${b * C + C / 2},${r * C + C}`).join(' '));
  // Landflächen beider Seiten als Pfad (für die Tönung)
  const leftArea = $derived('M0 0' + field.border.map((b, r) => `H${b * C + C / 2}V${(r + 1) * C}`).join('') + `H0Z`);
  const rightArea = $derived(`M${W} 0` + field.border.map((b, r) => `H${b * C + C / 2}V${(r + 1) * C}`).join('') + `H${W}Z`);

  function selectable(r: number, c: number) {
    if (selectableSide < 0 || grid[r][c] !== '.') return false;
    return selectableSide === 0 ? c <= field.border[r] - 2 : c >= field.border[r] + 2;
  }
  const occupied = $derived(new Set(units.filter((u) => !u.dead).map((u) => `${u.row}:${u.col}`)));
</script>

<svg viewBox="-4 -30 {W + 8} {H + 34}" style:max-height={maxHeight} role="img"
  aria-label="Schlachtfeld: links {names[0]}, rechts {names[1]}">
  <defs>
    <pattern id="bf-fields" width="48" height="48" patternUnits="userSpaceOnUse">
      <rect width="48" height="48" fill="#e6dcb6" />
      <path d="M0 12H48M0 24H48M0 36H48" stroke="#d6c99d" stroke-width="0.8" />
    </pattern>
    <pattern id="bf-water" width="12" height="6" patternUnits="userSpaceOnUse">
      <rect width="12" height="6" fill="#a9c1cf" />
      <path d="M0 3Q3 1 6 3T12 3" fill="none" stroke="#7f9db3" stroke-width="0.8" />
    </pattern>
  </defs>

  <!-- Seitenbeschriftung -->
  <text x={W / 4} y="-10" class="side a" text-anchor="middle">{names[0].toUpperCase()} · ANGRIFF →</text>
  <text x={(W * 3) / 4} y="-10" class="side d" text-anchor="middle">← VERTEIDIGUNG · {names[1].toUpperCase()}</text>

  <!-- Gelände: Felder, Tönung je Seite, Rahmen -->
  <rect width={W} height={H} fill="url(#bf-fields)" />
  <path d={leftArea} fill="#4e6b33" opacity="0.10" />
  <path d={rightArea} fill="#3f6a99" opacity="0.09" />

  {#each grid as row, r}
    {#each row as cell, c}
      {@const x = c * C}
      {@const y = r * C}
      {#if cell === 'T'}
        <circle cx={x + 10} cy={y + 15} r="7" fill="#5f7346" opacity=".45" />
        <circle cx={x + 9} cy={y + 12} r="7" fill="#7e9660" stroke="#3e4f2c" stroke-width="0.8" />
        <circle cx={x + 16} cy={y + 15} r="5.5" fill="#8aa06a" stroke="#3e4f2c" stroke-width="0.8" />
      {:else if cell === '~'}
        <rect x={x - 1} y={y - 1} width={C + 2} height={C + 2} fill="url(#bf-water)" />
      {:else if cell === 'M'}
        <path d="M{x + 5} {y + 12}L{x + 12} {y + 5}L{x + 19} {y + 12}Z" fill="#a0452f" stroke="#4a2a1c" stroke-width="0.8" />
        <rect x={x + 6} y={y + 12} width="12" height="8" fill="#e9ddbf" stroke="#4a2a1c" stroke-width="0.8" />
      {:else if cell === 'L'}
        <path d="M{x + 9} {y + 21}L{x + 10.5} {y + 11}H{x + 13.5}L{x + 15} {y + 21}Z" fill="#e9ddbf" stroke="#4a2a1c" stroke-width="0.8" />
        <path d="M{x + 12} {y + 10}L{x + 5} {y + 3}M{x + 12} {y + 10}L{x + 19} {y + 3}M{x + 12} {y + 10}L{x + 5} {y + 17}M{x + 12} {y + 10}L{x + 19} {y + 17}" stroke="#4a2a1c" stroke-width="1.6" stroke-linecap="round" />
      {:else if cell === 'P' || cell === 'H'}
        <rect x={x + 4} y={y + 10} width="16" height="10" fill="#d9cba6" stroke="#4a2a1c" stroke-width="0.9" />
        <path d="M{x + 4} {y + 10}V{y + 8}H{x + 7}V{y + 10}M{x + 10.5} {y + 10}V{y + 8}H{x + 13.5}V{y + 10}M{x + 17} {y + 10}V{y + 8}H{x + 20}V{y + 10}" fill="#d9cba6" stroke="#4a2a1c" stroke-width="0.9" />
        <path d="M{x + 12} {y + 8}V{y + 2}" stroke="#4a2a1c" stroke-width="1" />
        <path d="M{x + 12} {y + 2}L{x + 18} {y + 4}L{x + 12} {y + 6}Z" fill={cell === 'H' ? '#7a1f1f' : '#b08a3e'} />
      {:else if cell === 'K'}
        <path d="M{x + 6} {y + 21}V{y + 12}L{x + 12} {y + 7}L{x + 18} {y + 12}V{y + 21}Z" fill="#e9ddbf" stroke="#4a2a1c" stroke-width="0.9" />
        <path d="M{x + 12} {y + 7}V{y + 1}M{x + 9.5} {y + 3.5}H{x + 14.5}" stroke="#4a2a1c" stroke-width="1.2" />
      {:else if cell === 'x'}
        <path d="M{x + 5} {y + 18}L{x + 9} {y + 11}L{x + 12} {y + 16}L{x + 15} {y + 9}L{x + 19} {y + 18}Z" fill="#4a3a2a" opacity=".8" />
      {/if}
      {#if selectable(r, c) && !occupied.has(`${r}:${c}`)}
        <rect x={x + 1} y={y + 1} width={C - 2} height={C - 2} rx="3" class="pick" role="button" tabindex="-1"
          aria-label="Feld {r + 1}/{c + 1}" onclick={() => onpick?.(r, c)} onkeydown={(e) => e.key === 'Enter' && onpick?.(r, c)} />
      {/if}
    {/each}
  {/each}

  <!-- Grenzlinie -->
  <polyline points={borderLine} fill="none" stroke="#7a1f1f" stroke-width="3" stroke-dasharray="7 4" />

  <!-- Einheiten als Regimentsblöcke -->
  {#each units as u (u.id)}
    <g class="unit side{u.side}" class:dead={u.dead} style:transform="translate({u.col * C}px, {u.row * C}px)"
      onclick={() => selectableSide >= 0 && onpick?.(u.row, u.col)} role="presentation">
      <rect x="2" y="6" width={C - 4} height={C - 12} class="block" class:militia={u.kind === 'militia'} />
      {#if u.kind === 'cavalry'}<path d="M3 {C - 7}L{C - 3} 7" class="mark" />
      {:else if u.kind === 'infantry'}<path d="M2 11.5H{C - 2}" class="mark" /><path d="M5 6V0L11 2L5 4" class="flag" />
      {:else if u.kind === 'artillery'}<circle cx="9" cy="13" r="3" class="mark" /><path d="M10 12L{C - 2} 7" class="barrel" />
      {/if}
      {#if u.dead}<path d="M4 4L{C - 4} {C - 4}M{C - 4} 4L4 {C - 4}" class="x" />{/if}
    </g>
  {/each}

  {#if flash}
    <circle cx={flash.col * C + C / 2} cy={flash.row * C + C / 2} r={flash.kind === 'shot' ? 13 : 11} class="flash {flash.kind}" />
  {/if}
  <rect x="-2" y="-2" width={W + 4} height={H + 4} fill="none" stroke="#2b2118" stroke-width="2.5" />
</svg>

<style>
  svg { width: 100%; height: auto; display: block; }
  .side { font-family: var(--font-label); font-size: 15px; font-weight: 700; letter-spacing: 1.5px; }
  .side.a { fill: #3e5a28; }
  .side.d { fill: #1f3a5f; }
  /* Freie Felder der eigenen Seite: zart aufgehellt, beim Überfahren golden */
  .pick { fill: #fbf6ea; opacity: .5; cursor: pointer; }
  .pick:hover { opacity: .95; fill: #f3dc94; }
  .unit { transition: transform 60ms linear, opacity 400ms; cursor: default; }
  .block { stroke: #1e1710; stroke-width: 1.3; }
  .side0 .block { fill: #4e6b33; }
  .side1 .block { fill: #3f6a99; }
  .block.militia { fill: #eae0c6 !important; stroke-dasharray: 3 2; }
  .side0 .block.militia { stroke: #3e5a28; }
  .side1 .block.militia { stroke: #1f3a5f; }
  .mark { stroke: #f6eedb; stroke-width: 1.6; fill: none; }
  .flag { fill: #f6eedb; stroke: #1e1710; stroke-width: .8; }
  .barrel { stroke: #1e1710; stroke-width: 3; stroke-linecap: round; }
  .unit.dead { opacity: .35; }
  .x { stroke: #7a1f1f; stroke-width: 2.2; stroke-linecap: round; }
  .flash { fill: none; stroke-width: 3; animation: blink .35s ease-out; }
  .flash.hit { stroke: #7a1f1f; }
  .flash.fire { stroke: #d9822b; fill: #d9822b44; }
  .flash.shot { stroke: #2b2118; fill: #2b211833; }
  @keyframes blink { from { r: 4; } }
  @media (prefers-reduced-motion: reduce) { .unit { transition: none; } .flash { animation: none; } }
</style>
