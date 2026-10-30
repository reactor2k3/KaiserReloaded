<script lang="ts">
  // Schlachtbericht als Ereignis-Modal: Wiedergabe der Schlacht mit Verlauf in Worten,
  // nach dem Ende die Bilanz in der rechten Spalte. Öffnet sich bei jeder neuen Schlacht.
  import Modal from './Modal.svelte';
  import BattleReplay from './BattleReplay.svelte';
  import { app, autopilotAn } from '../lib/state.svelte';
  import { num, title } from '../lib/format';
  import type { FieldUnit } from '../lib/types';

  let open = $state(false);
  let done = $state(false);
  let shown: unknown = null;
  $effect(() => {
    if (app.lastBattle && app.lastBattle !== shown) {
      shown = app.lastBattle;
      done = false;
      // Nach dem Neuladen nicht ungefragt erneut abspielen; mit Autopilot gar nicht
      open = !app.lastBattle.historic && !autopilotAn();
    }
  });

  const b = $derived(app.lastBattle);
  const who = (seat: number) => {
    const p = app.players[seat];
    return p ? `${title(p.title, p.female)} ${p.name}` : '';
  };
  const names = $derived<[string, string]>(b ? [app.players[b.attacker]?.name ?? 'Angreifer', app.players[b.defender]?.name ?? 'Verteidiger'] : ['', '']);
  const rows: [string, string][] = [['cavalry', 'Kavallerie'], ['artillery', 'Artillerie'], ['infantry', 'Infanterie'], ['militia', 'Miliz'], ['markets', 'Märkte'], ['mills', 'Mühlen']];
</script>

{#if b}
  <Modal bind:open title="Schlacht: {names[0]} gegen {names[1]}" variant="event" wide>
    {#key b}
      <BattleReplay field={b.field} units={b.units as FieldUnit[]} steps={b.steps as any} {names} bind:done>
        {#snippet result()}
          <p class="lead">
            {#if b.result.land > 0}<strong>{who(b.attacker)}</strong> erobert <strong class="num">{num(b.result.land)} Hektar</strong> von {who(b.defender)}.
            {:else if b.result.land < 0}<strong>{who(b.defender)}</strong> schlägt den Angriff zurück und gewinnt <strong class="num">{num(-b.result.land)} Hektar</strong>.
            {:else}Die Schlacht endet ohne Landgewinn.{/if}
          </p>
          <table>
            <thead><tr><th scope="col">Verluste</th><th scope="col">{names[0]}</th><th scope="col">{names[1]}</th></tr></thead>
            <tbody>
              {#each rows as [k, label]}
                <tr><th scope="row">{label}</th><td class="num">{(b.result.losses[0] as any)[k]}</td><td class="num">{(b.result.losses[1] as any)[k]}</td></tr>
              {/each}
              <tr><th scope="row">Kasse</th><td class="num">{num(b.result.treasuryLoss[0])}</td><td class="num">{num(b.result.treasuryLoss[1])}</td></tr>
              <tr><th scope="row">Einwohner</th><td class="num">{num(b.result.populationLoss[0])}</td><td class="num">{num(b.result.populationLoss[1])}</td></tr>
            </tbody>
          </table>
          {#if b.result.bonus}<p class="hint">Prämie für die Überlebenden: {num(b.result.bonus)} Taler.</p>{/if}
        {/snippet}
      </BattleReplay>
    {/key}
    {#snippet actions()}
      <button class="btn btn-primary" type="button" onclick={() => (open = false)}>{done ? 'Weiter' : 'Schließen'}</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .lead { font-size: 17px; margin: 0 0 8px; line-height: 1.35; }
  table { border-collapse: collapse; width: 100%; font-size: 15px; }
  th, td { padding: 4px 6px; border-bottom: 1px solid var(--line); text-align: right; }
  th[scope=row], thead th:first-child { text-align: left; font-weight: 500; }
  .hint { color: var(--sepia); margin: 6px 0 0; font-size: 14px; }
</style>
