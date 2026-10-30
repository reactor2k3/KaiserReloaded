<script lang="ts">
  // Finanzen (Kaiser II): Bankkredit nach Bonität, Reichs- und Herrscheranleihen,
  // Spenden und Kredite an Mitspieler. Als Modal, weil es eine Übersicht über alle Herrscher braucht.
  // Geschäfte gehen in der Markt- und der Bauphase (Server: engine/finanzen.go).
  import { app } from '../lib/state.svelte';
  import { act } from '../lib/net';
  import { num, title } from '../lib/format';
  import Modal from './Modal.svelte';
  import Auswahl from './Auswahl.svelte';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  // Konstanten wie in engine/finanzen.go
  const REICH_PRICE = 250, REICH_YIELD = 18, LOAN_RATE = 8;

  let loanQty = $state(0);
  let bondQty = $state(10);
  let peer = $state(-1);
  let peerQty = $state(0);
  let peerRate = $state(5);
  let peerYears = $state(3);

  const self = $derived(app.self);
  const p = $derived(app.self?.player);
  const fin = $derived(app.self?.finance);
  const me = $derived(app.seat);
  const canAct = $derived(self?.phase === 'market' || self?.phase === 'build');
  const andere = $derived(app.players.filter((x) => x.seat !== me && x.alive));
  const name = (s: number) => app.players[s]?.name ?? '?';
  // Kredite, die man vergeben hat, und eigene Schulden bei Mitspielern
  const verliehen = $derived((fin?.loans ?? []).filter((l) => l.lender === me));
  const schulden = $derived((fin?.loans ?? []).filter((l) => l.borrower === me));
  const peerOpen = $derived(verliehen.some((l) => l.borrower === peer));
  const peerValid = $derived(peer >= 0 && peerQty > 0 && peerQty <= Math.max(0, p?.treasury ?? 0));
</script>

