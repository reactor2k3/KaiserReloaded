<script lang="ts">
  // Lobby: Spiel anlegen oder per Code beitreten, danach Warteraum bis zum Start.
  import { app } from '../lib/state.svelte';
  import { addBot, cancelJoin, createGame, kickPlayer, joinGame, peekGame, startGame, leaveGame } from '../lib/net';
  import Auswahl from './Auswahl.svelte';
  import Ruhmeshalle from './Ruhmeshalle.svelte';
  import MeineSpiele from './MeineSpiele.svelte';
  import Spieleliste from './Spieleliste.svelte';
  import { setSkin, skin } from '../lib/skin.svelte';
  import { ASYNC_SECONDS, COUNTRIES, DIFFICULTY, LIVE_SECONDS, MORTALITY, playerColor, zugzeit } from '../lib/format';
  import type { Settings } from '../lib/types';

  const inviteCode = new URLSearchParams(location.search).get('spiel') ?? '';
  let mode = $state<'create' | 'join'>(inviteCode ? 'join' : 'create');
  let name = $state('');
  let female = $state(false);
  let country = $state(5);
  let code = $state(inviteCode);
  let botStrategy = $state('feldherr');
  let settings = $state<Settings>({ phaseSeconds: 90, timePenalty: true, battleMode: 'tactical', classic: false, mortality: 'normal', difficulty: 'mittel', mode: 'live' });

  const joined = $derived(app.seat >= 0 && app.gameId !== '');
  // Vor dem Beitritt kommen die vergebenen Länder aus der Vorschau (peek), danach aus der Lobby.
  const preview = $derived(mode === 'join' && app.preview?.gameId === code.trim().toUpperCase() ? app.preview : null);
  // Belegung je Land: im Remake teilen sich bis zu drei Herrscher ein Land, im Original keiner.
  const perCountry = $derived((mode === 'create' ? settings.classic : preview?.settings.classic) ? 1 : 3);
  const counts = $derived((joined ? app.lobby : preview?.players ?? []).reduce((m, p) => m.set(p.country, (m.get(p.country) ?? 0) + 1), new Map<number, number>()));
  const taken = $derived(new Set([...counts].filter(([, n]) => n >= perCountry).map(([c]) => c)));

  $effect(() => {
    if (mode === 'join' && code.trim().length === 6 && app.connection === 'open') peekGame(code);
  });
  // Erstes freies Land vorwählen, falls das gewählte vergeben ist.
  $effect(() => {
    if (taken.has(country)) {
      const free = [1, 2, 3, 4, 5, 6, 7, 8, 9].find((c) => !taken.has(c));
      if (free) country = free;
    }
  });
  const inviteLink = $derived(`${location.origin}${location.pathname}?spiel=${app.gameId}`);
  const valid = $derived(name.trim().length > 0 && name.trim().length <= 20 && !taken.has(country) &&
    (mode === 'create' || (code.trim().length === 6 && !!preview?.joinable)));

  function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!valid) return;
    if (mode === 'create') createGame(name, country, female, settings);
    else joinGame(code, name, country, female);
  }

  // Auslastung des Servers (GET /api/status): laufende Spiele und Grenzen. Wird alle 30 s
  // aufgefrischt, damit „Spiel anlegen“ wieder frei wird, sobald ein Platz entsteht.
  let status = $state({ games: 0, maxGames: 10, maxPlayers: 6 });
  const full = $derived(status.games >= status.maxGames);
  async function loadStatus() {
    try {
      const res = await fetch('/api/status');
      if (res.ok) status = await res.json();
    } catch {
      /* Server nicht erreichbar: die Verbindungsanzeige meldet das bereits */
    }
  }
  $effect(() => {
    loadStatus();
    const t = setInterval(loadStatus, 30_000);
    return () => clearInterval(t);
  });

  // Störer oder überzählige Computer-Gegner vor dem Start entfernen (Menschen werden für
  // dieses Spiel gesperrt, daher nachfragen).
  function kick(seat: number, who: string) {
    if (!app.lobby[seat]?.bot && !confirm(`${who} aus der Lobby entfernen? Der Spieler kann diesem Spiel dann nicht mehr beitreten.`)) return;
    kickPlayer(seat);
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(inviteLink);
    } catch {
      /* Zwischenablage gesperrt: Link bleibt sichtbar zum Abschreiben */
    }
  }
</script>

