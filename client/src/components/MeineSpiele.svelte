<script lang="ts">
  // „Meine Spiele“ auf der Startseite: alle Plätze, die dieser Browser kennt, mit Stand,
  // Hinweis „Sie sind am Zug“, Fortsetzen und Entfernen aus der Liste.
  import { app } from '../lib/state.svelte';
  import { forgetGame, requestGames, resumeGame } from '../lib/net';
  import { COUNTRIES, title, restzeit } from '../lib/format';

  let now = $state(Math.floor(Date.now() / 1000));

  // Alle 20 s neu abfragen (Zugzeiten laufen), Uhr jede Sekunde
  $effect(() => {
    const poll = setInterval(requestGames, 20_000);
    const tick = setInterval(() => (now = Math.floor(Date.now() / 1000)), 1000);
    return () => { clearInterval(poll); clearInterval(tick); };
  });

  const games = $derived(app.myGames ?? []);
  const left = (d: number) => restzeit(d - now);
</script>

{#if games.length}
  <section class="card mine" aria-labelledby="mine-title">
    <h2 id="mine-title" class="display">Meine Spiele</h2>
    <ul>
      {#each games as g (g.gameId)}
        <li class:turn={g.myTurn}>
          <div class="info">
            {#if !g.found}
              <strong>Spiel {g.gameId}</strong><small>Gibt es nicht mehr.</small>
            {:else if g.pending}
              <strong>Spiel {g.gameId}</strong><small>Ihre Beitrittsanfrage wartet auf den Gastgeber.</small>
            {:else if !g.valid}
              <strong>Spiel {g.gameId}</strong><small>Ihr Platz ist hier nicht mehr gültig.</small>
            {:else}
              <strong>{title(g.title ?? 1, !!g.female)} {g.name} von {COUNTRIES[g.country ?? 0]}</strong>
              <small>
                {g.gameId} ·
                {#if g.over}beendet{g.winner ? `, ${g.winner} wurde Kaiser` : ''}
                {:else if !g.started}wartet auf den Start{g.host ? ' (Sie sind Gastgeber)' : ''}
                {:else if !g.alive}Ihr Herrscher ist verstorben
                {:else}Anno {g.year}{/if}
                · {(g.players ?? []).length} Herrscher
              </small>
              {#if g.myTurn}<span class="badge-turn">Sie sind am Zug{g.deadline ? ` · noch ${left(g.deadline)}` : ''}</span>{/if}
            {/if}
          </div>
          <div class="acts">
            {#if g.found && (g.valid || g.pending)}
              <button class="btn" class:btn-primary={g.myTurn} type="button" onclick={() => resumeGame(g.gameId)}>
                {g.pending ? 'Warten' : g.over ? 'Ansehen' : 'Fortsetzen'}
              </button>
            {/if}
            <button class="btn ghost" type="button" aria-label="Spiel {g.gameId} aus der Liste entfernen" title="Aus der Liste entfernen"
              onclick={() => forgetGame(g.gameId)}>✕</button>
          </div>
        </li>
      {/each}
    </ul>
    <p class="hint">Die Liste kennt nur dieser Browser. Entfernen löscht nicht Ihren Platz im Spiel.</p>
  </section>
{/if}

<style>
  .mine { width: min(520px, 100%); display: flex; flex-direction: column; gap: 8px; }
  .mine h2 { font-size: 30px; }
  ul { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
  li { display: flex; align-items: center; gap: 10px; padding: 6px 8px; border: 1px solid var(--line); border-radius: 6px; background: #fbf6ea; }
  li.turn { border-color: var(--blood); box-shadow: inset 3px 0 0 var(--blood); }
  .info { flex: 1; min-width: 0; display: flex; flex-direction: column; line-height: 1.3; }
  small { color: var(--sepia); font-size: 14px; }
  .badge-turn { align-self: flex-start; margin-top: 2px; font-family: var(--font-label); font-size: 12px; font-weight: 700; color: var(--blood); }
  .acts { display: flex; gap: 6px; flex-shrink: 0; }
  .ghost { min-width: 40px; padding: 0 10px; }
  .hint { color: var(--sepia); font-size: 14px; margin: 0; }
</style>
