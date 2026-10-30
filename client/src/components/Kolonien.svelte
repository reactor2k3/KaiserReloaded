<script lang="ts">
  // Flotte und Kolonien (Kaiser II): Werft, Kolonialflotten je Kontinent, Kolonialland und Seekrieg.
  // Als Modal, weil drei Kontinente und die Flotten der Gegner nicht in die Aktionsspalte passen.
  // Alle Aktionen nur in der Bauphase (Server: engine/kolonien.go).
  import { app } from '../lib/state.svelte';
  import { act } from '../lib/net';
  import { num, CONTINENTS, CONTINENT_GOODS, TRADER_COST, WARSHIP_COST, COLONY_PRICE, COVER_PER_TRADER } from '../lib/format';
  import Modal from './Modal.svelte';
  import Auswahl from './Auswahl.svelte';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  const self = $derived(app.self);
  const p = $derived(app.self?.player);
  const me = $derived(app.seat);
  const buildPhase = $derived(self?.phase === 'build');
  const colonies = $derived(p?.colonies ?? []);

  // Schiffe im Heimathafen = alle minus die in Kolonialflotten
  const homeT = $derived((p?.traders ?? 0) - colonies.reduce((s, c) => s + c.traders, 0));
  const homeW = $derived((p?.warships ?? 0) - colonies.reduce((s, c) => s + c.warships, 0));

  let shipQty = $state(1);
  // Eingaben je Kontinent; beim Öffnen mit dem aktuellen Stand belegt
  let fleetT = $state([0, 0, 0]);
  let fleetW = $state([0, 0, 0]);
  let parcels = $state([1, 1, 1]);
  $effect(() => {
    if (open) {
      fleetT = colonies.map((c) => c.traders);
      fleetW = colonies.map((c) => c.warships);
    }
  });
  const changed = (i: number) => fleetT[i] !== colonies[i]?.traders || fleetW[i] !== colonies[i]?.warships;
  const fleetOk = (i: number) => fleetT[i] >= 0 && fleetW[i] >= 0 && fleetT[i] <= homeT + colonies[i].traders
    && fleetW[i] <= homeW + colonies[i].warships && (fleetT[i] > 0 || fleetW[i] === 0);

  // Seekrieg: Ziel, Kontinent, eingesetzte Kriegsschiffe
  let target = $state(-1);
  let cont = $state(0);
  let sendW = $state(0);
  const gegner = $derived(app.players.filter((x) => x.seat !== me && x.alive));
  const isAlly = (s: number) => app.pacts.some((x) => (x.a === me && x.b === s) || (x.b === me && x.a === s));
  const feind = $derived(target >= 0 ? app.players[target]?.colonies?.[cont] : undefined);
</script>

