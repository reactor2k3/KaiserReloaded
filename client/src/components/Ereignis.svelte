<script lang="ts">
  // Hinweis-Modals für Kaiser-II-Ereignisse:
  // 1. Risiko-Entscheidung (Ja/Nein) in der Marktphase, solange sie offen ist.
  // 2. Zufallsereignisse und Ausgänge von Entscheidungen, nacheinander aus der Warteschlange.
  import { app, autopilotAn } from '../lib/state.svelte';
  import { act } from '../lib/net';
  import { fazit } from '../lib/format';
  import Modal from './Modal.svelte';

  let { riskOpen = $bindable(false) }: { riskOpen?: boolean } = $props();

  // Offene Frage: nur wenn der Server sie noch als offen führt
  const frage = $derived(app.risiko && app.self?.player.year.risk === app.risiko.id && app.self?.phase === 'market' ? app.risiko : null);
  let gefragt = '';
  $effect(() => {
    // Neue Frage einmal von selbst öffnen
    if (frage && gefragt !== `${app.year}-${frage.id}`) {
      gefragt = `${app.year}-${frage.id}`;
      if (!autopilotAn()) riskOpen = true;
    }
    if (!frage) riskOpen = false;
    if (app.risikoZeigen) {
      app.risikoZeigen = false;
      if (frage) riskOpen = true;
    }
  });

  const aktuell = $derived(app.ereignisse[0] ?? null);
  let ereignisOpen = $state(false);
  // Nächstes Ereignis zeigen, sobald keine Risiko-Frage im Weg ist (nur öffnen, nie still schließen)
  $effect(() => { if (aktuell && !riskOpen && !autopilotAn()) ereignisOpen = true; });
  // Autopilot: Fenster schließen und Warteschlange leeren; die Risiko-Frage beantwortet der
  // Computer selbst, die Ereignisse stehen in der Chronik
  $effect(() => {
    if (!autopilotAn()) return;
    riskOpen = false;
    ereignisOpen = false;
    if (app.ereignisse.length) app.ereignisse.splice(0);
  });

  function antwort(ja: boolean) {
    act({ type: 'risk', stance: ja ? 'ja' : 'nein' });
    riskOpen = false;
  }
  function weiter() {
    app.ereignisse.shift();
  }
  const folgen = $derived(aktuell ? fazit(aktuell) : []);
</script>

{#if frage}
  <Modal bind:open={riskOpen} title={frage.title}>
    <div class="urkunde">
      <span class="kopf label">Risiko!</span>
      <p class="text">{frage.question}</p>
      <p class="hint">Bei „Ja“ steht der Ausgang auf Messers Schneide. „Nein“ ist meist folgenlos.</p>
    </div>
    {#snippet actions()}
      <button class="btn" type="button" onclick={() => antwort(false)}>Nein</button>
      <button class="btn btn-primary" type="button" onclick={() => antwort(true)}>Ja, wagen</button>
    {/snippet}
  </Modal>
{/if}

{#if aktuell}
  <Modal bind:open={ereignisOpen} title={aktuell.title} onclose={weiter}>
    <div class="urkunde">
      <span class="kopf label">{aktuell.risk ? 'Ausgang Ihrer Entscheidung' : 'Ereignis'}</span>
      <p class="text">{aktuell.text}</p>
      {#if folgen.length}
        <div class="fazit">
          <span class="label">Fazit</span>
          <ul>
            {#each folgen as f}<li class:plus={f.startsWith('+')} class:minus={f.startsWith('−')}>{f}</li>{/each}
          </ul>
        </div>
      {/if}
    </div>
    {#snippet actions()}
      <button class="btn btn-primary" type="button" onclick={() => (ereignisOpen = false)}>Zur Kenntnis genommen</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .urkunde { display: flex; flex-direction: column; gap: 10px; max-width: 520px; }
  .kopf { color: var(--blood); }
  /* Ereignistext wie eine Depesche: großzügig, mit Initiale */
  .text { margin: 0; font-size: 19px; line-height: 1.5; }
  .text::first-letter { font-family: var(--font-display); font-size: 2.4em; float: left; line-height: .9; padding: 4px 6px 0 0; color: var(--blood); }
  .hint { margin: 0; color: var(--sepia); font-size: 15px; }
  .fazit { border-top: 1px solid #c9a95c; padding-top: 8px; display: flex; gap: 12px; align-items: baseline; flex-wrap: wrap; }
  .fazit ul { list-style: none; margin: 0; padding: 0; display: flex; flex-wrap: wrap; gap: 4px 14px; }
  .fazit li { font-size: 18px; font-weight: 600; }
  .fazit .plus { color: var(--good); }
  .fazit .minus { color: var(--blood); }
</style>
