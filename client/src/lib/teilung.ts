// Geteilte Länder auf der Reichskarte: ein Land wird nach den Hektaranteilen seiner Herrscher
// in Streifen zerlegt. Grundlage sind Stichproben-Punkte im Landespolygon (einmal je Land
// berechnet); die Schnitte liegen dort, wo die Punktanzahl dem Flächenanteil entspricht.
// Die Kartenpfade bestehen nur aus absoluten M/L/Z-Befehlen (scripts/map/export_map_json.py).

export interface LandSample {
  /** Streifen laufen quer zur längeren Ausdehnung des Landes */
  horizontal: boolean;
  /** Koordinaten entlang der Schnittachse, sortiert */
  axis: number[];
  /** Punkte in derselben Reihenfolge wie axis (für Schwerpunkte) */
  pts: [number, number][];
  box: { x: number; y: number; w: number; h: number };
}

/** n = Zahl der Stichproben im Streifen, ein Maß für seine Fläche (je Punkt 9 Kartenpunkte²) */
export interface Slice { from: number; to: number; cx: number; cy: number; n: number }

/** Ringe (Polygone) aus einem M/L/Z-Pfad. */
function rings(d: string): [number, number][][] {
  const out: [number, number][][] = [];
  let cur: [number, number][] = [];
  for (const m of d.matchAll(/([MLZ])([^MLZ]*)/g)) {
    if (m[1] === 'M' && cur.length) { out.push(cur); cur = []; }
    if (m[1] === 'Z') { if (cur.length) out.push(cur); cur = []; continue; }
    const n = m[2].trim().split(/[\s,]+/).map(Number);
    for (let i = 0; i + 1 < n.length; i += 2) cur.push([n[i], n[i + 1]]);
  }
  if (cur.length) out.push(cur);
  return out;
}

/** Gerade-ungerade-Regel wie bei SVG: liegt (x, y) im Land? */
function inside(rs: [number, number][][], x: number, y: number): boolean {
  let hit = false;
  for (const r of rs) {
    for (let i = 0, j = r.length - 1; i < r.length; j = i++) {
      const [xi, yi] = r[i], [xj, yj] = r[j];
      if ((yi > y) !== (yj > y) && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi) hit = !hit;
    }
  }
  return hit;
}

const cache = new Map<string, LandSample>();

/** Stichproben eines Landes (Raster 3 Kartenpunkte), zwischengespeichert je Pfad. */
export function sampleLand(d: string): LandSample {
  const hit = cache.get(d);
  if (hit) return hit;
  const rs = rings(d);
  const all = rs.flat();
  const xs = all.map((p) => p[0]), ys = all.map((p) => p[1]);
  const box = { x: Math.min(...xs), y: Math.min(...ys), w: 0, h: 0 };
  box.w = Math.max(...xs) - box.x;
  box.h = Math.max(...ys) - box.y;
  const horizontal = box.h > box.w;
  const pts: [number, number][] = [];
  for (let y = box.y + 1.5; y < box.y + box.h; y += 3) {
    for (let x = box.x + 1.5; x < box.x + box.w; x += 3) if (inside(rs, x, y)) pts.push([x, y]);
  }
  const k = horizontal ? 1 : 0;
  pts.sort((a, b) => a[k] - b[k]);
  const s = { horizontal, axis: pts.map((p) => p[k]), pts, box };
  cache.set(d, s);
  return s;
}

/** Streifen je Anteil (Summe beliebig): Grenzen entlang der Achse plus Schwerpunkt für die Beschriftung. */
export function slices(s: LandSample, shares: number[]): Slice[] {
  const total = shares.reduce((a, b) => a + b, 0) || 1;
  const n = s.axis.length;
  const lo = s.horizontal ? s.box.y : s.box.x;
  const hi = lo + (s.horizontal ? s.box.h : s.box.w);
  const out: Slice[] = [];
  let acc = 0, start = 0, from = lo;
  shares.forEach((sh, i) => {
    acc += sh;
    const end = i === shares.length - 1 ? n : Math.round((acc / total) * n);
    const to = i === shares.length - 1 ? hi : end > 0 && end < n ? (s.axis[end - 1] + s.axis[end]) / 2 : from;
    const part = s.pts.slice(start, Math.max(start + 1, end));
    const mx = part.reduce((a, p) => a + p[0], 0) / (part.length || 1);
    const my = part.reduce((a, p) => a + p[1], 0) / (part.length || 1);
    // Der Schwerpunkt kann bei gebogenen oder zerrissenen Ländern außerhalb liegen:
    // auf den nächsten Punkt einrasten, der sicher im eigenen Streifen und im Land liegt.
    let [cx, cy] = part[0] ?? [mx, my];
    for (const p of part) if (Math.hypot(p[0] - mx, p[1] - my) < Math.hypot(cx - mx, cy - my)) [cx, cy] = p;
    out.push({ from, to, cx, cy, n: Math.max(0, end - start) });
    start = end;
    from = to;
  });
  return out;
}