<Modal bind:open title="Flotte und Kolonien" wide>
  {#if p}
    {#if !buildPhase}<p class="hint warn">Schiffe bauen, Flotten zusammenstellen und entsenden geht nur in der Bauphase.</p>{/if}

    <!-- Werft: Preis und Jahresgrenze wie im Original (Einwohner / 200) -->
    <h3 class="label">Werft · Kasse {num(p.treasury)} Taler</h3>
    <p class="hint">Im Hafen: <strong>{homeT}</strong> Handels- und <strong>{homeW}</strong> Kriegsschiffe (gesamt {p.traders ?? 0} und {p.warships ?? 0}).
      Unterhalt 400 bzw. 200 Taler im Jahr. Ihre Werften bauen dieses Jahr noch {self?.shipLimit ?? 0} Schiffe.</p>
    <div class="row">
      <label class="feld klein"><input type="number" min="1" step="1" bind:value={shipQty} /><span class="feld-label">Anzahl</span></label>
      <button class="btn sm" type="button" disabled={!buildPhase || shipQty <= 0 || shipQty > (self?.shipLimit ?? 0)}
        onclick={() => act({ type: 'buildShip', unit: 'trader', qty: shipQty })}>Handelsschiff · {num(TRADER_COST * Math.max(1, shipQty))}</button>
      <button class="btn sm" type="button" disabled={!buildPhase || shipQty <= 0 || shipQty > (self?.shipLimit ?? 0)}
        onclick={() => act({ type: 'buildShip', unit: 'warship', qty: shipQty })}>Kriegsschiff · {num(WARSHIP_COST * Math.max(1, shipQty))}</button>
    </div>

    <!-- Kolonien: ein Handelsschiff bedient 4000 km², Ertrag etwa die Hälfte der bedienten Fläche in Talern -->
    <h3 class="label">Kolonien</h3>
    <div class="kontinente">
      {#each colonies as c, i (i)}
        <section class="kontinent">
          <h4>{CONTINENTS[i]} <small>{CONTINENT_GOODS[i]}</small></h4>
          <dl>
            <dt>Fläche</dt><dd class="num">{num(c.area)} km²</dd>
            <dt>bedient</dt><dd class="num">{num(Math.min(c.area, COVER_PER_TRADER * c.traders))} km²</dd>
            <dt>Ertrag</dt><dd class="num">≈ {num(Math.min(c.area, COVER_PER_TRADER * c.traders) / 2)} T.</dd>
          </dl>
          <div class="row">
            <label class="feld"><input type="number" min="0" step="1" bind:value={fleetT[i]} /><span class="feld-label">Händler</span></label>
            <label class="feld"><input type="number" min="0" step="1" bind:value={fleetW[i]} /><span class="feld-label">Kriegs.</span></label>
          </div>
          <button class="btn sm voll" type="button" disabled={!buildPhase || !changed(i) || !fleetOk(i)}
            title="Schiffe kommen aus dem Heimathafen und kehren dorthin zurück"
            onclick={() => act({ type: 'fleet', continent: i, qty: fleetT[i], warships: fleetW[i] })}>Flotte setzen</button>
          <div class="row">
            <label class="feld"><input type="number" min="1" step="1" bind:value={parcels[i]} /><span class="feld-label">× 1.000 km²</span></label>
            <button class="btn sm" type="button" disabled={!buildPhase || c.traders === 0 || parcels[i] <= 0}
              title={c.traders === 0 ? 'Erst eine Handelsflotte entsenden' : `${num(COLONY_PRICE)} Taler je 1.000 km²`}
              onclick={() => act({ type: 'colonize', continent: i, qty: parcels[i] })}>Land · {num(COLONY_PRICE * Math.max(1, parcels[i]))}</button>
          </div>
        </section>
      {/each}
    </div>

    <!-- Seekrieg gegen eine fremde Kolonialflotte, einmal im Jahr, nicht gegen Verbündete -->
    <h3 class="label">Kriegsflotte entsenden</h3>
    {#if gegner.length}
      <div class="row">
        <Auswahl label="Gegner" float="Gegen" bind:value={target}
          options={[{ value: -1, label: 'Herrscher wählen …' }, ...gegner.map((x) => ({ value: x.seat, label: x.name + (isAlly(x.seat) ? ' (verbündet)' : ''), disabled: isAlly(x.seat) }))]} />
        <Auswahl label="Kontinent" float="Flotte" bind:value={cont} options={CONTINENTS.map((n, i) => ({ value: i, label: n }))} />
        <label class="feld klein"><input type="number" min="1" max={homeW} step="1" bind:value={sendW} /><span class="feld-label">Kriegs.</span></label>
        <button class="btn sm" type="button" disabled={!buildPhase || target < 0 || !!p.year.navalAttack || sendW <= 0 || sendW > homeW || !feind || (feind.traders + feind.warships) === 0}
          onclick={() => act({ type: 'navalAttack', target, continent: cont, warships: sendW })}>Entsenden</button>
      </div>
      <p class="hint">{#if target >= 0}{app.players[target]?.name}s {CONTINENTS[cont]}-Flotte: {feind?.traders ?? 0} Handels- und {feind?.warships ?? 0} Kriegsschiffe.{/if}
        {#if p.year.navalAttack} Dieses Jahr ist schon eine Flotte ausgelaufen.{:else} Es kämpfen nur Kriegsschiffe aus dem Heimathafen ({homeW}).{/if}</p>
    {:else}
      <p class="hint">Kein Gegner auf See.</p>
    {/if}
  {/if}
  {#snippet actions()}
    <button class="btn btn-primary" type="button" onclick={() => (open = false)}>Schließen</button>
  {/snippet}
</Modal>

<style>
  h3.label { margin: 10px 0 4px; }
  .row { display: flex; align-items: center; gap: 6px; margin: 4px 0; }
  .feld.klein { width: 118px; }
  .hint { color: var(--sepia); margin: 2px 0; font-size: 14px; }
  .warn { color: var(--blood); }
  .btn.sm { min-height: 32px; padding: 0 10px; font-size: 12px; }
  .btn.voll { width: 100%; }
  /* Drei Kontinente nebeneinander, auf schmalen Schirmen untereinander */
  .kontinente { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 6px; }
  .kontinent { border: 1px solid var(--line); padding: 6px; display: flex; flex-direction: column; gap: 4px; }
  .kontinent h4 { margin: 0; font-family: var(--font-display, inherit); font-size: 17px; }
  .kontinent h4 small { color: var(--sepia); font-size: 12px; font-weight: normal; }
  .kontinent dl { display: grid; grid-template-columns: auto 1fr; gap: 0 8px; margin: 0; font-size: 14px; }
  .kontinent dd { margin: 0; text-align: right; }
  .kontinent .feld { flex: 1; min-width: 0; }
</style>
