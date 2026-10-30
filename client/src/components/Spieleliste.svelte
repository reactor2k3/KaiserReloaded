<script lang="ts">
  // Öffentliche Spieleliste (GET /api/games): alle laufenden Spiele, beitretbare zuerst.
  // „Anfragen“ übernimmt den Code ins Beitrittsformular; der Gastgeber entscheidet dann.
  import { COUNTRIES, zugzeit } from '../lib/format';
  import type { GameListing } from '../lib/types';

  let { onJoin }: { onJoin: (gameId: string) => void } = $props();
  let games = $state<GameListing[] | null>(null);

  async function load() {
    try {
      const res = await fetch('/api/games');
      if (res.ok) games = (await res.json()).games;
    } catch {
      /* Server nicht erreichbar: Verbindungsanzeige meldet das */
    }
  }
  // Alle 15 s auffrischen, damit neue Spiele und volle Tische ohne Neuladen sichtbar werden
  $effect(() => {
    load();
    const t = setInterval(load, 15_000);
    return () => clearInterval(t);
  });
</script>

{#if games}
  <section class="card liste" aria-labelledby="liste-title">
    <h2 id="liste-title" class="display">Laufende Spiele</h2>
    {#if games.length === 0}
      <p class="hint">Gerade läuft kein Spiel. Legen Sie doch eines an!</p>
    {:else}
      <ul>
        {#each games as g (g.gameId)}
          <li class:zu={!g.joinable}>
            <div class="info">
              <strong>{g.host} von {COUNTRIES[g.hostCountry]}</strong>
              <small>
                {g.gameId} · {g.started ? `Anno ${g.year}` : 'wartet auf den Start'} · {g.players}/{g.maxPlayers} Herrscher ·
                {g.mode === 'async' ? 'asynchron' : 'live'}{g.phaseSeconds ? `, ${zugzeit(g.phaseSeconds)} je Phase` : ''}{g.classic ? ' · Original-Regeln' : ''}
              </small>
              {#if !g.joinable}<small class="zu-text">Kein Beitritt mehr möglich</small>{/if}
            </div>
            {#if g.joinable}
              <button class="btn" type="button" onclick={() => onJoin(g.gameId)}>Anfragen</button>
            {/if}
          </li>
        {/each}
      </ul>
      <p class="hint">Beitreten geht, bis der erste Herrscher befördert wurde. Der Gastgeber muss zustimmen.</p>
    {/if}
  </section>
{/if}

<style>
  /* Gleiche Listenoptik wie „Meine Spiele“ */
  .liste ul { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
  .liste li { display: flex; align-items: center; gap: 8px; padding: 6px 4px; border-bottom: 1px solid var(--line, rgba(0, 0, 0, .1)); }
  .liste .info { display: flex; flex-direction: column; flex: 1; min-width: 0; }
  .liste small { color: var(--sepia); }
  .liste li.zu { opacity: .65; }
  .liste .zu-text { font-style: italic; }
  .hint { color: var(--sepia); font-size: 14px; margin: 6px 0 0; }
</style>
