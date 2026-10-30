<script lang="ts">
  // Reichskarte aus Natural-Earth-Daten (scripts/map). Die Pfade werden einmal gezeichnet;
  // bei Änderungen ändern sich nur Füllfarben, Beschriftungen und Animationen.
  // Animationen: fließende Flüsse, pulsierende Hauptstädte der Herrscher am Zug,
  // Heerzug-Pfeil mit gekreuzten Schwertern bei Krieg. „Bewegung reduzieren“ schaltet sie ab.
  // Besitz nach Hektar: Jedes Land hat app.countryArea Hektar. Jeder Herrscher (bis zu drei je Land)
  // bekommt einen Streifen nach seinem Anteil (lib/teilung), der Rest bleibt als freie Fläche
  // schraffiert. Die Streifen wandern weich bei Landkauf, Verkauf oder Eroberung.
  import map from '../assets/map.json';
  import { app } from '../lib/state.svelte';
  import { COUNTRY_COLOR, playerColor, title } from '../lib/format';
  import { band, curve, line, sampleLand, slices, spots } from '../lib/teilung';
  import type { PublicPlayer } from '../lib/types';

  // Herrscher je Land, nach Sitz sortiert (Reihenfolge der Streifen und Farbabstufung)
  const byLand = $derived(app.players.reduce((m, p) => m.set(p.country, [...(m.get(p.country) ?? []), p]), new Map<number, PublicPlayer[]>()));
  const owners = $derived(new Map([...byLand].map(([c, ps]) => [c, ps[0]])));
  const mine = $derived(app.self?.player.country ?? 0);

  // Streifen je besetztem Land; Tote zählen nicht mehr. Hat jemand durch Eroberung mehr Land
  // als die Landesfläche, gilt die Summe als Fläche (dann ohne freien Rest).
  const parts = $derived(new Map(map.lands.filter((l) => byLand.has(l.id)).map((l) => {
    const ps = byLand.get(l.id)!;
    const s = sampleLand(l.d);
    const held = ps.map((p) => (p.alive ? Math.max(1, p.land) : 0));
    const rest = Math.max(0, app.countryArea - held.reduce((a, b) => a + b, 0));
    // Besitz beginnt auf der Seite der Hauptstadt (Preußen: im Osten um Berlin, nicht in der Rheinprovinz)
    const k = s.horizontal ? 1 : 0;
    const fromEnd = Number(l.capital[k]) > (s.horizontal ? s.box.y + s.box.h / 2 : s.box.x + s.box.w / 2);
    const order = fromEnd ? [rest, ...[...held].reverse()] : [...held, rest];
    const sl = slices(s, order);
    // Geschwungene Grenzen: Kurve j trennt Streifen j-1 und j (Außenkanten gerade). Die Wellenhöhe
    // schrumpft bei schmalen Nachbarstreifen, damit sich Grenzen nicht kreuzen.
    const width = sl.map((x) => x.to - x.from);
    const curves = [sl[0].from, ...sl.map((x) => x.to)].map((pos, j) => {
      const inner = j > 0 && j < sl.length;
      const amp = inner ? Math.min(7, Math.min(width[j - 1], width[j]) / 3) : 0;
      return curve(s, pos, l.id * 7 + j, amp);
    });
    const withD = sl.map((x, i) => ({ ...x, d: band(curves[i], curves[i + 1]) }));
    const own = fromEnd ? withD.slice(1).reverse() : withD.slice(0, -1);
    const freeSl = fromEnd ? withD[0] : withD[withD.length - 1];
    return [l.id, { s, list: ps.map((p, i) => ({ p, ...own[i] })), free: rest > 0 ? freeSl : null, freeHa: rest,
      // innere Grenzlinien (leere Streifen erzeugen keine Doppellinie)
      cuts: curves.slice(1, -1).filter((_, j) => width[j] > 0.5 && width[j + 1] > 0.5).map(line) }];
  })));
  // Bauten als Symbole im Gebiet: Palast und Kathedrale (je eins, Größe nach Baufortschritt),
  // dazu jeder Markt und jede Mühle; Plätze stabil je Land (lib/teilung spots). Reicht der Platz
  // nicht, werden die Symbole schrittweise kleiner, bis alle hineinpassen.
  const NAMES: Record<string, string> = { palast: 'Palast', kathedrale: 'Kathedrale', muehle: 'Mühle', markt: 'Markt' };
  const bauten = $derived([...parts].flatMap(([id, part]) => {
    const land = map.lands.find((l) => l.id === id)!;
    const avoid: [number, number][] = part.list.map((x) => [x.cx, x.cy + 8] as [number, number]);
    avoid.push([Number(land.label[0]), part.s.box.y + 14]);
    return part.list.filter((x) => x.p.alive).flatMap((x) => {
      const want: string[] = [
        ...(x.p.palace > 0 ? ['palast'] : []), ...(x.p.cathedral > 0 ? ['kathedrale'] : []),
        // Mühlen und Märkte abwechselnd, damit bei Platznot beide Arten sichtbar bleiben
        ...Array.from({ length: Math.max(x.p.mills, x.p.markets) }, (_, i) => [
          ...(i < x.p.mills ? ['muehle'] : []), ...(i < x.p.markets ? ['markt'] : [])]).flat(),
      ];
      // Größe passend zum Gebiet: kleine Streifen bekommen kleinere Symbole (Fläche ~ Punktzahl n)
      let fit = Math.min(1, Math.max(0.6, Math.sqrt(x.n) / 22));
      // Kartenpunkte (Karte 900 breit): Markt < Mühle < Palast < Kathedrale
      const size = (k: string) => fit * (k === 'markt' ? 21 : k === 'muehle' ? 24
        : k === 'palast' ? 26 + 12 * (x.p.palace / 16) : 28 + 14 * (x.p.cathedral / 14));
      let at = spots(part.s, Math.min(x.from, x.to), Math.max(x.from, x.to), want.length, avoid, id * 13 + x.p.seat, 20 * fit);
      for (let tries = 0; at.length < want.length && tries < 4; tries++) {
        fit *= 0.8;
        at = spots(part.s, Math.min(x.from, x.to), Math.max(x.from, x.to), want.length, avoid, id * 13 + x.p.seat, 20 * fit);
      }
      return at.map(([px, py], i) => ({
        key: `${x.p.seat}-${want[i]}-${i}`, kind: want[i], x: px, y: py, size: size(want[i]),
        tip: `${NAMES[want[i]]} von ${x.p.name}` + (want[i] === 'palast' ? ` (${x.p.palace}/16)` : want[i] === 'kathedrale' ? ` (${x.p.cathedral}/14)`
          : want[i] === 'markt' ? ` – ${x.p.markets} Märkte` : ` – ${x.p.mills} Mühlen`),
      }));
    });
  }));
  // Zoom und Verschieben über die viewBox: Mausrad zoomt zum Zeiger, Ziehen verschiebt,
  // Doppelklick zoomt hinein; Knöpfe für Maus-, Touch- und Tastaturbedienung. 1- bis 6-fach.
  const MAX_ZOOM = 6;
  let svgEl: SVGSVGElement;
  let view = $state({ x: 0, y: 0, w: map.width, h: map.height });
  const zoom = $derived(map.width / view.w);
  let drag = $state<{ id: number; x: number; y: number; vx: number; vy: number; moved: boolean } | null>(null);

  /** Ansicht setzen und im Kartenrand halten. */
  function setView(x: number, y: number, w: number) {
    w = Math.min(map.width, Math.max(map.width / MAX_ZOOM, w));
    const h = (w * map.height) / map.width;
    view = { x: Math.min(map.width - w, Math.max(0, x)), y: Math.min(map.height - h, Math.max(0, y)), w, h };
  }

  /** Bildschirmpunkt in Kartenkoordinaten (berücksichtigt Seitenverhältnis/Ränder des SVG). */
  function toMap(cx: number, cy: number): [number, number] {
    const m = svgEl.getScreenCTM();
    if (!m) return [view.x + view.w / 2, view.y + view.h / 2];
    const p = new DOMPoint(cx, cy).matrixTransform(m.inverse());
    return [p.x, p.y];
  }

  /** Um Faktor f zoomen, Punkt (mx, my) bleibt unter dem Zeiger stehen. */
  function zoomAt(f: number, mx = view.x + view.w / 2, my = view.y + view.h / 2) {
    const w = Math.min(map.width, Math.max(map.width / MAX_ZOOM, view.w / f));
    const k = w / view.w;
    setView(mx - (mx - view.x) * k, my - (my - view.y) * k, w);
  }

  function onWheel(e: WheelEvent) {
    e.preventDefault();
    const [mx, my] = toMap(e.clientX, e.clientY);
    zoomAt(e.deltaY < 0 ? 1.2 : 1 / 1.2, mx, my);
  }
  function onDown(e: PointerEvent) {
    if (e.button !== 0 || zoom <= 1) return;
    drag = { id: e.pointerId, x: e.clientX, y: e.clientY, vx: view.x, vy: view.y, moved: false };
  }
  function onMove(e: PointerEvent) {
    if (!drag || drag.id !== e.pointerId) return;
    const scale = view.w / svgEl.getBoundingClientRect().width;
    const dx = (e.clientX - drag.x) * scale, dy = (e.clientY - drag.y) * scale;
    if (!drag.moved && Math.hypot(dx, dy) > 2) { drag.moved = true; svgEl.setPointerCapture(e.pointerId); }
    if (drag.moved) setView(drag.vx - dx, drag.vy - dy, view.w);
  }
  function onUp(e: PointerEvent) {
    if (drag?.id === e.pointerId) drag = null;
  }
  function onDbl(e: MouseEvent) {
    const [mx, my] = toMap(e.clientX, e.clientY);
    zoomAt(2, mx, my);
  }
  /** Auf das eigene Gebiet springen (2,5-fach). */
  function zoomMine() {
    const [x, y] = myPart ? [myPart.cx, myPart.cy] : capital(mine);
    const w = map.width / 2.5;
    setView(x - w / 2, y - (w * map.height) / map.width / 2, w);
  }

  // Mausrad braucht passive: false, damit die Seite nicht mitscrollt
  $effect(() => {
    svgEl.addEventListener('wheel', onWheel, { passive: false });
    return () => svgEl.removeEventListener('wheel', onWheel);
  });

  const myPart = $derived(parts.get(mine)?.list.find((x) => x.p.seat === app.seat) ?? null);

  /** Ankerpunkt eines Herrschers: Hauptstadt, in geteilten Ländern Mitte seines Streifens. */
  function anchor(seat: number): [number, number] {
    const p = app.players[seat];
    const part = p && parts.get(p.country)?.list.find((x) => x.p.seat === seat);
    return part ? [part.cx, part.cy] : capital(p?.country ?? 0);
  }
  // Hauptstadt-Koordinaten (capital = [x, y, Name])
  function capital(country: number): [number, number] {
    const c = map.lands.find((l) => l.id === country)?.capital;
    return c ? [Number(c[0]), Number(c[1])] : [0, 0];
  }

  // Aktueller Krieg: während Diplomatie/Aufstellung aus der eigenen Sicht, sonst die letzte Schlacht (bis ins Folgejahr).
  const war = $derived.by(() => {
    const w = app.self?.war;
    const pair = w ? { a: w.attacker, d: w.defender, fought: false }
      : app.lastBattle && app.lastBattle.year >= app.year - 1 ? { a: app.lastBattle.attacker, d: app.lastBattle.defender, fought: true } : null;
    if (!pair) return null;
    const pa = app.players[pair.a], pd = app.players[pair.d];
    if (!pa || !pd) return null;
    const [x1, y1] = anchor(pair.a), [x2, y2] = anchor(pair.d);
    // Bogen: Kontrollpunkt senkrecht zur Verbindungslinie versetzt
    const mx = (x1 + x2) / 2, my = (y1 + y2) / 2, dx = x2 - x1, dy = y2 - y1;
    const len = Math.hypot(dx, dy) || 1;
    // Kurze Wege (Nachbarn) bekommen einen deutlicheren Bogen, damit der Heerzug sichtbar bleibt
    const bow = Math.max(len * 0.3, 45);
    const cx = mx - (dy / len) * bow, cy = my + (dx / len) * bow;
    // Mittelpunkt der Kurve (t = 0,5) für die Schwerter
    const sx = 0.25 * x1 + 0.5 * cx + 0.25 * x2, sy = 0.25 * y1 + 0.5 * cy + 0.25 * y2;
    return { d: `M${x1} ${y1}Q${cx} ${cy} ${x2 - (dx / len) * 10} ${y2 - (dy / len) * 10}`, sx, sy, defender: pd.country, fought: pair.fought, key: `${app.year}-${pair.a}-${pair.d}-${pair.fought}` };
  });
