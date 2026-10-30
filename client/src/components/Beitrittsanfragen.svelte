<script lang="ts">
  // Beitrittsanfragen für den Gastgeber, in Lobby und laufendem Spiel gleichermaßen sichtbar.
  // Eigener <dialog> mit showModal(): Er liegt in der obersten Ebene über anderen offenen
  // Spielfenstern (Ereignisse, Schlachten), die sonst alles dahinter blockieren würden.
  import { app } from '../lib/state.svelte';
  import { decideJoin } from '../lib/net';
  import { COUNTRIES } from '../lib/format';

  let dlg: HTMLDialogElement | undefined = $state();
  // „Später“ blendet die aktuellen Anfragen aus, bis eine neue dazukommt
  let later = $state(new Set<string>());
  const shown = $derived(app.host ? app.joinRequests.filter((q) => !later.has(q.id)) : []);

  $effect(() => {
    if (!dlg) return;
    if (shown.length && !dlg.open) dlg.showModal();
    else if (!shown.length && dlg.open) dlg.close();
  });

  function postpone() {
    later = new Set([...later, ...shown.map((q) => q.id)]);
  }
</script>

<dialog bind:this={dlg} class="anfragen card" aria-label="Beitrittsanfragen" oncancel={(e) => { e.preventDefault(); postpone(); }}>
  <h2 class="label">Beitrittsanfragen</h2>
  {#each shown as q (q.id)}
    <div class="anfrage">
      <span>{q.female ? 'Frau' : 'Herr'} <strong>{q.name}</strong> von {COUNTRIES[q.country]} möchte beitreten.</span>
      <div class="knoepfe">
        <button class="btn btn-primary" type="button" onclick={() => decideJoin(q.id, true)}>Zulassen</button>
        <button class="btn" type="button" onclick={() => decideJoin(q.id, false)}>Ablehnen</button>
      </div>
    </div>
  {/each}
  <button class="btn ghost spaeter" type="button" onclick={postpone}>Später entscheiden</button>
</dialog>

<style>
  .anfragen { max-width: min(400px, calc(100vw - 24px)); padding: 12px; }
  .anfragen[open] { display: flex; flex-direction: column; gap: 10px; }
  .anfragen::backdrop { background: rgba(0, 0, 0, .35); }
  .anfragen h2 { margin: 0; }
  .anfrage { display: flex; flex-direction: column; gap: 6px; }
  .knoepfe { display: flex; gap: 8px; justify-content: flex-end; }
  .spaeter { align-self: flex-start; }
</style>
