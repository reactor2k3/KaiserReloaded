<script lang="ts">
  // Hauptbildschirm: Kopfzeile mit Jahresphasen, Reichskarte, eigenes Reich, Chronik.
  // Wird einmal aufgebaut; alle Werte aktualisieren sich über den Zustand.
  import { untrack } from 'svelte';
  import { app, autopilotAn } from '../lib/state.svelte';
  import { COUNTRIES, MOOD_NAMES, PHASES, num, playerColor, restzeit, ruler, title } from '../lib/format';
  import ReichsKarte from './ReichsKarte.svelte';
  import PhasePanel from './PhasePanel.svelte';
  import BattleReport from './BattleReport.svelte';
  import ProfileDialog from './ProfileDialog.svelte';
  import Wappen from './Wappen.svelte';
  import { leaveGame, setAutopilot } from '../lib/net';
  import { cycleSound, setMusicAllowed, sound } from '../lib/audio.svelte';
  import Modal from './Modal.svelte';
  import HelpDialog from './HelpDialog.svelte';
  import Auswahl from './Auswahl.svelte';
  import Ratgeber from './Ratgeber.svelte';
  import SpielerKarte from './SpielerKarte.svelte';
  import Kroenung from './Kroenung.svelte';
  import Ereignis from './Ereignis.svelte';
  import { erlaubnisAnfragen, kannFragen, sieSindDran } from '../lib/benachrichtigung';

  const self = $derived(app.self!);
  const p = $derived(self.player);
  const phaseIndex = $derived(PHASES.findIndex((ph) => ph.id === self.phase));
  let profileOpen = $state(false);
  let helpOpen = $state(false);
  let leaveOpen = $state(false);
  let adviceOpen = $state(false);
  let kroenungOpen = $state(false);
  let riskOpen = $state(false);
  let now = $state(Math.floor(Date.now() / 1000));

  // Musik nur, solange die Partie läuft; beim Verlassen des Spielbildschirms aus
  $effect(() => {
    setMusicAllowed(!app.over);
    return () => setMusicAllowed(false);
  });

  // Asynchrones Spiel: melden, wenn eine neue eigene Phase beginnt (nach dem Warten auf andere)
  let letztePhase = '';
  $effect(() => {
    const ph = self.phase;
    if (app.settings?.mode === 'async' && letztePhase === 'done' && ph !== 'done') {
      sieSindDran(app.year, PHASES.find((x) => x.id === ph)?.label ?? '');
    }
    letztePhase = ph;
  });
  let fragen = $state(kannFragen());

  // Rat des Kanzlers automatisch zu Beginn jedes Schritts (Markt, Volk, Steuern, Bauen), nach
  // den Ereignisfenstern; der Rat hängt von der Phase ab.
  // Abschaltbar im Rat-Fenster; Vorgabe an, Wahl gilt für diesen Browser (localStorage).
  const RAT_KEY = 'kaiser.ratAuto';
  function loadRatAuto(): boolean {
    try {
      return localStorage.getItem(RAT_KEY) !== 'aus';
    } catch {
      return true;
    }
  }
  let ratAuto = $state(loadRatAuto());
  function setRatAuto(on: boolean) {
    ratAuto = on;
    try {
      localStorage.setItem(RAT_KEY, on ? 'an' : 'aus');
    } catch {
      /* privater Modus: Wahl gilt nur bis zum Neuladen */
    }
  }
  const RAT_PHASEN = ['market', 'feed', 'taxes', 'build'];
  let ratSchritt = untrack(() => `${app.year}/${self.phase}`); // Stand beim Betreten: Wiedereinstieg öffnet den Rat nicht
  let ratFaellig = $state(false);
  $effect(() => {
    const schritt = `${app.year}/${self.phase}`;
    if (schritt === ratSchritt) return;
    ratSchritt = schritt;
    if (ratAuto && !app.over && me?.alive && !autopilotAn() && RAT_PHASEN.includes(self.phase)) ratFaellig = true;
    else ratFaellig = false; // Schritt schon vorbei (z. B. Autopilot): alten Rat nicht mehr zeigen
  });
  // Warten, bis Ereignis-, Risiko- und Schlachtfenster geschlossen sind (alle sind <dialog>).
  // Die erste Prüfung kommt verzögert, weil Ereignisse kurz nach dem Jahreswechsel eintreffen.
  $effect(() => {
    if (!ratFaellig) return;
    let t: ReturnType<typeof setInterval> | undefined;
    const start = setTimeout(() => {
      t = setInterval(() => {
        if (autopilotAn()) {
          ratFaellig = false; // Autopilot inzwischen an: kein Rat nötig
        } else if (!document.querySelector('dialog[open]')) {
          ratFaellig = false;
          adviceOpen = true;
        }
      }, 400);
    }, 1200);
    return () => {
      clearTimeout(start);
      clearInterval(t);
    };
  });

  // Sekundentakt nur für die Zugzeit-Anzeige
  $effect(() => {
    const t = setInterval(() => (now = Math.floor(Date.now() / 1000)), 1000);
    return () => clearInterval(t);
  });

  const remaining = $derived(self.deadline ? Math.max(0, self.deadline - now) : null);
  const clock = $derived(remaining === null ? '' : restzeit(remaining));
  const me = $derived(app.players[app.seat]);
  const waitingFor = $derived(app.players.filter((x) => x.alive && x.phase !== 'done' && x.seat !== app.seat));