<main class="lobby">
  <header>
    <span class="logo display">Kaiser</span>
    <span class="label">Reloaded</span>
  </header>

  <!-- Inhalte nebeneinander; die Seite selbst scrollt nie, nur die Karten -->
  <div class="columns">
  {#if !joined && app.pendingJoin}
    <!-- Eigene Anfrage wartet: kein Formular, nur Status und Zurückziehen -->
    <section class="card warten" aria-live="polite">
      <h2 class="display">Anfrage gesendet</h2>
      <p>Spiel <strong>{app.pendingJoin.gameId}</strong>: {app.pendingJoin.host} entscheidet über Ihren Beitritt.
        Sie können die Seite offen lassen oder später über „Meine Spiele“ zurückkommen.</p>
      <button class="btn" type="button" onclick={cancelJoin}>Anfrage zurückziehen</button>
    </section>
  {:else if !joined}
    <form class="card" onsubmit={submit}>
      <div class="tabs" role="tablist" aria-label="Spiel wählen">
        <button type="button" role="tab" aria-selected={mode === 'create'} class:active={mode === 'create'} onclick={() => (mode = 'create')}>Neues Spiel</button>
        <button type="button" role="tab" aria-selected={mode === 'join'} class:active={mode === 'join'} onclick={() => (mode = 'join')}>Beitreten</button>
      </div>

      {#if mode === 'join'}
        <label class="feld">
          <input bind:value={code} maxlength="6" autocomplete="off" placeholder="z. B. K7QH2M" class="code-input" />
          <span class="feld-label">Spielcode</span>
        </label>
        {#if preview}
          <p class="hint">
            {#if !preview.joinable}In dieses Spiel kann man nicht mehr einsteigen (voll oder schon jemand befördert).
            {:else if preview.started}Das Spiel läuft bereits. Nach der Zulassung steigen Sie zum nächsten Jahr ein.
              Am Tisch: {preview.players.map((p) => `${p.name} (${COUNTRIES[p.country]})`).join(', ')}.
            {:else}Am Tisch: {preview.players.map((p) => `${p.name} (${COUNTRIES[p.country]})`).join(', ') || 'noch niemand'}.{/if}
          </p>
        {/if}
      {/if}

      <label class="feld">
        <input bind:value={name} maxlength="20" autocomplete="nickname" required placeholder="z. B. Friedrich" />
        <span class="feld-label">Ihr Name</span>
      </label>

      <fieldset>
        <legend class="label">Anrede</legend>
        <label><input type="radio" bind:group={female} value={false} /> Herr</label>
        <label><input type="radio" bind:group={female} value={true} /> Frau</label>
      </fieldset>

      <Auswahl label="Land" float="Land" bind:value={country}
        options={COUNTRIES.slice(1).map((c, i) => ({ value: i + 1, disabled: taken.has(i + 1),
          label: c + (taken.has(i + 1) ? ' (vergeben)' : counts.get(i + 1) ? ` (${counts.get(i + 1)}/${perCountry} Herrscher)` : '') }))} />

      {#if mode === 'create'}
        <fieldset class="settings">
          <legend class="label">Spieleinstellungen</legend>
          <!-- Ein Raster mit zwei Spalten, damit „Spiel anlegen“ ohne Scrollen sichtbar bleibt -->
          <div class="zwei">
            <!-- Spielmodus: live (alle gleichzeitig) oder asynchron (Stunden bis Tage je Phase) -->
            <Auswahl label="Spielmodus" float="Spielmodus" bind:value={settings.mode}
              onchange={(m) => { settings.phaseSeconds = m === 'async' ? 86400 : 90; settings.timePenalty = m !== 'async'; }}
              options={[{ value: 'live', label: 'Live (alle gleichzeitig)' }, { value: 'async', label: 'Asynchron (Stunden/Tage)' }]} />
            <Auswahl label="Zugzeit je Phase" float="Zugzeit je Phase" bind:value={settings.phaseSeconds}
              options={(settings.mode === 'async' ? ASYNC_SECONDS : LIVE_SECONDS).map((s) => ({ value: s, label: zugzeit(s) }))} />
            <Auswahl label="Schlachten" float="Schlachten" bind:value={settings.battleMode}
              options={[{ value: 'tactical', label: 'Taktisch (eigene Aufstellung)' }, { value: 'quick', label: 'Schnell (automatisch)' }]} />
            <Auswahl label="Computer-Gegner" float="Computer-Gegner" bind:value={settings.difficulty}
              options={DIFFICULTY.map((d) => ({ value: d.value, label: d.label }))} />
            <!-- Sterblichkeit (natürlicher Tod) über beide Spalten, die Beschreibung ist länger -->
            <div class="breit"><Auswahl label="Sterblichkeit" float="Sterblichkeit" bind:value={settings.mortality}
              options={MORTALITY.map((m) => ({ value: m.value, label: `${m.label} · ${m.info}` }))} /></div>
          </div>
          <div class="zwei">
            <label><input type="checkbox" bind:checked={settings.timePenalty} disabled={settings.phaseSeconds === 0 || settings.mode === 'async'} /> Überziehen kostet Lebenszeit</label>
            <label><input type="checkbox" bind:checked={settings.classic} /> Original-Regeln des C64</label>
          </div>
        </fieldset>
      {/if}

      {#if mode === 'create' && full}
        <p class="voll" role="status">Im Moment sind keine weiteren Spiele möglich, weil gerade {status.maxGames} Spiele laufen.
          Bitte später noch einmal versuchen oder einem laufenden Spiel beitreten.</p>
      {/if}
      <button class="btn btn-primary" type="submit" disabled={!valid || app.connection !== 'open' || (mode === 'create' && full)}>
        {mode === 'create' ? 'Spiel anlegen' : 'Beitritt anfragen'}
      </button>
    </form>
  {:else}
    <section class="card room" aria-labelledby="room-title">
      <h1 id="room-title" class="display">Der Reichstag versammelt sich</h1>
      <p>Spielcode <strong class="code">{app.gameId}</strong></p>
      <div class="invite">
        <label class="feld grow"><input readonly value={inviteLink} /><span class="feld-label">Einladungslink</span></label>
        <button class="btn" type="button" onclick={copyLink}>Link kopieren</button>
      </div>
      <ul class="players">
        {#each app.lobby as p, i (i)}
          <li>
            <span class="badge" style:background={playerColor(app.lobby.map((x, k) => ({ seat: k, country: x.country })), i)}>{p.name.slice(0, 1).toUpperCase()}</span>
            <span>{p.female ? 'Frau' : 'Herr'} {p.name} von {COUNTRIES[p.country]}{i === 0 ? ' · Gastgeber' : ''}</span>
            {#if p.bot}
              <span class="online on">Computer · {p.bot === 'feldherr' ? 'Feldherr' : 'Kaufmann'}</span>
            {:else}
              <span class="online" class:on={p.online}>{p.online ? 'da' : 'fehlt'}</span>
            {/if}
            {#if app.host && i > 0}
              <button class="btn sm kick" type="button" onclick={() => kick(i, p.name)}
                aria-label="{p.name} aus der Lobby entfernen" title="Aus der Lobby entfernen">✕</button>
            {/if}
          </li>
        {/each}
      </ul>
      {#if app.settings}
        <p class="hint">{app.settings.mode === 'async' ? 'Asynchron' : 'Live'} · Zugzeit {app.settings.phaseSeconds ? `${zugzeit(app.settings.phaseSeconds)} je Phase` : 'unbegrenzt'} ·
          {app.settings.battleMode === 'quick' ? 'Schnelle Schlachten' : 'Taktische Schlachten'} ·
          Sterblichkeit {MORTALITY.find((m) => m.value === (app.settings?.mortality ?? 'normal'))?.label} ·
          Computer {DIFFICULTY.find((d) => d.value === (app.settings?.difficulty || 'schwer'))?.label} ·
          {app.settings.classic ? 'Original-Regeln' : 'Remake-Regeln'}</p>
      {/if}
      {#if app.host && app.lobby.length < status.maxPlayers}
        <div class="invite">
          <div class="grow"><Auswahl label="Spielweise des Computer-Herrschers" float="Computer-Herrscher" bind:value={botStrategy}
            options={[{ value: 'kaufmann', label: 'Kaufmann – baut Wirtschaft und Prachtbauten' }, { value: 'feldherr', label: 'Feldherr – rüstet auf und führt Krieg' }]} /></div>
          <button class="btn" type="button" onclick={() => addBot(botStrategy)}>Computer-Herrscher hinzufügen</button>
        </div>
      {/if}
      <div class="actions">
        <button class="btn" type="button" onclick={leaveGame}>Verlassen</button>
        {#if app.host}
          <button class="btn btn-primary" type="button" onclick={startGame}>Spiel starten</button>
        {:else}
          <span class="hint">Warten auf den Gastgeber …</span>
        {/if}
      </div>
    </section>
  {/if}
  {#if !joined}
    <!-- Rechte Spalte: eigene laufende Spiele über der Ruhmeshalle, beide scrollen einzeln -->
    <div class="side"><Spieleliste onJoin={(id) => { mode = 'join'; code = id; }} /><MeineSpiele /><Ruhmeshalle /></div>
  {/if}
  </div>
  <p class="skin-switch">
    Darstellung:
    <button type="button" class:on={skin.value === 'barock'} aria-pressed={skin.value === 'barock'} onclick={() => setSkin('barock')}>Barock</button>
    <button type="button" class:on={skin.value === 'c64'} aria-pressed={skin.value === 'c64'} onclick={() => setSkin('c64')}>C64</button>
  </p>
</main>

<style>
  /* Feste Fensterhöhe: Kopf, Spalten, Darstellungsschalter. Gescrollt wird nur in den Karten. */
  .lobby { height: 100vh; height: 100dvh; overflow: hidden; display: grid; grid-template-rows: auto minmax(0, 1fr) auto;
    justify-items: center; padding: 2px 2px 6px; gap: 2px; }
  header { display: flex; align-items: baseline; gap: 12px; }
  .logo { font-size: clamp(40px, 8vh, 64px); line-height: 1; color: var(--ink); }
  .columns { width: 100%; min-height: 0; display: flex; justify-content: center; align-items: flex-start; gap: 2px; }
  .columns > :global(.card), .side > :global(.card) { max-height: 100%; overflow-y: auto; overscroll-behavior: contain; }
  /* Kinder nicht stauchen, sonst quetscht Flex den Inhalt statt zu scrollen */
  .columns > :global(.card > *), .side > :global(.card > *) { flex-shrink: 0; }
  .side { width: min(520px, 100%); max-height: 100%; min-height: 0; display: flex; flex-direction: column; gap: 2px; }
  .side > :global(.card) { width: 100%; flex: 0 1 auto; min-height: 0; }
  form, .room { width: min(520px, 100%); display: flex; flex-direction: column; gap: 16px; }
  /* Schmal: Karten untereinander, beide teilen sich die Höhe und scrollen einzeln */
  @media (max-width: 1100px) {
    .columns { flex-direction: column; align-items: center; gap: 12px; }
    .columns > :global(.card), .side { flex: 0 1 auto; min-height: 0; }
  }
  .tabs { display: flex; border: 1px solid var(--line-strong); border-radius: 4px; overflow: hidden; align-self: flex-start; }
  .tabs button { min-height: 44px; padding: 0 18px; border: none; background: transparent; font-family: var(--font-label); font-weight: 600; cursor: pointer; }
  .tabs button.active { background: var(--ink); color: var(--parchment); }
  fieldset { border: none; padding: 0; margin: 0; display: flex; gap: 18px; flex-wrap: wrap; align-items: center; }
  /* Abstand oben für die Kartuschen der Auswahllisten */
  .breit { grid-column: 1 / -1; }
  .zwei { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px 8px; align-items: center; }
  @media (max-width: 640px) { .zwei { grid-template-columns: 1fr; } }
  fieldset.settings { flex-direction: column; align-items: stretch; gap: 12px; border-top: 1px solid var(--line); padding-top: 12px; }
  legend { margin-bottom: 6px; }
  .room h1 { font-size: 36px; }
  .code { font-family: var(--font-label); font-size: 24px; letter-spacing: .2em; }
  .invite { display: flex; gap: 8px; }
  .invite .grow { flex: 1; min-width: 0; }
  .skin-switch { display: flex; align-items: center; gap: 8px; color: var(--sepia); margin: 0; }
  .skin-switch button { min-height: 36px; padding: 0 12px; border: 1px solid var(--line-strong); border-radius: 4px; background: transparent; font-family: var(--font-label); font-size: 12px; cursor: pointer; }
  .skin-switch button.on { background: var(--ink); color: var(--parchment); }
  .invite { align-items: center; }
  .code-input { text-transform: uppercase; letter-spacing: .15em; }
  .players { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
  .players li { display: flex; align-items: center; gap: 12px; }
  .badge { width: 36px; height: 36px; border-radius: 50%; display: flex; align-items: center; justify-content: center; color: var(--ink); font-family: var(--font-label); font-weight: 700; border: 2px solid var(--ink); }
  .online { margin-left: auto; font-size: 14px; color: var(--sepia); }
  .online.on { color: var(--good); font-weight: 600; }
  /* Entfernen-Knopf des Gastgebers: klein, nicht gestreckt wie andere .btn.sm */
  .players .kick { flex: none; padding: 2px 8px; min-width: 0; }
  .hint { color: var(--sepia); font-size: 15px; margin: 0; }
  /* Hinweis „Server voll“: deutlich, aber kein Fehlerrot – Beitreten bleibt ja möglich */
  .voll { margin: 0; padding: 8px 10px; border: 1px solid var(--sepia); border-radius: 6px; font-size: 15px; }
  .actions { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
</style>
