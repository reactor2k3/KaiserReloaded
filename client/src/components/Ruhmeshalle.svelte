<script lang="ts">
  // Ruhmeshalle auf der Startseite: jüngste Partien mit gekröntem Kaiser und Bestenliste
  // der menschlichen Herrscher (zusammengefasst nach Namen, da es keine Konten gibt).
  import { COUNTRIES, title } from '../lib/format';

  interface Player { name: string; country: number; female: boolean; title: number; bot: boolean; winner: boolean }
  interface Entry { id: string; ended: number; year: number; winner: string; players: Player[] }
  interface Stat { name: string; games: number; crowns: number; bestTitle: number }

  let recent = $state<Entry[]>([]);
  let best = $state<Stat[]>([]);
  let failed = $state(false);

  $effect(() => {
    fetch('/api/ruhmeshalle')
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((d) => { recent = d.recent ?? []; best = d.best ?? []; })
      .catch(() => (failed = true));
  });

  const date = (s: number) => new Date(s * 1000).toLocaleDateString('de-DE');
</script>

<section class="card hall" aria-labelledby="hall-title">
  <h2 id="hall-title" class="display">Ruhmeshalle</h2>
  {#if failed}
    <p class="hint">Die Ruhmeshalle ist gerade nicht erreichbar.</p>
  {:else if recent.length === 0}
    <p class="hint">Noch wurde keine Partie beendet. Wer wird der erste Kaiser?</p>
  {:else}
    <h3 class="label">Die letzten Partien</h3>
    <ol class="recent">
      {#each recent as e (e.id)}
        {@const w = e.players.find((p) => p.winner)}
        <li>
          <span class="crown" aria-hidden="true">{w ? '♛' : '✝'}</span>
          <span>
            {#if w}<strong>{title(9, w.female)} {w.name}</strong> von {COUNTRIES[w.country]}, gekrönt {e.year}
            {:else}Ohne Kaiser zu Ende gegangen ({e.year}){/if}
            <small>{e.players.filter((p) => !p.winner).map((p) => `${p.name}${p.bot ? ' (Computer)' : ''}`).join(', ')} · {date(e.ended)}</small>
          </span>
        </li>
      {/each}
    </ol>
    {#if best.length}
      <h3 class="label">Bestenliste</h3>
      <table>
        <thead><tr><th scope="col">Name</th><th scope="col">Kronen</th><th scope="col">Bester Titel</th><th scope="col">Partien</th></tr></thead>
        <tbody>
          {#each best as s (s.name)}
            <tr><th scope="row">{s.name}</th><td class="num">{s.crowns}</td><td>{title(s.bestTitle, false)}</td><td class="num">{s.games}</td></tr>
          {/each}
        </tbody>
      </table>
    {/if}
  {/if}
</section>

<style>
  .hall { width: min(520px, 100%); display: flex; flex-direction: column; gap: 8px; }
  .hall h2 { font-size: 30px; }
  h3 { margin: 6px 0 0; }
  .hint { color: var(--sepia); margin: 0; }
  .recent { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
  .recent li { display: flex; gap: 10px; align-items: flex-start; line-height: 1.3; }
  .recent small { display: block; color: var(--sepia); font-size: 14px; }
  .crown { color: var(--gold); font-size: 20px; width: 22px; text-align: center; }
  table { border-collapse: collapse; width: 100%; font-size: 15px; }
  th, td { padding: 4px 6px; border-bottom: 1px solid var(--line); text-align: left; }
  th[scope=row] { font-weight: 600; }
  td.num { text-align: right; }
</style>