</script>

<div class="karte">
<svg bind:this={svgEl} viewBox="{view.x} {view.y} {view.w} {view.h}" role="img" aria-label="Karte der neun Länder des Reichs"
  class:zoomed={zoom > 1} class:dragging={drag?.moved}
  onpointerdown={onDown} onpointermove={onMove} onpointerup={onUp} onpointercancel={onUp} ondblclick={onDbl}>
  <defs>
    <pattern id="hatch" width="7" height="7" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
      <rect width="7" height="7" fill="#ede2c8" />
      <line x1="0" y1="0" x2="0" y2="7" stroke="#d4c39c" stroke-width="2" />
    </pattern>
    <!-- Bausymbole im 24er-Raster (Mitte unten = Standfläche). Größen relativ zueinander:
         Markt < Mühle < Palast < Kathedrale; Palast und Kathedrale wachsen mit dem Baufortschritt. -->
    <symbol id="sym-markt" viewBox="0 0 24 24">
      <path d="M5 10V21M19 10V21" stroke="#2b2118" stroke-width="1.4" stroke-linecap="round" />
      <rect x="4" y="15" width="16" height="5.5" rx=".6" fill="#b07a43" stroke="#2b2118" stroke-width="1" />
      <circle cx="8" cy="14.2" r="1.5" fill="#b3262b" stroke="#2b2118" stroke-width=".6" /><circle cx="11.5" cy="14.2" r="1.5" fill="#e5b93f" stroke="#2b2118" stroke-width=".6" />
      <circle cx="15" cy="14.2" r="1.5" fill="#2f7a37" stroke="#2b2118" stroke-width=".6" />
      <path d="M2.5 10.5L5 4H19L21.5 10.5Z" fill="#b3262b" stroke="#2b2118" stroke-width="1" stroke-linejoin="round" />
      <path d="M8.2 4L7.4 10.5H10.4L10.8 4ZM13.2 4L13.6 10.5H16.6L15.8 4Z" fill="#fffaf0" />
      <path d="M2.5 10.5Q4 12.4 5.5 10.5Q7 12.4 8.5 10.5Q10 12.4 11.5 10.5Q13 12.4 14.5 10.5Q16 12.4 17.5 10.5Q19 12.4 20.5 10.5" fill="none" stroke="#2b2118" stroke-width=".9" />
    </symbol>
    <symbol id="sym-muehle" viewBox="0 0 24 24">
      <path d="M8 22.5L9.6 10.5H14.4L16 22.5Z" fill="#f2e8d2" stroke="#2b2118" stroke-width="1" stroke-linejoin="round" />
      <path d="M8.8 11L12 7.4L15.2 11Z" fill="#7a1f1f" stroke="#2b2118" stroke-width=".9" stroke-linejoin="round" />
      <path d="M10.8 22.5V19Q12 17.4 13.2 19V22.5Z" fill="#2b2118" /><rect x="11.3" y="13.5" width="1.4" height="2" fill="#2b2118" />
      <!-- Flügel drehen sich langsam (CSS .fluegel), Gitter als feine Linien -->
      <g class="fluegel">
        <g id="sym-fluegel"><path d="M12 9V1" stroke="#2b2118" stroke-width="1.1" /><path d="M12.4 1.6H15.2V7.6H12.4Z" fill="#fffaf0" stroke="#2b2118" stroke-width=".7" /><path d="M12.4 3.6H15.2M12.4 5.6H15.2M13.8 1.6V7.6" stroke="#2b2118" stroke-width=".4" /></g>
        <use href="#sym-fluegel" transform="rotate(90 12 9)" /><use href="#sym-fluegel" transform="rotate(180 12 9)" /><use href="#sym-fluegel" transform="rotate(270 12 9)" />
        <circle cx="12" cy="9" r="1.1" fill="#2b2118" />
      </g>
    </symbol>
    <symbol id="sym-palast" viewBox="0 0 24 24">
      <rect x="1.5" y="12.5" width="7" height="9.5" fill="#f4e7c5" stroke="#2b2118" stroke-width=".9" /><rect x="15.5" y="12.5" width="7" height="9.5" fill="#f4e7c5" stroke="#2b2118" stroke-width=".9" />
      <path d="M1 12.8L5 10.2L9 12.8ZM15 12.8L19 10.2L23 12.8Z" fill="#4f6a8a" stroke="#2b2118" stroke-width=".8" stroke-linejoin="round" />
      <rect x="8.2" y="10.5" width="7.6" height="11.5" fill="#fbf3dc" stroke="#2b2118" stroke-width=".9" />
      <path d="M8.2 10.6Q12 3.6 15.8 10.6Z" fill="#e5b93f" stroke="#2b2118" stroke-width=".9" />
      <path d="M12 5.2V2.6M11 3.4H13" stroke="#2b2118" stroke-width=".8" stroke-linecap="round" />
      <path d="M10.6 22V18.6Q12 16.8 13.4 18.6V22Z" fill="#2b2118" />
      <path d="M3 15h1.6v2H3zM5.6 15h1.6v2H5.6zM16.8 15h1.6v2h-1.6zM19.4 15h1.6v2h-1.6zM9.6 13h1.4v2H9.6zM13 13h1.4v2H13z" fill="#4f6a8a" />
    </symbol>
    <symbol id="sym-kathedrale" viewBox="0 0 24 24">
      <path d="M6.5 22.5V12.5L12 8.5L17.5 12.5V22.5Z" fill="#efe6d2" stroke="#2b2118" stroke-width=".9" stroke-linejoin="round" />
      <rect x="2.8" y="8" width="4.2" height="14.5" fill="#e6dac0" stroke="#2b2118" stroke-width=".9" /><rect x="17" y="8" width="4.2" height="14.5" fill="#e6dac0" stroke="#2b2118" stroke-width=".9" />
      <path d="M2.5 8.2L4.9 .8L7.3 8.2ZM16.7 8.2L19.1 .8L21.5 8.2Z" fill="#6b7f9a" stroke="#2b2118" stroke-width=".8" stroke-linejoin="round" />
      <circle cx="12" cy="14" r="2.2" fill="#b3262b" stroke="#e5b93f" stroke-width=".8" />
      <path d="M12 11.8V16.2M9.8 14H14.2" stroke="#e5b93f" stroke-width=".5" />
      <path d="M10.4 22.5V19.6Q12 17.4 13.6 19.6V22.5Z" fill="#2b2118" />
      <path d="M4.2 11.5V10.4Q4.9 9.4 5.6 10.4V11.5ZM18.4 11.5V10.4Q19.1 9.4 19.8 10.4V11.5ZM4.2 16V14.9Q4.9 13.9 5.6 14.9V16ZM18.4 16V14.9Q19.1 13.9 19.8 14.9V16Z" fill="#2b2118" />
    </symbol>
    <marker id="heerzug-spitze" viewBox="0 0 10 10" refX="7" refY="5" markerWidth="5" markerHeight="5" orient="auto-start-reverse">
      <path d="M0 0L10 5L0 10Z" fill="#7a1f1f" />
    </marker>
  </defs>
  <rect width={map.width} height={map.height} fill="#bfd0d6" />
  <path d={map.neighbours} fill="#e6dcc4" stroke="#cdbb94" stroke-width="0.8" />
  {#each map.lands as land (land.id)}
    {@const owner = owners.get(land.id)}
    {@const part = parts.get(land.id)}
    {#if part}
      <!-- Geteiltes Land: Streifen je Herrscher, auf die Landesform zugeschnitten -->
      <clipPath id="teil-{land.id}"><path d={land.d} /></clipPath>
      <g clip-path="url(#teil-{land.id})">
        {#each part.list as x (x.p.seat)}
          <path class="teil" class:meins={x.p.seat === app.seat} data-tip="{land.name} – {x.p.name}: {x.p.land.toLocaleString('de-DE')} ha"
            fill={x.p.alive ? playerColor(app.players, x.p.seat) : '#c9bba0'} style:d="path('{x.d}')" />
        {/each}
        {#if part.free}
          <!-- Freie, noch nicht vergebene Fläche des Landes -->
          <path class="teil" fill="url(#hatch)" data-tip="{land.name} – frei: {part.freeHa.toLocaleString('de-DE')} ha" style:d="path('{part.free.d}')" />
        {/if}
        <!-- Grenzlinien zwischen den Teilgebieten (auch zur freien Fläche) -->
        {#each part.cuts as cut, ci (ci)}
          <path class="teilgrenze" style:d="path('{cut}')" />
        {/each}
      </g>
      <path d={land.d} fill="none" stroke="#6b5537" stroke-width="1.2" stroke-linejoin="round" />
      {#if myPart && land.id === mine}
        <!-- Eigenes Gebiet: goldene Umrandung nur um den eigenen Streifen (Landesgrenze + eigene Schnittkanten) -->
        <clipPath id="mein-{land.id}"><path class="teil" style:d="path('{myPart.d}')" /></clipPath>
        <g clip-path="url(#mein-{land.id})">
          <path d={land.d} class="mein-rand" />
          <g clip-path="url(#teil-{land.id})"><path class="teil mein-rand" style:d="path('{myPart.d}')" /></g>
        </g>
      {/if}
    {:else}
      <path d={land.d} class="land-area" data-tip="{land.name}{owner ? ` – ${title(owner.title, owner.female)} ${owner.name}` : ' – neutral'}" fill={owner ? (owner.alive ? COUNTRY_COLOR[land.id] : '#c9bba0') : 'url(#hatch)'}
        stroke="#6b5537" stroke-width="1.2" stroke-linejoin="round">
      </path>
    {/if}
  {/each}
  {#if war}
    {#key war.key}
      {@const target = map.lands.find((l) => l.id === war.defender)}
      {#if target}<path d={target.d} class="under-attack" />{/if}
    {/key}
  {/if}
  {#each map.lands as land (land.id)}
    {#if land.id === mine && !parts.has(land.id)}<path d={land.d} fill="none" stroke="#b08a3e" stroke-width="3.5" stroke-linejoin="round" />{/if}
  {/each}
  <path d={map.rivers} fill="none" stroke="#4f78a6" stroke-width="1.1" stroke-linecap="round" opacity="0.85" />
  <path d={map.majorRivers} fill="none" stroke="#3f6a99" stroke-width="2.4" stroke-linecap="round" />
  <!-- Fließendes Wasser: helle Striche wandern über die großen Ströme -->
  <path d={map.majorRivers} class="flow" fill="none" stroke-linecap="round" />
  <path d={map.lakes} fill="#bfd0d6" stroke="#4f78a6" stroke-width="0.6" />
  {#each Object.entries(map.riverLabels) as [name, [x, y]]}
    <text {x} {y} class="river" text-anchor="middle">{name}</text>
  {/each}

  <!-- Bauten in den Gebieten; erscheinen mit kleinem Aufploppen, wenn neu gebaut -->
  <g class="bauten">
    {#each bauten as b (b.key)}
      <!-- Standfläche auf dem Platz, Symbol steht darüber -->
      <use href="#sym-{b.kind}" class="bau" data-tip={b.tip} x={b.x - b.size / 2} y={b.y - b.size * 0.9} width={b.size} height={b.size} />
    {/each}
  </g>

  {#if war}
    {#key war.key}
      <g class="heerzug" class:fought={war.fought} aria-label="Heerzug">
        <path d={war.d} class="route" marker-end="url(#heerzug-spitze)" pathLength="100" />
        <g transform="translate({war.sx} {war.sy})" class="swords">
          <circle r="13" fill="#f6eedb" stroke="#7a1f1f" stroke-width="1.5" />
          <path d="M-7 -7L7 7M7 -7L-7 7M-8 -3L-3 -8M8 -3L3 -8" stroke="#2b2118" stroke-width="2" stroke-linecap="round" />
        </g>
      </g>
    {/key}
  {/if}

  {#each map.lands as land (land.id)}
    {@const owner = owners.get(land.id)}
    {@const part = parts.get(land.id)}
    {#if part}
      <!-- Geteiltes Land: Sitz jedes Herrschers in der Mitte seines Streifens, Name darunter -->
      {#each part.list as x (x.p.seat)}
        {#if x.p.alive && x.p.phase !== 'done'}<circle cx={x.cx} cy={x.cy} r="6" class="pulse" class:me={x.p.seat === app.seat} />{/if}
        <circle cx={x.cx} cy={x.cy} r={x.p.seat === app.seat ? 5 : 3.5} fill={x.p.seat === app.seat ? '#b08a3e' : '#2b2118'} stroke="#2b2118" class="sitz" />
        <text x={x.cx} y={x.cy + 15} class="ruler small" text-anchor="middle">{x.p.name}{x.p.alive ? '' : ' †'}</text>
      {/each}
      <text x={land.label[0]} y={part.s.box.y + 16} class="land" text-anchor="middle">{land.name.toUpperCase().replace('ß', 'SS')}</text>
    {:else}
      {#if owner && owner.alive && owner.phase !== 'done'}
        <!-- Herrscher, die noch am Zug sind: pulsierender Ring um die Hauptstadt -->
        <circle cx={land.capital[0]} cy={land.capital[1]} r="6" class="pulse" class:me={land.id === mine} />
      {/if}
      <circle cx={land.capital[0]} cy={land.capital[1]} r={land.id === mine ? 5 : 3.5} fill={land.id === mine ? '#b08a3e' : '#2b2118'} stroke="#2b2118" />
      <text x={land.label[0]} y={land.label[1] - (owner ? 6 : -4)} class="land" text-anchor="middle">{land.name.toUpperCase().replace('ß', 'SS')}</text>
      {#if owner}
        <text x={land.label[0]} y={land.label[1] + 12} class="ruler" text-anchor="middle">{title(owner.title, owner.female)} {owner.name}{owner.alive ? '' : ' †'}</text>
      {/if}
    {/if}
  {/each}
</svg>
<!-- Zoom-Knöpfe im Spieldesign -->
<div class="zoom" role="group" aria-label="Kartenzoom">
  <button type="button" title="Hineinzoomen" aria-label="Hineinzoomen" disabled={zoom >= MAX_ZOOM - 0.01} onclick={() => zoomAt(1.5)}>+</button>
  <button type="button" title="Herauszoomen" aria-label="Herauszoomen" disabled={zoom <= 1.01} onclick={() => zoomAt(1 / 1.5)}>−</button>
  <button type="button" title="Mein Gebiet zeigen" aria-label="Mein Gebiet zeigen" onclick={zoomMine}>◎</button>
  <button type="button" title="Ganze Karte" aria-label="Ganze Karte" disabled={zoom <= 1.01} onclick={() => setView(0, 0, map.width)}>⤢</button>
</div>
</div>

<style>
  .karte { position: relative; width: 100%; height: 100%; }
  svg { width: 100%; height: 100%; display: block; touch-action: none; user-select: none; }
  svg.zoomed { cursor: grab; }
  svg.dragging { cursor: grabbing; }
  /* Zoom-Knöpfe: kleine Siegelplatten oben rechts in der Karte */
  .zoom { position: absolute; top: 8px; right: 8px; display: flex; flex-direction: column; gap: 4px; }
  .zoom button {
    width: 34px; height: 34px; display: flex; align-items: center; justify-content: center; padding: 0;
    border: 1.5px solid #5a4630; border-radius: 4px; color: var(--ink); cursor: pointer;
    background: linear-gradient(180deg, #fffaf0, #e3cfa4);
    box-shadow: inset 0 0 0 2px #fffaf0aa, inset 0 0 0 3px #b08a3e99, 0 2px 4px #2b211844;
    font-family: var(--font-label); font-size: 20px; font-weight: 700; line-height: 1;
  }
  .zoom button:hover:not(:disabled) { border-color: var(--blood); color: var(--blood); }
  .zoom button:disabled { opacity: .45; cursor: default; }
  .zoom button:focus-visible { outline: 3px solid var(--gold); outline-offset: 2px; }
  text { paint-order: stroke; stroke: #f6eedb; stroke-width: 3px; }
  .land { font-family: var(--font-label); font-size: 15px; font-weight: 700; letter-spacing: 1.5px; fill: var(--ink); }
  .ruler { font-family: var(--font-body); font-size: 14px; font-style: italic; fill: var(--ink); }
  .river { font-family: var(--font-body); font-size: 13px; font-style: italic; fill: var(--prussian); stroke-width: 2.5px; }

  /* Farbwechsel (z. B. Tod eines Herrschers) weich überblenden */
  .land-area { transition: fill 1.2s ease; }
  /* Teilgebiete wandern weich, wenn sich der Landbesitz ändert */
  .teil { transition: d 1.4s ease, fill 1.2s ease; }
  .teilgrenze { fill: none; stroke: #6b5537; stroke-width: 1.4; stroke-dasharray: 5 3; stroke-linejoin: round; transition: d 1.4s ease; }
  .sitz { transition: cx 1.4s ease, cy 1.4s ease; }
  .ruler.small { font-size: 12px; }
  .bau { filter: drop-shadow(0 1px 0 #f6eedb) drop-shadow(0 1px 1px #2b211866); transform-box: fill-box; transform-origin: center; animation: bau .5s ease-out; }
  @keyframes bau { from { transform: scale(0); opacity: 0; } to { transform: scale(1); opacity: 1; } }
  /* Mühlenflügel drehen sich langsam */
  :global(.fluegel) { transform-box: view-box; transform-origin: 12px 9px; animation: fluegel 9s linear infinite; }
  @keyframes fluegel { to { transform: rotate(360deg); } }
  /* Eigener Streifen: goldene Linie */
  .mein-rand { fill: none; stroke: #b08a3e; stroke-width: 3.5; stroke-linejoin: round; }

  /* Fließende Flüsse */
  .flow { stroke: #cfe0ea; stroke-width: 1.2; stroke-dasharray: 3 22; opacity: .8; animation: flow 6s linear infinite; }
  @keyframes flow { to { stroke-dashoffset: -50; } }

  /* Hauptstädte der Herrscher am Zug */
  .pulse { fill: none; stroke: #2b2118; stroke-width: 1.5; transform-box: fill-box; transform-origin: center; animation: pulse 2s ease-out infinite; }
  .pulse.me { stroke: #b08a3e; stroke-width: 2.5; }
  @keyframes pulse { from { transform: scale(1); opacity: .9; } to { transform: scale(3); opacity: 0; } }

  /* Krieg: Ziel-Land blinkt rot, Heerzug zeichnet sich ein, Schwerter erscheinen */
  .under-attack { fill: #7a1f1f; pointer-events: none; animation: alarm 1.4s ease-in-out 3; opacity: 0; }
  @keyframes alarm { 50% { opacity: .35; } }
  .route { fill: none; stroke: #7a1f1f; stroke-width: 4.5; stroke-linecap: round; stroke-dasharray: 100; stroke-dashoffset: 100; animation: draw 1.8s ease-out forwards; }
  .heerzug:not(.fought) .route { animation: draw 1.8s ease-out forwards, march 1.2s linear 1.8s infinite; }
  @keyframes draw { to { stroke-dashoffset: 0; } }
  @keyframes march { from { stroke-dasharray: 4 3; stroke-dashoffset: 0; } to { stroke-dasharray: 4 3; stroke-dashoffset: -7; } }
  .swords { opacity: 0; animation: appear .5s ease-out 1.6s forwards; }
  @keyframes appear { to { opacity: 1; } }

  @media (prefers-reduced-motion: reduce) {
    .flow, .pulse, .under-attack, .bau, :global(.fluegel) { animation: none; }
    .flow { display: none; }
    .route { animation: none; stroke-dashoffset: 0; }
    .swords { animation: none; opacity: 1; }
  }
</style>
