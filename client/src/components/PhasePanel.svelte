<script lang="ts">
  // Aktionsspalte neben der Karte für die aktuelle Phase: Markt, Volk versorgen, Steuern, Bauen,
  // Diplomatie. Die Spalte bleibt stehen, beim Phasenwechsel tauscht nur der Inhalt.
  // Ausnahme Aufstellung: das Schlachtfeld braucht Breite und öffnet als großes Modal.
  import Modal from './Modal.svelte';
  import DeployBoard from './DeployBoard.svelte';
  import Auswahl from './Auswahl.svelte';
  import Aussenpolitik from './Aussenpolitik.svelte';
  import Finanzen from './Finanzen.svelte';
  import Kolonien from './Kolonien.svelte';
  import { app } from '../lib/state.svelte';
  import { act } from '../lib/net';
  import { COUNTRIES, FEST_COST, GENDARM_COST, GENDARM_PER, JUSTICE, MOOD_NAMES, OPPO_COST, WEATHER, num, title } from '../lib/format';
  import type { Placement } from '../lib/types';

  let { waitingFor = [] }: { waitingFor?: string[] } = $props();

  const self = $derived(app.self!);
  const p = $derived(self.player);
  const titles: Record<string, string> = {
    market: 'Ernte und Markt', feed: 'Volk versorgen', taxes: 'Steuern und Justiz', build: 'Staatskäufe',
    diplomacy: 'Krieg im Reich', deploy: 'Aufstellung', done: 'Jahr beendet',
  };

  // Krieg: Ziel wählen, dann Bestätigung im eigenen Modal
  let warTarget = $state(-1);
  let confirmWar = $state(false);
  let aussen = $state(false);
  let finanzen = $state(false);
  let kolonien = $state(false);
  // Verbündete des gewählten Gegners (Warnung vor der Kriegserklärung, wie in Kaiser II)
  const warAllies = $derived(warTarget < 0 ? [] : app.pacts.filter((x) => x.a === warTarget || x.b === warTarget)
    .map((x) => (x.a === warTarget ? x.b : x.a)).filter((s) => s !== app.seat));
  const isAlly = (s: number) => app.pacts.some((x) => (x.a === app.seat && x.b === s) || (x.b === app.seat && x.a === s));
  const enemies = $derived(app.players.filter((x) => x.alive && x.seat !== app.seat));
  function who(seat: number) {
    const x = app.players[seat];
    return x ? `${title(x.title, x.female)} ${x.name} von ${COUNTRIES[x.country]}` : '';
  }

  // Aufstellung: lokale Auswahl, wird erst mit „bestätigen“ gesendet
  let placements = $state<Placement[]>([]);
  let deployOpen = $state(false);
  $effect(() => {
    if (self.phase !== 'deploy') placements = [];
    deployOpen = self.phase === 'deploy';
  });

  // Markt
  let grainQty = $state(0);
  let landQty = $state(0);
  const grainCost = $derived(Math.floor((grainQty * p.year.grainPrice) / 1000));
  const landCost = $derived(Math.floor(landQty * p.year.landPrice));

  // Versorgung: Vorauswahl = Bedarf, begrenzt auf die erlaubte Spanne
  let feedChoice = $state<'min' | 'need' | 'max' | 'custom'>('need');
  let feedCustom = $state(0);
  let kornInfo = $state(false);
  const feedAmount = $derived(
    feedChoice === 'min' ? self.feedMin :
    feedChoice === 'max' ? self.feedMax :
    feedChoice === 'need' ? Math.max(self.feedMin, Math.min(self.feedMax, p.year.grainNeed)) :
    feedCustom);
  const feedValid = $derived(feedAmount >= self.feedMin && feedAmount <= self.feedMax);

  // Steuern: lokale Regler, Änderung wird verzögert an den Server geschickt (Vorschau kommt zurück)
  let customs = $state(0), vat = $state(0), incomeTax = $state(0), justice = $state(2);
  let taxTimer: ReturnType<typeof setTimeout> | undefined;
  $effect(() => {
    if (self.phase === 'taxes') {
      customs = p.customs; vat = p.vat; incomeTax = p.incomeTax; justice = p.justice;
    }
  });
  const taxesChanged = $derived(customs !== p.customs || vat !== p.vat || incomeTax !== p.incomeTax || justice !== p.justice);
  function scheduleTaxes() {
    clearTimeout(taxTimer);
    taxTimer = setTimeout(() => act({ type: 'setTaxes', customs, vat, incomeTax, justice }), 300);
  }

  // Innenpolitik: Stimmung und Gendarmen
  const mood = $derived(p.moodInit ? p.mood : 5);
  const gendMax = $derived(Math.floor(p.population / GENDARM_PER));
  const gendFree = $derived(gendMax - (p.gendarmes ?? 0));

  // Militär: Rekruten oder Söldner
  let mercenary = $state(false);
  const costs = $derived(mercenary ? self.mercCosts : self.unitCosts);

  // Direktkauf: feste Mengen mit einem Klick (Korn in Maß, Land in Hektar)
  const QUICK_GRAIN = [5000, 10000, 20000];
  const QUICK_LAND = [1000, 2000, 5000];
  const grainLeft = $derived(3 * p.year.grainNeed - p.year.grainBought); // dieses Jahr noch kaufbar

  function trade(type: string, qty: number) {
    if (qty > 0) act({ type, qty });
  }