/** Farbe mischen (#rrggbb), t = Anteil der zweiten Farbe. */
export function mix(a: string, b: string, t: number): string {
  const p = (h: string) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16));
  const [x, y] = [p(a), p(b)];
  return '#' + x.map((v, i) => Math.round(v + (y[i] - v) * t).toString(16).padStart(2, '0')).join('');
}

/** Stabile Plätze für Bausymbole in einem Streifen: Punkte in fester Zufallsreihenfolge (je Land
 *  gleich, damit Symbole nicht springen), mit Mindestabstand zueinander und zur Beschriftung. */
export function spots(s: LandSample, from: number, to: number, n: number, avoid: [number, number][], seed: number, gap = 9): [number, number][] {
  const out: [number, number][] = [];
  if (n <= 0 || !s.pts.length) return out;
  const k = s.horizontal ? 1 : 0;
  // Fester Durchlauf: Index-Sprung mit zur Punktanzahl teilerfremder Schrittweite
  const len = s.pts.length;
  let step = 7919 + seed * 104729;
  while (gcd(step % len || 1, len) !== 1) step++;
  for (let i = 0, idx = seed % len; i < len && out.length < n; i++, idx = (idx + step) % len) {
    const p = s.pts[idx];
    if (p[k] < from + 8 || p[k] > to - 8) continue; // Abstand zu den geschwungenen Grenzen
    if (avoid.some(([x, y]) => Math.abs(p[0] - x) < 30 && Math.abs(p[1] - y) < 16)) continue;
    if (out.some(([x, y]) => Math.hypot(p[0] - x, p[1] - y) < gap)) continue;
    out.push(p);
  }
  return out;
}

function gcd(a: number, b: number): number {
  return b ? gcd(b, a % b) : a;
}

/** Kurve quer zur Schnittachse an Position pos (Kartenpunkte), feste Wellen je seed.
 *  amp = 0 ergibt eine gerade Kante (Außenränder). Alle Kurven haben gleich viele Punkte,
 *  damit die Grenzen per CSS-Übergang (d) weich wandern können. */
export function curve(s: LandSample, pos: number, seed: number, amp: number, steps = 28): [number, number][] {
  const a0 = (s.horizontal ? s.box.x : s.box.y) - 6;
  const a1 = (s.horizontal ? s.box.x + s.box.w : s.box.y + s.box.h) + 6;
  // Zwei überlagerte Sinuswellen mit festen, je Grenze verschiedenen Frequenzen und Phasen
  const f1 = 1.3 + (seed % 5) * 0.37, f2 = 3.1 + (seed % 7) * 0.53;
  const p1 = seed * 1.7, p2 = seed * 2.9;
  const out: [number, number][] = [];
  for (let i = 0; i <= steps; i++) {
    const t = i / steps;
    const off = amp * (0.65 * Math.sin(t * Math.PI * 2 * f1 + p1) + 0.35 * Math.sin(t * Math.PI * 2 * f2 + p2));
    const across = a0 + (a1 - a0) * t;
    out.push(s.horizontal ? [across, pos + off] : [pos + off, across]);
  }
  return out;
}

const r1 = (n: number) => Math.round(n * 10) / 10;

/** Fläche zwischen zwei Kurven als Pfad (für Streifen). */
export function band(a: [number, number][], b: [number, number][]): string {
  const pts = [...a, ...[...b].reverse()];
  return 'M' + pts.map(([x, y]) => `${r1(x)} ${r1(y)}`).join('L') + 'Z';
}

/** Kurve als offene Linie (für die gestrichelten Grenzen). */
export function line(a: [number, number][]): string {
  return 'M' + a.map(([x, y]) => `${r1(x)} ${r1(y)}`).join('L');
}
