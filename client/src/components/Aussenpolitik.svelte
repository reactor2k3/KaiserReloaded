<script lang="ts">
  // Außenpolitik (Kaiser II): Verteidigungsabkommen anbieten, annehmen, ablehnen, kündigen.
  // Als Modal, weil es eine Übersicht über alle Herrscher braucht; Aktionen nur in der Bauphase.
  import { app } from '../lib/state.svelte';
  import { act } from '../lib/net';
  import { COUNTRIES, title } from '../lib/format';
  import Modal from './Modal.svelte';
  import Wappen from './Wappen.svelte';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  const me = $derived(app.seat);
  const allied = (s: number) => app.pacts.some((x) => (x.a === me && x.b === s) || (x.b === me && x.a === s));
  const offersMe = $derived(app.self?.pactOffers ?? []);
  const andere = $derived(app.players.filter((p) => p.seat !== me && p.alive));
  const buildPhase = $derived(app.self?.phase === 'build');
  // Bündnisse der anderen untereinander (für die Einschätzung von Kriegsrisiken)
  const fremde = $derived(app.pacts.filter((x) => x.a !== me && x.b !== me));
  const name = (s: number) => app.players[s]?.name ?? '?';
</script>

<Modal bind:open title="Außenpolitik">
  <p class="hint">Verbündete greifen einander nicht an. Wird einer angegriffen, stehen ihm seine Verbündeten
    automatisch bei. Wer kündigt, darf den bisherigen Partner erst im nächsten Jahr angreifen.</p>
  {#if !buildPhase}<p class="hint warn">Anbieten und kündigen geht nur in der Bauphase, Angebote annehmen jederzeit.</p>{/if}
  <ul class="liste">
    {#each andere as p (p.seat)}
      <li>
        <Wappen arms={p.arms} country={p.country} size={30} label="Wappen von {p.name}" />
        <span class="wer">{title(p.title, p.female)} {p.name} <small>von {COUNTRIES[p.country]}</small></span>
        {#if allied(p.seat)}
          <span class="status ok">verbündet</span>
          <button class="btn sm" type="button" disabled={!buildPhase} onclick={() => act({ type: 'cancelPact', target: p.seat })}>Kündigen</button>
        {:else if offersMe.includes(p.seat)}
          <span class="status offer">bietet an</span>
          <button class="btn sm btn-primary" type="button" onclick={() => act({ type: 'acceptPact', target: p.seat })}>Annehmen</button>
          <button class="btn sm" type="button" onclick={() => act({ type: 'declinePact', target: p.seat })}>Ablehnen</button>
        {:else if (app.self?.pactSent ?? []).includes(p.seat)}
          <span class="status">angeboten, wartet auf Antwort</span>
        {:else}
          <button class="btn sm" type="button" disabled={!buildPhase} onclick={() => act({ type: 'proposePact', target: p.seat })}>Abkommen anbieten</button>
        {/if}
      </li>
    {/each}
  </ul>
  {#if fremde.length}
    <p class="hint">Weitere Bündnisse im Reich: {fremde.map((x) => `${name(x.a)} & ${name(x.b)}`).join(', ')}.</p>
  {/if}
  {#snippet actions()}
    <button class="btn btn-primary" type="button" onclick={() => (open = false)}>Schließen</button>
  {/snippet}
</Modal>

<style>
  .liste { list-style: none; margin: 8px 0; padding: 0; display: flex; flex-direction: column; gap: 6px; min-width: min(560px, 80vw); }
  /* Zeilen dürfen umbrechen (Handy): Name füllt, Status und Knöpfe rutschen notfalls darunter */
  .liste li { display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; padding: 4px 0; border-bottom: 1px solid var(--line); }
  .wer { flex: 1 1 160px; min-width: 0; }
  .wer small { color: var(--sepia); }
  .status { font-family: var(--font-label); font-size: 11px; letter-spacing: .08em; text-transform: uppercase; }
  .status.ok { color: var(--good); }
  .status.offer { color: var(--blood); }
  .btn.sm { min-height: 32px; padding: 0 10px; font-size: 12px; }
  .hint { color: var(--sepia); margin: 0; font-size: 15px; }
  .warn { color: var(--blood); }
</style>
