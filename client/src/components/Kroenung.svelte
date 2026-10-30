<script lang="ts">
  // Krönungsurkunde: prunkvolles Modal, sobald ein Herrscher Kaiser wird (Spielende).
  // Zeigt Reichskrone, Bildnis, Wappen, Wahlspruch, die Bilanz des Kaisers und die übrigen Fürsten.
  // Öffnet einmal je Partie von selbst (gemerkt im sessionStorage), danach über die Fußleiste.
  import { app } from '../lib/state.svelte';
  import { COUNTRIES, num, title } from '../lib/format';
  import { leaveGame } from '../lib/net';
  import { sfxFanfare } from '../lib/audio.svelte';
  import Modal from './Modal.svelte';
  import Wappen from './Wappen.svelte';
  import Fanfare from './Fanfare.svelte';

  let { open = $bindable(false) }: { open?: boolean } = $props();

  const kaiser = $derived(app.over && app.winner >= 0 ? app.players[app.winner] : null);
  const ich = $derived(app.winner === app.seat);
  // Übrige Fürsten nach Rang: Titel, dann Land
  const fuersten = $derived(app.players.filter((p) => p.seat !== app.winner)
    .sort((a, b) => b.title - a.title || b.land - a.land));

  // Einmal je Partie automatisch öffnen, mit Fanfare
  $effect(() => {
    if (!kaiser) return;
    const key = `kaiser.kroenung.${app.gameId}`;
    let seen = false;
    try { seen = sessionStorage.getItem(key) === '1'; sessionStorage.setItem(key, '1'); } catch { /* ohne Speicher: jedes Mal */ }
    if (!seen) {
      open = true;
      sfxFanfare();
    }
  });
</script>