</script>

<div class="screen">
  <header class="top">
    <div class="brand"><span class="display logo">Kaiser</span><span class="label">Reloaded</span></div>
    <div class="year"><span class="label">Anno</span><span class="num">{app.year}</span></div>
    <nav aria-label="Phasen des Jahres">
      <ol>
        {#each PHASES.slice(0, 4) as ph, i}
          <li class:done={i < phaseIndex} class:current={i === phaseIndex} aria-current={i === phaseIndex ? 'step' : undefined}>{ph.label}</li>
        {/each}
      </ol>
    </nav>
    <div class="autopilot" class:on={me?.bot}>
      <span class="label" aria-hidden="true">Autopilot</span>
      <Auswahl dark label="Autopilot: Computer spielt für mich" value={me?.bot ?? ''} onchange={(v) => setAutopilot(v)}
        options={[{ value: '', label: 'Aus – ich spiele selbst' }, { value: 'kaufmann', label: 'Kaufmann' }, { value: 'feldherr', label: 'Feldherr' }]} />
    </div>
    {#if clock}<div class="clock num" class:urgent={remaining !== null && remaining < 15} aria-label="Verbleibende Zugzeit">{clock}</div>{/if}
    <ul class="players" class:viele={app.players.length > 6} aria-label="Mitspieler">
      {#each app.players as x (x.seat)}
        <!-- Knopf im Kreis: fokussierbar, damit die Spielerkarte auch per Tastatur erscheint -->
        <li style:background={playerColor(app.players, x.seat)} class:ready={x.phase === 'done'} class:me={x.seat === app.seat} class:dead={!x.alive}>
          <button class="chip" type="button" aria-label={ruler(x)} aria-describedby="spieler-{x.seat}">
            {#if x.avatar}<img src="/media/{x.avatar}" alt="" width="34" height="34" />{:else}{x.name.slice(0, 1).toUpperCase()}{/if}
          </button>
          <SpielerKarte p={x} me={x.seat === app.seat} id="spieler-{x.seat}" />
        </li>
      {/each}
    </ul>
    <div class="tools">
      <button class="tool" type="button" onclick={cycleSound}
        aria-label="Ton: {sound.level === 'off' ? 'aus' : sound.level === 'sfx' ? 'nur Geräusche' : 'Geräusche und Musik'} – klicken zum Umschalten"
        title={sound.level === 'off' ? 'Ton aus' : sound.level === 'sfx' ? 'Geräusche an' : 'Geräusche und Musik an'}>
        <svg aria-hidden="true" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M11 5L6 9H2v6h4l5 4z" fill="currentColor" />
          {#if sound.level === 'off'}<path d="M22 9l-6 6M16 9l6 6" />
          {:else}<path d="M15.5 8.5a5 5 0 0 1 0 7" />{#if sound.level === 'all'}<path d="M19 5a10 10 0 0 1 0 14" />{/if}{/if}
        </svg>
      </button>
      <button class="tool" type="button" aria-label="Hilfe" title="Hilfe" onclick={() => (helpOpen = true)}>?</button>
      <button class="tool" type="button" aria-label="Spiel verlassen oder neues Spiel" title="Spiel verlassen / Neues Spiel" onclick={() => (leaveOpen = true)}>
        <svg aria-hidden="true" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l-5-5 5-5M5 12h12"/></svg>
      </button>
    </div>
  </header>

  <main>
    <section class="card map" aria-label="Reichskarte">
      <h1 class="display">Das Heilige Römische Reich</h1>
      <div class="map-box"><ReichsKarte /></div>
    </section>

    <!-- Mittlere Spalte: Aktionen der laufenden Phase -->
    <PhasePanel waitingFor={waitingFor.map((x) => x.name)} />

    <aside>
      <section class="card ruler">
        <Wappen arms={p.arms} country={p.country} size={56} label="Ihr Wappen" />
        {#if p.avatar}<img class="avatar" src="/media/{p.avatar}" alt="Ihr Spielerbild" width="64" height="64" />{/if}
        <div class="who">
          <span class="label">{title(p.title, p.female)} von {COUNTRIES[p.country]}</span>
          <span class="display name">{p.name}</span>
          {#if p.motto}<span class="motto">„{p.motto}“</span>{/if}
          <button class="edit" type="button" onclick={() => (profileOpen = true)}>Profil &amp; Wappen bearbeiten</button>
        </div>
      </section>
      <section class="stats" aria-label="Kennzahlen">
        <div class="card"><span class="label">Land</span><span class="v num">{num(p.land)}</span><span class="u">Hektar</span></div>
        <div class="card"><span class="label">Staatskasse</span><span class="v num" class:neg={p.treasury < 0}>{num(p.treasury)}</span><span class="u">Taler</span></div>
        <div class="card"><span class="label">Einwohner</span><span class="v num">{num(p.population)}</span><span class="u">Seelen</span></div>
        <div class="card"><span class="label">Kornreserve</span><span class="v num">{num(p.grain)}</span><span class="u">Bedarf {num(p.year.grainNeed)}</span></div>
        <div class="card"><span class="label">Armee</span><span class="v num">{num(p.soldiers)}</span><span class="u">Moral {Math.round(p.morale * 100)} %</span></div>
        <!-- Volksstimmung (Kaiser II): Zahl 0–10 und Bezeichnung, rot ab „schlecht“ -->
        <div class="card" title="Volksstimmung 0–10: wirkt auf Steuern, bei 0–1 drohen Aufstände"><span class="label">Stimmung</span><span class="v num" class:neg={p.moodInit && p.mood <= 3}>{p.moodInit ? p.mood : 5}</span><span class="u">{MOOD_NAMES[p.moodInit ? p.mood : 5]}</span></div>
      </section>
      <section class="card buildings" aria-label="Bauten">
        <div><span class="v num">{p.markets}</span><span>Märkte</span></div>
        <div><span class="v num">{p.mills}</span><span>Mühlen</span></div>
        <div><span class="v num">{p.palace}/16</span><span>Palast</span></div>
        <div><span class="v num">{p.cathedral}/14</span><span>Kathedrale</span></div>
        <!-- Flotte und Kolonien (Kaiser II): nur wenn vorhanden, als schmale Zusatzzeile -->
        {#if (p.traders ?? 0) + (p.warships ?? 0) > 0}
          <p class="flotte" title="Flotte: {p.traders ?? 0} Handels- und {p.warships ?? 0} Kriegsschiffe · Kolonialland in Afrika, Amerika und Asien">
            ⚓ {p.traders ?? 0} Handels- · {p.warships ?? 0} Kriegsschiffe · {num((p.colonies ?? []).reduce((s, c) => s + c.area, 0))} km²</p>
        {/if}
      </section>
      <section class="card chronicle" aria-label="Reichschronik">
        <h2 class="display">Reichschronik</h2>
        <ul>
          {#each app.chronicle as c (c.id)}
            <li><span class="label">{c.year}</span><span>{c.text}</span></li>
          {/each}
        </ul>
      </section>
    </aside>
  </main>

  <footer>
    {#if app.over}
      <p><strong>Das Spiel ist beendet.</strong> {app.winner >= 0 ? `${ruler(app.players[app.winner])} wurde Kaiser!` : ''}</p>
      {#if app.winner >= 0}<button class="btn" type="button" onclick={() => (kroenungOpen = true)}>Krönungsurkunde</button>{/if}
      <button class="btn btn-primary" type="button" onclick={leaveGame}>Neues Spiel</button>
    {:else if self.phase === 'done'}
      <p>Warten auf {waitingFor.map((x) => x.name).join(', ') || 'das neue Jahr'} …</p>
    {:else}
      <!-- Fußleiste: Phase plus der wichtigste Rat des Kanzlers -->
      <div class="next"><strong class="label">Phase {PHASES[phaseIndex]?.label}</strong><Ratgeber compact /></div>
      <!-- Alle Ratschläge als Hinweis-Modal; die Aktionsspalte bleibt frei davon -->
      {#if app.settings?.mode === 'async' && fragen}
        <!-- Erlaubnis nur auf Klick erfragen (Browser-Vorgabe) -->
        <button class="btn" type="button" title="Browser-Benachrichtigung, wenn Sie wieder am Zug sind"
          onclick={async () => { await erlaubnisAnfragen(); fragen = kannFragen(); }}>Benachrichtigen</button>
      {/if}
      <button class="btn" type="button" onclick={() => (adviceOpen = true)}>Rat des Kanzlers</button>
    {/if}
  </footer>
</div>

<ProfileDialog bind:open={profileOpen} />
<BattleReport />
<Kroenung bind:open={kroenungOpen} />
<Ereignis bind:riskOpen />
<HelpDialog bind:open={helpOpen} />

<Modal bind:open={adviceOpen} title="Der Kanzler rät">
  <Ratgeber all />
  {#snippet actions()}
    <label class="rat-auto"><input type="checkbox" checked={ratAuto} onchange={(e) => setRatAuto(e.currentTarget.checked)} />
      Bei jedem Schritt automatisch zeigen</label>
    <button class="btn btn-primary" type="button" onclick={() => (adviceOpen = false)}>Verstanden</button>
  {/snippet}
</Modal>

<Modal bind:open={leaveOpen} title="Spiel verlassen?" variant="confirm">
  <p>Sie kehren zur Startseite zurück und können ein neues Spiel anlegen oder einem anderen beitreten.</p>
  {#if !app.over}
    <p class="hint">Ihr Platz in dieser Partie bleibt bestehen. Ohne Autopilot läuft Ihre Zugzeit ab und der Zug wird automatisch fortgesetzt; mit Autopilot spielt der Computer für Sie weiter.</p>
  {/if}
  {#snippet actions()}
    <button class="btn" type="button" onclick={() => (leaveOpen = false)}>Bleiben</button>
    {#if !app.over && !me?.bot}
      <button class="btn" type="button" onclick={() => { setAutopilot('kaufmann'); setTimeout(leaveGame, 300); }}>Autopilot an und verlassen</button>
    {/if}
    <button class="btn btn-primary" type="button" onclick={leaveGame}>Verlassen</button>
  {/snippet}
</Modal>

<style>
  /* Ganzer Hauptbildschirm passt in eine Fensterhöhe; nur die Chronik scrollt. */
  .screen { height: 100vh; height: 100dvh; display: flex; flex-direction: column; overflow: hidden; }
  .top { background: var(--ink); color: var(--parchment); border-bottom: 3px solid var(--gold); display: flex; flex-wrap: wrap; align-items: center; gap: 12px 28px; padding: 8px 12px; }
  .brand { display: flex; align-items: baseline; gap: 8px; }
  .logo { font-size: 34px; color: var(--gold-light); }
  .brand .label, .year .label { color: #cdbb94; }
  .year { display: flex; flex-direction: column; line-height: 1; font-family: var(--font-label); font-size: 24px; font-weight: 600; }
  nav { flex: 1; }
  ol { list-style: none; margin: 0; padding: 0; display: flex; flex-wrap: wrap; gap: 6px; font-family: var(--font-label); font-size: 13px; }
  ol li { padding: 8px 12px; border-radius: 4px; color: #a8987c; }
  ol li.done { color: #cdbb94; }
  ol li.done::before { content: '✓ '; }
  ol li.current { background: var(--gold); color: var(--ink); font-weight: 700; }
  .autopilot { display: flex; flex-direction: column; gap: 2px; }
  .autopilot .label { color: #cdbb94; font-size: 11px; }
  .autopilot { min-width: 210px; }
  /* Hilfe und Verlassen: runde Knöpfe in der Kopfleiste */
  .tools { display: flex; gap: 6px; }
  .tool { width: 36px; height: 36px; border-radius: 50%; border: 1px solid var(--gold); background: var(--ink); color: var(--gold-light); font-family: var(--font-label); font-size: 18px; font-weight: 700; cursor: pointer; display: flex; align-items: center; justify-content: center; }
  .tool:hover { background: var(--gold); color: var(--ink); }
  .tool:focus-visible { outline: 3px solid var(--gold); outline-offset: 2px; }
  .hint { color: var(--sepia); margin: 0; }
  /* Autopilot aktiv: Knopf golden hervorheben */
  .autopilot.on :global(button[role=combobox]) { background: var(--gold); color: var(--ink); }
  .clock { font-family: var(--font-label); font-size: 20px; color: var(--gold-light); }
  .clock.urgent { color: #ff9b8a; }
  .players { list-style: none; margin: 0; padding: 0; display: flex; gap: 6px; }
  .players li { width: 34px; height: 34px; border-radius: 50%; display: flex; align-items: center; justify-content: center; color: var(--ink); font-family: var(--font-label); font-weight: 700; box-shadow: 0 0 0 2px var(--ink); }
  .players li.ready { box-shadow: 0 0 0 2px #6f8f4e; }
  .players li.me { box-shadow: 0 0 0 2px var(--gold-light); }
  .players li.dead { opacity: .4; }
  /* Viele Herrscher: kleinere, überlappende Kreise, damit die Kopfleiste einzeilig bleibt */
  .players.viele { gap: 0; }
  .players.viele li { width: 28px; height: 28px; font-size: 12px; margin-left: -4px; }
  .players.viele li img { width: 28px; height: 28px; }
  /* Spielerkarte (SpielerKarte.svelte) bei Hover oder Tastaturfokus einblenden */
  .players li { position: relative; transition: transform .12s ease; }
  .players .chip { all: unset; width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; border-radius: 50%; cursor: help; }
  .players li:hover, .players li:focus-within { transform: scale(1.12); box-shadow: 0 0 0 2px var(--gold-light), 0 0 10px #e9c45a88; }
  .players li:hover :global(.tip), .players li:focus-within :global(.tip) { opacity: 1; transform: none; visibility: visible; transition-delay: 0s; }
  .top { position: relative; z-index: 20; }

  /* Drei Spalten: Karte | Aktionen der Phase | eigenes Reich und Chronik */
  main { flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(0, 1fr) clamp(330px, 26vw, 430px) clamp(300px, 24vw, 400px); gap: 2px; padding: 2px; }
  /* Mittelbreit: Karte über die volle Höhe links, rechts Aktionen und Reich nebeneinander schmaler */
  @media (max-width: 1250px) { main { grid-template-columns: minmax(0, 1fr) 320px 290px; gap: 2px; padding: 2px; } }
  .map { min-height: 0; min-width: 0; display: flex; flex-direction: column; gap: 8px; }
  .map h1 { font-size: clamp(22px, 2.2vw, 30px); }
  .map-box { flex: 1; min-height: 0; }
  aside { min-height: 0; display: flex; flex-direction: column; gap: 12px; }
  aside > section:not(.chronicle) { flex-shrink: 0; }
  /* Zierrahmen brauchen Platz: Seitenleiste kompakter halten */
  aside { gap: 2px; }
  aside :global(.card) { padding: 2px 6px; }
  .buildings { border-width: 14px; border-image-width: 22px; }
  .stats { gap: 2px; }
  .stats .label { font-size: 10px; letter-spacing: .06em; }
  .ruler .who { min-width: 0; }
  .ruler { display: flex; align-items: center; gap: 12px; padding: 12px 16px; }
  /* Profil-Link unter dem Namen: schlicht, spart Platz im Zierrahmen */
  .ruler .edit { align-self: flex-start; min-height: 32px; padding: 0; border: none; background: none; color: var(--blood); font-family: var(--font-label); font-size: 12px; font-weight: 600; letter-spacing: .04em; text-decoration: underline; text-underline-offset: 3px; cursor: pointer; }
  .ruler .edit:hover { color: var(--blood-dark); }
  .ruler .who { flex: 1; min-width: 140px; display: flex; flex-direction: column; gap: 2px; }
  .ruler .name { font-size: clamp(26px, 2.4vw, 36px); line-height: 1; }
  .ruler .motto { font-style: italic; color: var(--sepia); }
  .ruler .avatar { width: 64px; height: 64px; border-radius: 50%; border: 2px solid var(--gold); object-fit: cover; }
  .players li img { width: 34px; height: 34px; border-radius: 50%; object-fit: cover; }
  .stats { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 2px; }
  .stats .card { padding: 2px 4px; display: flex; flex-direction: column; }
  .v { font-size: 24px; font-weight: 600; }
  .v.neg { color: var(--blood); }
  .u { font-size: 13px; color: var(--sepia); }
  .buildings { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); text-align: center; gap: 6px; }
  .buildings div { display: flex; flex-direction: column; }
  .buildings .flotte { grid-column: 1 / -1; margin: 0; font-size: 13px; color: var(--sepia); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .chronicle { flex: 1; min-height: 120px; overflow-y: auto; overscroll-behavior: contain; }
  .chronicle h2 { font-size: 24px; margin-bottom: 6px; }
  .chronicle ul { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; font-size: 16px; line-height: 1.3; }
  .chronicle li { display: flex; gap: 10px; }
  .chronicle .label { padding-top: 3px; }
  footer { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; gap: 12px; padding: 8px 12px; border-top: 1px solid var(--line); background: var(--ground-dark); }
  footer p { margin: 0; }
  .next { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  @media (max-width: 640px) {
    .stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
    /* Handy: Kopfleiste kompakt – Phasen in einer Zeile über die volle Breite, Autopilot ohne Überschrift */
    .top { gap: 6px 12px; padding: 6px 8px; }
    nav { flex: 1 1 100%; order: 3; }
    ol { flex-wrap: nowrap; gap: 2px; font-size: 11px; }
    ol li { padding: 4px 6px; }
    .autopilot { min-width: 0; flex: 1 1 150px; }
    .autopilot .label { display: none; }
  }
  /* Schmale Fenster (Tablet hochkant, Handy): untereinander, Seite darf scrollen. */
  @media (max-width: 900px) {
    /* Schmal: Fenster bleibt fest, statt der Seite scrollt der Hauptbereich */
    main { grid-template-columns: 1fr; overflow-y: auto; overscroll-behavior: contain; }
    .map-box { height: 55vh; flex: none; }
    /* Untereinander in voller Inhaltshöhe; nur main scrollt */
    main { display: flex; flex-direction: column; }
    main > :global(*) { flex: none; }
    .chronicle { max-height: 50vh; }
  }
  /* Schalter im Rat-Fenster links, „Verstanden“ bleibt rechts */
  .rat-auto { margin-right: auto; display: flex; align-items: center; gap: 6px; font-size: 15px; }
</style>