<Modal bind:open title="Finanzen" wide>
  {#if p && fin}
    {#if !canAct}<p class="hint warn">Geldgeschäfte gehen nur in der Markt- und der Bauphase.</p>{/if}

    <!-- Bank: Kredit bis zur Bonität, günstiger als der Dispo, blockiert aber den Titelaufstieg -->
    <h3 class="label">Bank · Kasse {num(p.treasury)} Taler</h3>
    <p class="hint">Bankkredit <strong class="num">{num(p.loan ?? 0)}</strong> von höchstens <strong class="num">{num(fin.bonitaet)}</strong> Taler (Bonität),
      {LOAN_RATE} % Zins im Jahr. Mit offenem Kredit gibt es keinen neuen Titel.</p>
    <div class="row">
      <label class="feld grow"><input type="number" min="0" step="1000" bind:value={loanQty} placeholder="0" /><span class="feld-label">Betrag in Taler</span></label>
      <button class="btn sm" type="button" disabled={!canAct || loanQty <= 0 || (p.loan ?? 0) + loanQty > fin.bonitaet}
        onclick={() => act({ type: 'takeLoan', qty: loanQty })}>Leihen</button>
      <button class="btn sm" type="button" disabled={!canAct || loanQty <= 0 || loanQty > (p.loan ?? 0) || loanQty > p.treasury}
        onclick={() => act({ type: 'repayLoan', qty: loanQty })}>Tilgen</button>
      <button class="btn sm" type="button" disabled={!canAct || !(p.loan ?? 0) || p.treasury <= 0}
        onclick={() => act({ type: 'repayLoan', qty: Math.min(p.loan ?? 0, p.treasury) })}>Alles tilgen</button>
    </div>

    <!-- Anleihen: Reichsanleihe sicher, Herrscheranleihen zahlen 1 ‰ der Staatseinnahmen des Herausgebers -->
    <h3 class="label">Anleihen</h3>
    <div class="row">
      <label class="feld"><input type="number" min="1" step="1" bind:value={bondQty} placeholder="0" /><span class="feld-label">Stück</span></label>
      <p class="hint grow">Herrscheranleihen bringen je Stück 1 ‰ der Staatseinnahmen (Steuern, Märkte, Mühlen) des Herausgebers, Rückgabe zu 90 %. Stirbt er, sind sie wertlos.</p>
    </div>
    <!-- Tabelle darf auf schmalen Schirmen seitlich scrollen (nur der Container, nie die Seite) -->
    <div class="tabelle"><table class="anleihen">
      <thead><tr><th>Herausgeber</th><th class="num">Preis</th><th class="num">Ertrag</th><th class="num">frei</th><th class="num">Besitz</th><th></th></tr></thead>
      <tbody>
        <tr>
          <td>Reichsanleihe</td><td class="num">{REICH_PRICE}</td><td class="num">{REICH_YIELD}</td><td class="num">∞</td><td class="num">{p.reichBonds ?? 0}</td>
          <td class="akt">
            <button class="btn sm" type="button" disabled={!canAct || bondQty <= 0 || bondQty * REICH_PRICE > p.treasury} onclick={() => act({ type: 'buyReichBonds', qty: bondQty })}>Kaufen</button>
            <button class="btn sm" type="button" disabled={!canAct || bondQty <= 0 || bondQty > (p.reichBonds ?? 0)} onclick={() => act({ type: 'sellReichBonds', qty: bondQty })}>Verkaufen</button>
          </td>
        </tr>
        {#each andere as x (x.seat)}
          <tr>
            <td>{title(x.title, x.female)} {x.name}</td>
            <td class="num">{num(fin.bondPrice[x.seat] ?? 0)}</td><td class="num">{num(fin.bondDiv[x.seat] ?? 0)}</td>
            <td class="num">{fin.bondFree[x.seat] ?? 0}</td><td class="num">{fin.holdings[x.seat] ?? 0}</td>
            <td class="akt">
              <button class="btn sm" type="button" disabled={!canAct || bondQty <= 0 || bondQty > (fin.bondFree[x.seat] ?? 0) || bondQty * (fin.bondPrice[x.seat] ?? 0) > p.treasury}
                onclick={() => act({ type: 'buyBonds', target: x.seat, qty: bondQty })}>Kaufen</button>
              <button class="btn sm" type="button" disabled={!canAct || bondQty <= 0 || bondQty > (fin.holdings[x.seat] ?? 0)}
                onclick={() => act({ type: 'sellBonds', target: x.seat, qty: bondQty })}>Verkaufen</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table></div>
    {#if fin.issuedOwn > 0}
      <p class="hint">Andere besitzen {fin.issuedOwn} Ihrer Anleihen. Nach der Steuer zahlen Sie je Stück etwa {num(fin.dividendPA)} Taler aus.</p>
    {/if}

    <!-- Mitspieler: Spende oder Kredit mit Zins und Laufzeit (gleiche Raten ab Jahresende) -->
    <h3 class="label">Hilfe für Mitspieler</h3>
    <div class="row">
      <Auswahl label="Empfänger" float="Empfänger" bind:value={peer}
        options={[{ value: -1, label: 'Herrscher wählen …' }, ...andere.map((x) => ({ value: x.seat, label: x.name }))]} />
      <label class="feld grow"><input type="number" min="0" step="500" bind:value={peerQty} placeholder="0" /><span class="feld-label">Taler</span></label>
      <label class="feld klein"><input type="number" min="0" max="15" step="1" bind:value={peerRate} /><span class="feld-label">Zins %</span></label>
      <label class="feld klein"><input type="number" min="1" max="10" step="1" bind:value={peerYears} /><span class="feld-label">Jahre</span></label>
    </div>
    <div class="row end">
      <button class="btn sm" type="button" disabled={!canAct || !peerValid} onclick={() => act({ type: 'gift', target: peer, qty: peerQty })}>Schenken</button>
      <button class="btn sm" type="button" disabled={!canAct || !peerValid || peerOpen || peerRate < 0 || peerRate > 15 || peerYears < 1 || peerYears > 10}
        title={peerOpen ? 'Der letzte Kredit an diesen Herrscher ist noch offen' : 'Zins 0 bis 15 %, Laufzeit 1 bis 10 Jahre'}
        onclick={() => act({ type: 'lend', target: peer, qty: peerQty, rate: peerRate, years: peerYears })}>Kredit geben</button>
    </div>
    <p class="hint">Verschenkt und verliehen wird nur echtes Guthaben. Der Schuldner zahlt jedes Jahr Zins und Rate und kann jederzeit ganz zurückzahlen.</p>
    {#if verliehen.length || schulden.length}
      <ul class="kredite">
        {#each verliehen as l (l.borrower)}
          <li><span>An {name(l.borrower)}: noch <span class="num">{num(l.amount)}</span> T. · {l.rate} % · {l.years} {l.years === 1 ? 'Rate' : 'Raten'}</span></li>
        {/each}
        {#each schulden as l (l.lender)}
          <li><span>Von {name(l.lender)}: noch <span class="num">{num(l.amount)}</span> T. · {l.rate} % · {l.years} {l.years === 1 ? 'Rate' : 'Raten'}</span>
            <button class="btn sm" type="button" disabled={!canAct || l.amount > p.treasury} onclick={() => act({ type: 'repayPeer', target: l.lender })}>Zurückzahlen</button></li>
        {/each}
      </ul>
    {/if}
  {:else}
    <p class="hint">Mit den Original-Regeln gibt es keine Geldgeschäfte.</p>
  {/if}
  {#snippet actions()}
    <button class="btn btn-primary" type="button" onclick={() => (open = false)}>Schließen</button>
  {/snippet}
</Modal>

<style>
  h3.label { margin: 10px 0 4px; }
  .row { display: flex; align-items: center; gap: 6px; margin: 4px 0; }
  .row.end { justify-content: flex-end; }
  .grow { flex: 1; }
  .feld.klein { width: 118px; } /* Platz für die Kartuschen-Beschriftung */
  .hint { color: var(--sepia); margin: 2px 0; font-size: 14px; }
  .warn { color: var(--blood); }
  .btn.sm { min-height: 32px; padding: 0 10px; font-size: 12px; }
  .anleihen { width: 100%; border-collapse: collapse; font-size: 14px; }
  .anleihen th { font-family: var(--font-label); font-size: 11px; letter-spacing: .06em; text-transform: uppercase; color: var(--sepia); text-align: left; font-weight: normal; }
  .anleihen th, .anleihen td { padding: 3px 4px; border-bottom: 1px solid var(--line); }
  .anleihen .num { text-align: right; }
  .akt { text-align: right; white-space: nowrap; }
  .tabelle { max-width: 100%; overflow-x: auto; }
  /* Handy: Kauf/Verkauf untereinander, schmalere Spalten */
  @media (max-width: 600px) {
    .anleihen { font-size: 13px; }
    .akt { display: flex; flex-direction: column; gap: 4px; align-items: flex-end; }
    .row { flex-wrap: wrap; }
  }
  .kredite { list-style: none; margin: 4px 0; padding: 0; font-size: 14px; }
  .kredite li { display: flex; align-items: center; gap: 8px; justify-content: space-between; padding: 2px 0; }
</style>