{#if kaiser}
  <Modal bind:open title="Krönungsurkunde" wide>
    <div class="urkunde">
      <!-- Herolde: Fanfaren mit Bannern links und rechts, dazwischen Strahlenkranz und Reichskrone -->
      <div class="kopf">
      <Fanfare side="links" arms={kaiser.arms} country={kaiser.country} />
      <div class="krone-feld" aria-hidden="true">
        <div class="strahlen"></div>
        <svg class="krone" viewBox="0 0 220 170">
          <defs>
            <linearGradient id="kr-gold" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stop-color="#fbe7a1" /><stop offset=".45" stop-color="#e5b93f" /><stop offset="1" stop-color="#9c7425" />
            </linearGradient>
            <linearGradient id="kr-samt" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#b3262b" /><stop offset="1" stop-color="#5e1414" /></linearGradient>
            <radialGradient id="kr-stein" cx=".35" cy=".35" r=".7"><stop offset="0" stop-color="#fff" stop-opacity=".9" /><stop offset=".3" stop-color="currentColor" /><stop offset="1" stop-color="#000" stop-opacity=".5" /></radialGradient>
          </defs>
          <!-- Samthaube hinter den Platten -->
          <path d="M48 112Q50 52 110 46Q170 52 172 112Z" fill="url(#kr-samt)" stroke="#3a0a0a" stroke-width="1.5" />
          <!-- Acht Platten (sichtbar fünf), oben gerundet -->
          {#each [[34, 18], [70, 26], [112, 30], [154, 26], [190, 18]] as [cx, h], i}
            <path d="M{cx - 18} 150V{100 - h}Q{cx} {82 - h} {cx + 18} {100 - h}V150Z" fill="url(#kr-gold)" stroke="#6b5023" stroke-width="1.4" />
            <!-- Perlen auf der Plattenkante -->
            <path d="M{cx - 14} 148V{102 - h}Q{cx} {87 - h} {cx + 14} {102 - h}V148" fill="none" stroke="#fffaf0" stroke-width="2.4" stroke-dasharray="0.1 4.6" stroke-linecap="round" />
            {#if i % 2 === 0}
              <!-- Edelsteinplatte: großer Stein, Perlen im Kreis -->
              <circle cx={cx} cy={122 - h / 2} r="8" fill="url(#kr-stein)" color={i === 2 ? '#2a5ea8' : '#2f7a37'} stroke="#6b5023" stroke-width="1" />
              {#each [0, 60, 120, 180, 240, 300] as a}
                <circle cx={cx + 13 * Math.cos((a * Math.PI) / 180)} cy={122 - h / 2 + 13 * Math.sin((a * Math.PI) / 180)} r="2.2" fill="#fffaf0" stroke="#9c7425" stroke-width=".5" />
              {/each}
            {:else}
              <!-- Bildplatte: Rubin oben, Emailfeld -->
              <rect x={cx - 11} y={108 - h / 2} width="22" height="26" rx="2" fill="#1f3a5f" stroke="#e5b93f" stroke-width="1.6" />
              <circle cx={cx} cy={102 - h} r="4" fill="url(#kr-stein)" color="#b3262b" stroke="#6b5023" stroke-width=".8" />
            {/if}
          {/each}
          <!-- Bügel mit Perlenreihe, über den Platten -->
          <path d="M60 108Q62 40 110 34Q158 40 160 108" fill="none" stroke="url(#kr-gold)" stroke-width="9" stroke-linecap="round" />
          {#each Array(13) as _, i}
            {@const t = 0.12 + i * 0.063}
            {@const x = (1 - t) * (1 - t) * 60 + 2 * (1 - t) * t * 110 + t * t * 160}
            {@const y = (1 - t) * (1 - t) * 108 + 2 * (1 - t) * t * -28 + t * t * 108}
            <circle cx={x} cy={y} r="2.6" fill="#fffaf0" stroke="#9c7425" stroke-width=".6" />
          {/each}
          <!-- Reichsapfel-Kreuz auf dem Stirnbügel -->
          <circle cx="110" cy="30" r="9" fill="url(#kr-gold)" stroke="#6b5023" stroke-width="1.2" />
          <path d="M110 4V22M102 11H118" stroke="url(#kr-gold)" stroke-width="5" stroke-linecap="round" />
          <path d="M110 4V22M102 11H118" stroke="#6b5023" stroke-width="1" stroke-linecap="round" fill="none" />
          <!-- Stirnreif mit Steinen -->
          <rect x="14" y="146" width="192" height="14" rx="3" fill="url(#kr-gold)" stroke="#6b5023" stroke-width="1.4" />
          {#each [34, 70, 110, 150, 186] as cx, i}
            <circle cx={cx} cy="153" r="4" fill="url(#kr-stein)" color={['#b3262b', '#2f7a37', '#2a5ea8', '#2f7a37', '#b3262b'][i]} />
          {/each}
        </svg>
      </div>
      <Fanfare side="rechts" arms={kaiser.arms} country={kaiser.country} />
      </div>

      <p class="anno">Im Jahre des Herrn {app.year}</p>
      {#if ich}<p class="du">Ihnen wurde die Krone des Reiches aufgesetzt!</p>{/if}
      <div class="person">
        <!-- Bildnis im Goldoval, daneben das Wappen -->
        <div class="bildnis">
          {#if kaiser.avatar}<img src="/media/{kaiser.avatar}" alt="Bildnis von {kaiser.name}" />
          {:else}<span class="initial display" aria-hidden="true">{kaiser.name.slice(0, 1).toUpperCase()}</span>{/if}
        </div>
        <div class="titel">
          <span class="majestaet">{kaiser.female ? 'Ihre' : 'Seine'} Majestät</span>
          <span class="name display">{kaiser.name}</span>
          <span class="rang">{title(9, kaiser.female)} des Heiligen Römischen Reiches</span>
          <span class="haus">aus dem Hause {COUNTRIES[kaiser.country]}</span>
          {#if kaiser.motto}<span class="motto">„{kaiser.motto}“</span>{/if}
        </div>
        <Wappen arms={kaiser.arms} country={kaiser.country} size={96} label="Wappen von {kaiser.name}" />
      </div>

      <dl class="bilanz">
        <div><dt>Land</dt><dd class="num">{num(kaiser.land)} ha</dd></div>
        <div><dt>Staatskasse</dt><dd class="num">{num(kaiser.treasury)} Taler</dd></div>
        <div><dt>Untertanen</dt><dd class="num">{num(kaiser.population)}</dd></div>
        <div><dt>Heer</dt><dd class="num">{num(kaiser.soldiers)}</dd></div>
        <div><dt>Märkte · Mühlen</dt><dd class="num">{kaiser.markets} · {kaiser.mills}</dd></div>
        <div><dt>Palast · Kathedrale</dt><dd class="num">{kaiser.palace}/16 · {kaiser.cathedral}/14</dd></div>
      </dl>

      {#if fuersten.length}
        <h3 class="label">Es huldigen die Fürsten des Reiches</h3>
        <ol class="fuersten">
          {#each fuersten as f (f.seat)}
            <li class:tot={!f.alive}>
              <Wappen arms={f.arms} country={f.country} size={26} label="Wappen von {f.name}" />
              <span>{title(f.title, f.female)} {f.name} von {COUNTRIES[f.country]}{f.alive ? '' : ' †'}{f.seat === app.seat ? ' (Sie)' : ''}</span>
              <span class="num land">{num(f.land)} ha</span>
            </li>
          {/each}
        </ol>
      {/if}
    </div>
    {#snippet actions()}
      <button class="btn" type="button" onclick={() => (open = false)}>Zur Karte</button>
      <button class="btn btn-primary" type="button" onclick={leaveGame}>Neues Spiel</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .urkunde { display: flex; flex-direction: column; align-items: center; gap: 10px; text-align: center; }
  /* Strahlenkranz: langsam drehende Goldstrahlen hinter der Krone */
  .kopf { display: flex; align-items: center; justify-content: center; gap: 8px; }
  /* Schmal: Fanfaren ausblenden, Krone bleibt */
  @media (max-width: 700px) { .kopf :global(.fanfare) { display: none; } }
  .krone-feld { position: relative; width: 240px; height: 180px; display: flex; align-items: center; justify-content: center; }
  .strahlen {
    position: absolute; inset: -40px; border-radius: 50%;
    background: repeating-conic-gradient(from 0deg, #e5b93f55 0deg 6deg, transparent 6deg 15deg);
    mask-image: radial-gradient(circle, #000 25%, transparent 70%);
    animation: strahlen 40s linear infinite;
  }
  @keyframes strahlen { to { transform: rotate(360deg); } }
  .krone { position: relative; width: 220px; height: 170px; filter: drop-shadow(0 6px 10px #2b211866); animation: krone 1.2s ease-out; }
  @keyframes krone { from { transform: translateY(-40px) scale(.7); opacity: 0; } to { transform: none; opacity: 1; } }
  .anno { margin: 0; font-family: var(--font-label); letter-spacing: .2em; text-transform: uppercase; font-size: 13px; color: var(--sepia); }
  .du { margin: 0; font-family: var(--font-display); font-size: 26px; color: var(--blood); }
  .person { display: flex; align-items: center; gap: 22px; flex-wrap: wrap; justify-content: center; }
  .bildnis {
    width: 120px; height: 150px; border-radius: 50%; overflow: hidden; flex-shrink: 0;
    border: 5px solid #e5b93f; box-shadow: 0 0 0 2px #6b5023, 0 0 0 7px #fffaf0, 0 0 0 9px #b08a3e, 0 8px 18px #2b211866;
    background: radial-gradient(circle at 40% 35%, #8a5a2b, #3a2614); display: flex; align-items: center; justify-content: center;
  }
  .bildnis img { width: 100%; height: 100%; object-fit: cover; }
  .initial { font-size: 72px; color: #e5b93f; }
  .titel { display: flex; flex-direction: column; align-items: center; gap: 2px; max-width: 420px; }
  .majestaet { font-family: var(--font-label); letter-spacing: .18em; text-transform: uppercase; font-size: 14px; color: var(--sepia); }
  .name { font-size: clamp(44px, 6vw, 64px); line-height: 1; color: var(--ink); text-shadow: 0 2px 0 #e5b93f66; }
  .rang { font-family: var(--font-label); font-weight: 700; font-size: 15px; letter-spacing: .06em; color: var(--blood); }
  .haus { font-style: italic; }
  .motto { font-style: italic; font-size: 18px; color: var(--sepia); }
  .bilanz { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 6px 18px; margin: 4px 0 0; padding: 10px 0; border-top: 1px solid #c9a95c; border-bottom: 1px solid #c9a95c; width: min(620px, 100%); }
  .bilanz dt { font-family: var(--font-label); font-size: 11px; letter-spacing: .1em; text-transform: uppercase; color: var(--sepia); }
  .bilanz dd { margin: 0; font-size: 18px; font-weight: 600; }
  h3 { margin: 4px 0 0; }
  .fuersten { list-style: none; margin: 0; padding: 0; width: min(520px, 100%); display: flex; flex-direction: column; gap: 4px; text-align: left; }
  .fuersten li { display: flex; align-items: center; gap: 10px; }
  .fuersten .land { margin-left: auto; color: var(--sepia); }
  .fuersten li.tot { opacity: .6; }
  @media (prefers-reduced-motion: reduce) { .strahlen, .krone { animation: none; } }
</style>