</script>

<section class="card panel" aria-labelledby="phase-title">
  <h2 id="phase-title" class="display">{titles[self.phase]}</h2>
  <!-- Inhalt scrollt, Kopf und Aktionsknöpfe bleiben stehen -->
  <div class="body">
  {#if self.phase === 'market'}
    <!-- Wetter und Zugang zu den Geldgeschäften (Kaiser II) in einer Zeile -->
    <div class="mil-head">
      <p class="weather">{WEATHER[p.year.weather]}</p>
      {#if self.finance}<button class="linkbtn" type="button" onclick={() => (finanzen = true)}>Finanzen</button>{/if}
    </div>
    {#if p.year.risk}
      <!-- Offene Risiko-Entscheidung: Modal wieder öffnen (unbeantwortet gilt „Nein“) -->
      <button class="btn sm risk" type="button" onclick={() => (app.risikoZeigen = true)}>Entscheidung offen: {app.risiko?.title ?? 'Risiko'}</button>
    {/if}
    <!-- Kompakte Kennzahlen: zwei Spalten, Wert rechtsbündig -->
    <dl class="facts">
      <dt>Kornpreis</dt><dd class="num">{num(p.year.grainPrice)} T./1000 Maß</dd>
      <dt>Landpreis</dt><dd class="num">{p.year.landPrice.toLocaleString('de-DE')} T./ha</dd>
      <dt>Bedarf</dt><dd class="num">{num(p.year.grainNeed)} Maß</dd>
      <dt>Reserve</dt><dd class="num">{num(p.grain)} Maß</dd>
    </dl>
    <!-- Handel je Ware: Menge, darunter Kaufen/Verkaufen und die Preise -->
    <div class="trade">
      <div class="row">
        <label class="feld grow"><input type="number" min="0" step="100" bind:value={grainQty} placeholder="0" /><span class="feld-label">Korn in Maß</span>
        <span class="stepper">
          <button type="button" tabindex="-1" aria-label="Korn um 100 erhöhen" onclick={() => (grainQty = Math.max(0, (grainQty || 0) + 100))}><svg viewBox="0 0 12 8" aria-hidden="true"><path d="M1 7L6 2L11 7" /></svg></button>
          <button type="button" tabindex="-1" aria-label="Korn um 100 verringern" disabled={grainQty <= 0} onclick={() => (grainQty = Math.max(0, (grainQty || 0) - 100))}><svg viewBox="0 0 12 8" aria-hidden="true"><path d="M1 1L6 6L11 1" /></svg></button>
        </span></label>
        <div class="buysell">
          <button class="btn sm" type="button" disabled={grainQty <= 0} onclick={() => trade('buyGrain', grainQty)}>Kaufen</button>
          <button class="btn sm" type="button" disabled={grainQty <= 0 || grainQty > p.grain} onclick={() => trade('sellGrain', grainQty)}>Verkaufen</button>
        </div>
      </div>
      <p class="hint" title="Taler für Kauf und Verkauf; höchstens noch so viel Korn ist dieses Jahr kaufbar">Kauf {num(grainCost)} · Verkauf {num(Math.floor((grainQty * p.year.grainPrice) / 1111))} T. · max. {num(grainLeft)}</p>
      <div class="quick" role="group" aria-label="Korn direkt kaufen">
        <span class="label">Direkt</span>
        {#each QUICK_GRAIN as q}
          <button class="btn sm" type="button" disabled={q > grainLeft} onclick={() => trade('buyGrain', q)}
            title="{num(q)} Maß für {num(Math.floor((q * p.year.grainPrice) / 1000))} Taler kaufen">{num(q)}</button>
        {/each}
      </div>
    </div>
    <div class="trade">
      <div class="row">
        <label class="feld grow"><input type="number" min="0" step="100" bind:value={landQty} placeholder="0" /><span class="feld-label">Land in Hektar</span>
        <span class="stepper">
          <button type="button" tabindex="-1" aria-label="Land um 100 erhöhen" onclick={() => (landQty = Math.max(0, (landQty || 0) + 100))}><svg viewBox="0 0 12 8" aria-hidden="true"><path d="M1 7L6 2L11 7" /></svg></button>
          <button type="button" tabindex="-1" aria-label="Land um 100 verringern" disabled={landQty <= 0} onclick={() => (landQty = Math.max(0, (landQty || 0) - 100))}><svg viewBox="0 0 12 8" aria-hidden="true"><path d="M1 1L6 6L11 1" /></svg></button>
        </span></label>
        <div class="buysell">
          <button class="btn sm" type="button" disabled={landQty <= 0 || (self.freeLand >= 0 && landQty > self.freeLand)} onclick={() => trade('buyLand', landQty)}>Kaufen</button>
          <button class="btn sm" type="button" disabled={landQty <= 0 || landQty >= p.land} onclick={() => trade('sellLand', landQty)}>Verkaufen</button>
        </div>
      </div>
      <p class="hint">Kauf {num(landCost)} · Verkauf {num(Math.floor(landCost * 0.9))} T.{#if self.freeLand >= 0}{` · frei ${num(self.freeLand)} ha`}{/if}</p>
      <div class="quick" role="group" aria-label="Land direkt kaufen">
        <span class="label">Direkt</span>
        {#each QUICK_LAND as q}
          <button class="btn sm" type="button" disabled={self.freeLand >= 0 && q > self.freeLand} onclick={() => trade('buyLand', q)}
            title="{num(q)} ha für {num(Math.floor(q * p.year.landPrice))} Taler kaufen">{num(q)}</button>
        {/each}
      </div>
    </div>
  {:else if self.phase === 'feed'}
    <!-- Kurz die Regel, ausführlich im Modal „Wie wirkt Korn?“ -->
    <p class="hint">Bedarf <strong class="num">{num(p.year.grainNeed)} Maß</strong>. Verteilen dürfen Sie 20 bis 80 % der Reserve ({num(p.grain)} Maß).
      Ab etwa 110 bis 150 % des Bedarfs kommen Einwanderer.
      <button class="linkbtn" type="button" onclick={() => (kornInfo = true)}>Wie wirkt Korn?</button></p>
    <fieldset class="choices">
      <legend class="sr">Menge</legend>
      <label><input type="radio" bind:group={feedChoice} value="min" /> Minimum <span class="num">{num(self.feedMin)}</span></label>
      <label><input type="radio" bind:group={feedChoice} value="need" /> Bedarf <span class="num">{num(Math.max(self.feedMin, Math.min(self.feedMax, p.year.grainNeed)))}</span></label>
      <label><input type="radio" bind:group={feedChoice} value="max" /> Maximum <span class="num">{num(self.feedMax)}</span></label>
      <label><input type="radio" bind:group={feedChoice} value="custom" onchange={() => { if (!feedCustom) feedCustom = Math.max(self.feedMin, Math.min(self.feedMax, p.year.grainNeed)); }} /> Eigene Menge</label>
      <!-- Eigene Menge im gleichen Feld-Design wie im Markt (mit eigenen Pfeilen, Schritt 100) -->
      {#if feedChoice === 'custom'}
        <label class="feld custom">
          <input type="number" min={self.feedMin} max={self.feedMax} step="100" bind:value={feedCustom} placeholder="0" />
          <span class="feld-label">Menge in Maß</span>
          <span class="stepper">
            <button type="button" tabindex="-1" aria-label="Menge um 100 erhöhen" disabled={feedCustom >= self.feedMax} onclick={() => (feedCustom = Math.min(self.feedMax, (feedCustom || 0) + 100))}><svg viewBox="0 0 12 8" aria-hidden="true"><path d="M1 7L6 2L11 7" /></svg></button>
            <button type="button" tabindex="-1" aria-label="Menge um 100 verringern" disabled={feedCustom <= self.feedMin} onclick={() => (feedCustom = Math.max(self.feedMin, (feedCustom || 0) - 100))}><svg viewBox="0 0 12 8" aria-hidden="true"><path d="M1 1L6 6L11 1" /></svg></button>
          </span>
        </label>
      {/if}
    </fieldset>
    {#if !feedValid}
      <p class="error">{feedAmount > self.feedMax
        ? `Höchstens 80 % der Reserve: ${num(self.feedMax)} Maß. Mehr geht nur mit Zukauf im Markt.`
        : `Mindestens 20 % der Reserve: ${num(self.feedMin)} Maß.`}</p>
    {/if}
  {:else if self.phase === 'taxes'}
    {#each [{ label: 'Zoll', get: () => customs, set: (v: number) => (customs = v), val: self.taxes.customs },
            { label: 'Mehrwertsteuer', get: () => vat, set: (v: number) => (vat = v), val: self.taxes.vat },
            { label: 'Einkommensteuer', get: () => incomeTax, set: (v: number) => (incomeTax = v), val: self.taxes.incomeTax }] as t}
      <label class="tax">
        <span>{t.label}</span>
        <input type="range" min="0" max="99" value={t.get()} oninput={(e) => { t.set(+e.currentTarget.value); scheduleTaxes(); }} />
        <span class="num pct">{t.get()} %</span>
        <span class="num money">{num(t.val)}</span>
      </label>
    {/each}
    <label class="tax">
      <span>Justiz</span>
      <Auswahl label="Justiz" value={justice} onchange={(v) => { justice = v; scheduleTaxes(); }}
        options={JUSTICE.slice(1).map((j, i) => ({ value: i + 1, label: j }))} />
      <span class="pct"></span>
      <span class="num money">{num(self.taxes.justice)}</span>
    </label>
    <p class="total">Staatseinnahmen <strong class="num">{num(self.taxes.total)} Taler</strong>{#if taxesChanged} <span class="hint">(wird berechnet …)</span>{/if}</p>
    {#if customs + vat + incomeTax > 90 || justice > 2}<p class="warn">Hohe Steuern oder harte Justiz treiben Einwohner aus dem Land.</p>{/if}
    <!-- Innenpolitik (Kaiser II): Stimmung pflegen und Gendarmen gegen Aufstände -->
    <h3 class="label" title="Stimmung {mood}: {MOOD_NAMES[mood]}">Innenpolitik · Stimmung {mood}</h3>
    <div class="grid">
      <button class="btn" type="button" disabled={mood >= 10} title="Stimmung +1" onclick={() => act({ type: 'fest' })}>Volksfest · {num(FEST_COST * p.title)}</button>
      <button class="btn" type="button" disabled={!!p.year.opposition} title="Opposition verhaften: Stimmung +3, einmal im Jahr" onclick={() => act({ type: 'opposition' })}>Opposition · {num(OPPO_COST * p.title)}</button>
      <button class="btn" type="button" disabled={gendFree <= 0} title="Je {num(GENDARM_COST)} Taler, Sold 20 Taler im Jahr; schützen vor Revolution und Attentat" onclick={() => act({ type: 'gendarm', qty: Math.min(10, gendFree) })}>+{Math.min(10, Math.max(gendFree, 1))} Gendarmen</button>
      <span class="hint gend">Gendarmen {p.gendarmes ?? 0} / {gendMax}</span>
    </div>
  {:else if self.phase === 'build'}
    <div class="mil-head">
      <h3 class="label">Bauten · Kasse {num(p.treasury)} Taler</h3>
      {#if self.finance}<span class="links">
        <button class="linkbtn" type="button" onclick={() => (kolonien = true)}>Kolonien</button>
        <button class="linkbtn" type="button" onclick={() => (finanzen = true)}>Finanzen</button>
      </span>{/if}
    </div>
    <div class="grid">
      <button class="btn" type="button" onclick={() => act({ type: 'build', building: 'market' })}>Marktplatz · 1.000</button>
      <button class="btn" type="button" onclick={() => act({ type: 'build', building: 'mill' })}>Kornmühle · 2.000</button>
      <button class="btn" type="button" onclick={() => act({ type: 'build', building: 'palace' })}>Palastteil · 5.000</button>
      <button class="btn" type="button" onclick={() => act({ type: 'build', building: 'cathedral' })}>Kathedralteil · 9.000</button>
    </div>
    <!-- Überschrift und Söldner-Schalter in einer Zeile, spart Höhe -->
    <div class="mil-head">
      <!-- Bestand und Schlagkraft im Hinweis, damit die Spalte ohne Scrollen auskommt -->
      <h3 class="label" title="Schlagkraft aus Ausbildung und Material · Bestand: {p.cavalry} Kav., {p.artillery} Art., {p.infantry} Inf., {p.militia} Miliz">Militär · {Math.round((1 + 0.03 * (p.training ?? 0) + 0.05 * (p.equipment ?? 0)) * 100)} %</h3>
      <label class="merc" title="Teurer und höherer Sold, kostet aber keine Einwohner"><input type="checkbox" bind:checked={mercenary} /> Söldner</label>
    </div>
    <div class="grid">
      <button class="btn" type="button" onclick={() => act({ type: 'recruit', unit: 'cavalry', mercenary })}>Kavallerie · {num(costs.Cavalry)}</button>
      <button class="btn" type="button" onclick={() => act({ type: 'recruit', unit: 'artillery', mercenary })}>Artillerie · {num(costs.Artillery)}</button>
      <button class="btn" type="button" onclick={() => act({ type: 'recruit', unit: 'infantry', mercenary })}>Infanterie · {num(costs.Infantry)}</button>
      <button class="btn" type="button" onclick={() => act({ type: 'maneuver' })}>Manöver · {num(4 * p.soldiers + 1000)}</button>
      <!-- Kaiser II: Ausbildung und Material verstärken jede Einheit im Gefecht -->
      <button class="btn" type="button" disabled={(p.training ?? 0) >= 10} title="Ausbildung {p.training ?? 0}/10: +3 % Schlagkraft je Stufe, verfällt ohne Übung"
        onclick={() => act({ type: 'train' })}>Ausbildung {p.training ?? 0} · {num(((p.training ?? 0) + 1) * 400 + 2 * p.soldiers)}</button>
      <button class="btn" type="button" disabled={(p.equipment ?? 0) >= 10} title="Material {p.equipment ?? 0}/10: +5 % Schlagkraft je Stufe"
        onclick={() => act({ type: 'equip' })}>Material {p.equipment ?? 0} · {num(((p.equipment ?? 0) + 1) * 600 + 4 * p.soldiers)}</button>
    </div>
    <!-- Krieg und Außenpolitik: Bündnisse im Modal, Angebote als Hinweis am Knopf -->
    <div class="mil-head">
      <h3 class="label">Krieg führen</h3>
      <button class="linkbtn" type="button" onclick={() => (aussen = true)}>Außenpolitik{(self.pactOffers ?? []).length ? ` (${(self.pactOffers ?? []).length} Angebot)` : ''}</button>
    </div>
    {#if p.title < 2}
      <p class="hint">Krieg erst ab dem Titel Baron.</p>
    {:else}
      <div class="row">
        <div class="grow"><Auswahl label="Gegner wählen" float="Gegner" bind:value={warTarget}
          options={[{ value: -1, label: 'Gegner wählen …' }, ...enemies.map((e) => ({ value: e.seat, label: who(e.seat) + (isAlly(e.seat) ? ' (verbündet)' : ''), disabled: isAlly(e.seat) }))]} /></div>
        <button class="btn" type="button" disabled={warTarget < 0} onclick={() => (confirmWar = true)}>Krieg erklären …</button>
      </div>
    {/if}
  {:else if self.phase === 'diplomacy' && self.war}
    <p class="lead">{who(self.war.attacker)} erklärt {who(self.war.defender)} den Krieg. Wie verhalten Sie sich?</p>
    <div class="grid">
      <button class="btn" type="button" onclick={() => act({ type: 'stance', stance: 'neutral' })}>Neutral bleiben</button>
      <button class="btn" type="button" onclick={() => act({ type: 'stance', stance: 'passage' })}>Durchmarsch gewähren</button>
      <button class="btn" type="button" onclick={() => act({ type: 'stance', stance: 'helpAttacker' })}>Durchmarsch und Hilfe für {app.players[self.war.attacker]?.name}</button>
      <button class="btn" type="button" onclick={() => act({ type: 'stance', stance: 'helpDefender' })}>Hilfe für {app.players[self.war.defender]?.name}</button>
    </div>
    <p class="hint">Helfer leihen Kavallerie, Artillerie und Infanterie. Verlorene Leihtruppen werden bezahlt.</p>
  {:else if self.phase === 'deploy'}
    <p>Die Schlacht steht bevor. Stellen Sie Ihre Truppen auf dem Schlachtfeld auf.</p>
  {:else}
    {#if p.joined && p.joined >= app.year}
      <!-- Mitten im Jahr beigetreten: Einstieg erst zum nächsten Jahr -->
      <p>Willkommen im Reich! Sie steigen mit dem nächsten Jahr ein, sobald alle Herrscher das laufende Jahr beendet haben.</p>
    {:else}
      <p>Ihr Jahr ist abgeschlossen. Sobald alle Herrscher fertig sind, beginnt das nächste Jahr.</p>
    {/if}
    {#if waitingFor.length}<p class="hint">Es fehlen noch: {waitingFor.join(', ')}.</p>{/if}
  {/if}
  </div>

  <div class="actions">
    {#if self.phase === 'market'}
      <button class="btn btn-primary" type="button" onclick={() => act({ type: 'next' })}>Weiter zum Volk</button>
    {:else if self.phase === 'feed'}
      <button class="btn btn-primary" type="button" disabled={!feedValid} onclick={() => act({ type: 'feed', qty: feedAmount })}>Korn verteilen</button>
    {:else if self.phase === 'taxes'}
      <button class="btn btn-primary" type="button" disabled={taxesChanged} onclick={() => act({ type: 'next' })}>Steuern eintreiben</button>
    {:else if self.phase === 'build'}
      <button class="btn btn-primary" type="button" onclick={() => act({ type: 'endTurn' })}>Jahr beenden</button>
    {:else if self.phase === 'deploy'}
      <button class="btn btn-primary" type="button" onclick={() => (deployOpen = true)}>Schlachtfeld öffnen</button>
    {/if}
  </div>
</section>

<!-- Aufstellung braucht die ganze Breite: eigenes Modal -->
<Modal bind:open={deployOpen} title="Aufstellung" wide>
  {#if self.phase === 'deploy' && self.war?.field}
    <p>{self.war.side === 0 ? 'Ihr Heer greift an' : 'Sie verteidigen Ihr Land'} – stellen Sie Ihre Truppen auf Ihrer Seite der roten Grenzlinie auf.</p>
    <DeployBoard field={self.war.field} side={self.war.side} counts={self.war.counts ?? {}} bind:placements
      names={[app.players[self.war.attacker]?.name ?? 'Angreifer', app.players[self.war.defender]?.name ?? 'Verteidiger']} />
  {/if}
  {#snippet actions()}
    <button class="btn" type="button" onclick={() => act({ type: 'autoDeploy' })}>Feldherr stellt auf</button>
    <button class="btn btn-primary" type="button" onclick={() => act({ type: 'deploy', placements })}>Aufstellung bestätigen</button>
  {/snippet}
</Modal>

<!-- Hinweis-Modal zur Versorgung: Regeln und Wirkung mit den aktuellen Zahlen -->
<Modal bind:open={kornInfo} title="Wie wirkt Korn?">
  <p>Jedes Jahr verteilen Sie Korn aus Ihrer Reserve an das Volk. Wie viel, entscheidet über Hunger, Geburten und Einwanderer.</p>
  <table class="korn">
    <tbody>
      <tr><th scope="row">Weniger als der Bedarf</th><td>Hunger: Menschen sterben, je größer der Mangel, desto mehr.</td></tr>
      <tr><th scope="row">Bedarf ({num(p.year.grainNeed)} Maß)</th><td>Alle sind satt; es gibt Geburten, Adel und Klerus wachsen.</td></tr>
      <tr><th scope="row">110 bis 150 % ({num(p.year.grainNeed * 1.1)} bis {num(p.year.grainNeed * 1.5)} Maß)</th><td>Überfluss lockt Einwanderer, je mehr, desto mehr – höchstens 10 % mehr Einwohner im Jahr. Hohe Steuern bremsen den Zuzug.</td></tr>
    </tbody>
  </table>
  <p><strong>Die 80-%-Regel:</strong> Verteilt werden dürfen 20 bis 80 % der Reserve (bei Ihnen {num(self.feedMin)} bis {num(self.feedMax)} Maß). Der Rest ist Saatgut und Vorrat.</p>
  <p class="hint">Tipp: Wer viele Einwanderer will, kauft in der Marktphase Korn zu. Für 150 % des Bedarfs braucht die Reserve etwa {num(p.year.grainNeed * 1.5 / 0.8)} Maß. Mehr Einwohner brauchen aber auch mehr Land.</p>
  {#snippet actions()}
    <button class="btn btn-primary" type="button" onclick={() => (kornInfo = false)}>Verstanden</button>
  {/snippet}
</Modal>

<Aussenpolitik bind:open={aussen} />
<Finanzen bind:open={finanzen} />
<Kolonien bind:open={kolonien} />

<Modal bind:open={confirmWar} title="Krieg erklären?" variant="confirm">
  {#if warAllies.length}
    <p class="warn">Achtung: {who(warTarget)} ist verbündet mit {warAllies.map(who).join(', ')}. Sie riskieren einen Krieg gegen alle!</p>
  {/if}
  <p>Sie erklären <strong>{who(warTarget)}</strong> den Krieg. Die Schlacht findet am Ende des Jahres statt, Ihr Zug endet damit.</p>
  <p class="hint">Die übrigen Herrscher entscheiden über Durchmarsch und Hilfe. Je länger der Weg, desto geringer die Moral.</p>
  {#snippet actions()}
    <button class="btn" type="button" onclick={() => (confirmWar = false)}>Abbrechen</button>
    <button class="btn btn-primary" type="button" onclick={() => { act({ type: 'declareWar', target: warTarget }); confirmWar = false; }}>Krieg erklären</button>
  {/snippet}
</Modal>

<style>
  /* Spalte füllt die Höhe; Inhalte sind so knapp gehalten, dass ab 1280×720 nichts scrollt.
     Ratschläge stehen nicht hier, sondern in der Fußleiste und im Kanzler-Modal. */
  .panel { min-height: 0; min-width: 0; display: flex; flex-direction: column; gap: 8px; }
  .panel h2 { font-size: clamp(22px, 2vw, 28px); flex-shrink: 0; }
  .body { flex: 1; min-height: 0; overflow-x: hidden; overflow-y: auto; overscroll-behavior: contain; display: flex; flex-direction: column; gap: 10px; }
  .body > :global(*) { flex-shrink: 0; }
  .actions { flex-shrink: 0; display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; padding-top: 8px; border-top: 1px solid var(--line); }
  .actions:empty { display: none; }
  /* Schmal: Spalten liegen untereinander im scrollenden Hauptbereich, die Spalte wächst mit dem Inhalt */
  @media (max-width: 900px) { .body { flex: none; overflow: visible; } }
  .weather { font-size: 17px; font-weight: 600; margin: 0; line-height: 1.2; }
  .facts { display: grid; grid-template-columns: auto 1fr; gap: 0 12px; margin: 0; font-size: 15px; line-height: 1.35; padding: 3px 0; border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); }
  .facts dt { color: var(--sepia); }
  .facts dd { margin: 0; text-align: right; font-weight: 600; }
  .trade { display: flex; flex-direction: column; gap: 4px; margin-top: 4px; }
  /* Kaufen und Verkaufen nebeneinander unter dem Mengenfeld */
  .trade .row { flex-direction: column; align-items: stretch; gap: 4px; }
  /* Mengenfelder in der Spalte etwas flacher als sonst, damit der Markt ohne Scrollen passt */
  .trade :global(.feld input) { min-height: 46px; padding-top: 16px; }
  .trade .hint { font-size: 13px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .buysell { display: flex; gap: 6px; }
  .gend { align-self: center; text-align: center; }
  /* Hinweisknopf auf die offene Entscheidung: eine Zeile, nicht über die Spalte gestreckt */
  .btn.sm.risk { border-color: var(--blood); color: var(--blood); align-self: flex-start; flex: none; min-height: 36px; height: auto; white-space: normal; text-align: left; }
  .btn.sm { flex: 1; min-height: 30px; padding: 0 10px; font-size: 12px; }
  /* Direktkauf-Zeile: kleines Etikett, drei gleich breite Mengenknöpfe */
  .quick { display: flex; align-items: center; gap: 4px; }
  .quick .label { font-size: 11px; color: var(--sepia); margin-right: 2px; }
  .sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  fieldset { border: 1px solid var(--line); border-radius: 6px; padding: 10px 14px 14px; margin: 0; display: flex; flex-direction: column; gap: 8px; }
  /* Unauffälliger Textknopf für das Hinweis-Modal */
  .linkbtn { border: none; background: none; padding: 0; color: var(--blood); font: inherit; text-decoration: underline; text-underline-offset: 2px; cursor: pointer; }
  .korn { border-collapse: collapse; width: 100%; font-size: 16px; }
  .korn th, .korn td { text-align: left; vertical-align: top; padding: 6px 8px; border-bottom: 1px solid var(--line); }
  .korn th { width: 40%; font-weight: 600; }
  .choices { border: none; padding: 0; gap: 2px; position: relative; }
  .choices label { display: flex; align-items: center; gap: 10px; min-height: 30px; }
  .choices .num { margin-left: auto; }
  .choices .custom { margin-top: 10px; }
  .row { display: flex; gap: 6px; align-items: center; }
  .hint { color: var(--sepia); font-size: 14px; margin: 0; line-height: 1.3; }
  .error, .warn { color: var(--warn); margin: 0; }
  .tax { min-width: 0; display: grid; grid-template-columns: minmax(0, 1fr) 48px 64px; grid-template-areas: 'name pct money' 'ctl ctl ctl'; gap: 2px 10px; align-items: center; }
  .tax > :first-child { grid-area: name; }
  .tax > :global(input), .tax > :global(.auswahl) { grid-area: ctl; width: 100%; }
  .tax .pct { grid-area: pct; text-align: right; }
  .tax .money { grid-area: money; }
  .money { text-align: right; }
  .total { font-size: 19px; margin: 4px 0 0; }
  .grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 6px; }
  .grid .btn { min-height: 36px; padding: 0 6px; font-size: 12px; }
  h3 { margin: 4px 0 0; font-size: 12px; }
  .mil-head { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
  .mil-head h3 { margin: 0; }
  .links { display: flex; gap: 10px; }
  .merc { display: flex; align-items: center; gap: 8px; min-height: 32px; font-size: 15px; }
  .lead { font-size: 19px; margin: 0; }
  .row .grow { flex: 1; min-width: 0; }
</style>
