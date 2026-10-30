// Ton und Musik, komplett per Web Audio erzeugt (keine Audiodateien, keine Lizenzfragen).
// Drei Stufen: aus, nur Geräusche, Geräusche und Musik. Die Wahl merkt sich der Browser
// (localStorage, nur Komfort). Der AudioContext entsteht erst nach einer Nutzeraktion,
// weil Browser vorher keinen Ton erlauben.

export type SoundLevel = 'off' | 'sfx' | 'all';

const KEY = 'kaiser.sound';
function loadLevel(): SoundLevel {
  try {
    const v = localStorage.getItem(KEY);
    return v === 'off' || v === 'all' ? v : 'sfx';
  } catch {
    return 'sfx';
  }
}

export const sound = $state({ level: loadLevel() as SoundLevel });

let ctx: AudioContext | null = null;
let sfx: GainNode, music: GainNode;
let musicTimer: ReturnType<typeof setInterval> | undefined;
let lastSfx = 0;

/** Audio erst bei Bedarf anlegen; Browser erlauben das nur nach einer Nutzeraktion. */
function audio(): AudioContext | null {
  if (typeof window === 'undefined' || !('AudioContext' in window)) return null;
  if (!ctx) {
    ctx = new AudioContext();
    const master = ctx.createGain();
    master.gain.value = 0.5;
    master.connect(ctx.destination);
    sfx = ctx.createGain();
    sfx.gain.value = 0.7;
    sfx.connect(master);
    music = ctx.createGain();
    music.gain.value = 0.18;
    music.connect(master);
  }
  if (ctx.state === 'suspended') ctx.resume();
  return ctx;
}

/** Ein Ton mit kurzer Hüllkurve. */
function tone(out: AudioNode, freq: number, at: number, dur: number, type: OscillatorType = 'triangle', vol = 0.3, filter = 0) {
  const c = ctx!;
  const o = c.createOscillator();
  const g = c.createGain();
  o.type = type;
  o.frequency.setValueAtTime(freq, at);
  g.gain.setValueAtTime(0.0001, at);
  g.gain.exponentialRampToValueAtTime(vol, at + 0.012);
  g.gain.exponentialRampToValueAtTime(0.0001, at + dur);
  let node: AudioNode = o;
  if (filter) {
    const f = c.createBiquadFilter();
    f.type = 'lowpass';
    f.frequency.value = filter;
    o.connect(f);
    node = f;
  } else {
    o.connect(g);
  }
  if (filter) node.connect(g);
  g.connect(out);
  o.start(at);
  o.stop(at + dur + 0.05);
}

/** Rauschstoß für Kanonen und Einschläge. */
function noise(at: number, dur: number, vol: number, cutoff: number) {
  const c = ctx!;
  const buf = c.createBuffer(1, Math.ceil(c.sampleRate * dur), c.sampleRate);
  const data = buf.getChannelData(0);
  for (let i = 0; i < data.length; i++) data[i] = (Math.random() * 2 - 1) * (1 - i / data.length);
  const src = c.createBufferSource();
  src.buffer = buf;
  const f = c.createBiquadFilter();
  f.type = 'lowpass';
  f.frequency.value = cutoff;
  const g = c.createGain();
  g.gain.value = vol;
  src.connect(f).connect(g).connect(sfx);
  src.start(at);
}

/** Gemeinsamer Einstieg für Geräusche; drosselt sehr schnelle Folgen (z. B. Schlacht im Schnelllauf). */
function play(fn: (t: number) => void, minGap = 60) {
  if (sound.level === 'off') return;
  const now = performance.now();
  if (now - lastSfx < minGap) return;
  lastSfx = now;
  const c = audio();
  if (c) fn(c.currentTime + 0.01);
}

// Note → Frequenz (A4 = 440 Hz)
const hz = (semi: number) => 440 * Math.pow(2, semi / 12);
const D4 = -7, Fs4 = -3, A4 = 0, D5 = 5, Fs5 = 9, A5 = 12; // Töne für Glocke, Münze, Fanfare

export const sfxClick = () => play((t) => tone(sfx, 660, t, 0.06, 'square', 0.05, 2400), 40);
export const sfxCoin = () => play((t) => { tone(sfx, hz(A5), t, 0.12, 'triangle', 0.18); tone(sfx, hz(A5 + 5), t + 0.07, 0.22, 'triangle', 0.16); });
export const sfxChime = () => play((t) => [D5, Fs5, A5].forEach((n, i) => tone(sfx, hz(n), t + i * 0.12, 0.6, 'sine', 0.12)));
/** Fanfare für Titelaufstieg und Kaiserkrönung: Dreiklang wie aus Naturtrompeten */
export const sfxFanfare = () => play((t) => {
  [[D4, 0, 0.18], [Fs4, 0.18, 0.18], [A4, 0.36, 0.18], [D5, 0.54, 0.7]].forEach(([n, at, d]) => {
    tone(sfx, hz(n), t + at, d, 'sawtooth', 0.12, 1800);
    tone(sfx, hz(n - 12), t + at, d, 'sawtooth', 0.06, 900);
  });
}, 0);
/** Totenglocke */
export const sfxBell = () => play((t) => { tone(sfx, hz(-17), t, 2.2, 'sine', 0.25); tone(sfx, hz(-17) * 2.76, t, 1.2, 'sine', 0.06); }, 0);
export const sfxCannon = () => play((t) => { noise(t, 0.5, 0.6, 700); tone(sfx, 55, t, 0.4, 'sine', 0.4); }, 90);
export const sfxHit = () => play((t) => { noise(t, 0.12, 0.35, 2500); tone(sfx, 180, t, 0.1, 'square', 0.06, 900); }, 70);
export const sfxCrash = () => play((t) => { noise(t, 0.35, 0.45, 1500); }, 90);
export const sfxError = () => play((t) => { tone(sfx, 220, t, 0.18, 'square', 0.06, 900); tone(sfx, 165, t + 0.12, 0.25, 'square', 0.06, 900); }, 0);

// Hintergrundmusik: ruhige barocke Kadenz in D-Dur, arpeggiert wie ein Cembalo.
// I – V – vi – IV – I – IV – V – I, je Takt vier Achtel aufwärts, darunter der Basston.
// Halbtöne relativ zu A4: D-Dur [D F# A D], A-Dur [E A C# E], h-Moll [D F# B D], G-Dur [D G B D]
const Dd = [-7, -3, 0, 5], Ad = [-5, 0, 4, 7], Hm = [-7, -3, 2, 5], Gd = [-7, -2, 2, 5];
const CHORDS = [Dd, Ad, Hm, Gd, Dd, Gd, Ad, Dd];
// Basstöne D3, A2, H2, G2 …
const BASS = [-19, -24, -22, -26, -19, -26, -24, -19];
const BEAT = 0.42; // Sekunden je Achtel

function startMusic() {
  const c = audio();
  if (!c || musicTimer) return;
  let bar = 0;
  let next = c.currentTime + 0.1;
  // Vorausplanung: immer einen Takt im Voraus einreihen, damit es nicht ruckelt
  const schedule = () => {
    while (next < c.currentTime + 1.2) {
      const chord = CHORDS[bar % CHORDS.length];
      tone(music, hz(BASS[bar % BASS.length]), next, BEAT * 4, 'triangle', 0.22, 600);
      chord.forEach((n, i) => tone(music, hz(n), next + i * BEAT, BEAT * 1.6, 'sawtooth', 0.09, 1600));
      next += BEAT * 4;
      bar++;
    }
  };
  schedule();
  musicTimer = setInterval(schedule, 400);
}

function stopMusic() {
  clearInterval(musicTimer);
  musicTimer = undefined;
}

// Musik nur während einer laufenden Partie (nicht auf der Startseite oder nach Spielende)
let musicAllowed = false;

/** Vom Spielbildschirm gesetzt: Musik erlaubt, solange eine Partie läuft. */
export function setMusicAllowed(on: boolean) {
  musicAllowed = on;
  if (on && sound.level === 'all') startMusic();
  else stopMusic();
}

/** Nächste Stufe: aus → Geräusche → Geräusche und Musik → aus. */
export function cycleSound() {
  sound.level = sound.level === 'off' ? 'sfx' : sound.level === 'sfx' ? 'all' : 'off';
  try { localStorage.setItem(KEY, sound.level); } catch { /* privater Modus */ }
  if (sound.level === 'all' && musicAllowed) startMusic();
  else stopMusic();
  if (sound.level !== 'off') sfxClick();
}

/** Nach der ersten Nutzeraktion Musik starten, falls gewünscht (Autoplay-Regeln der Browser). */
export function resumeAudioOnGesture() {
  if (typeof window === 'undefined') return;
  const go = () => {
    if (sound.level === 'all' && musicAllowed) startMusic();
    window.removeEventListener('pointerdown', go);
    window.removeEventListener('keydown', go);
  };
  window.addEventListener('pointerdown', go);
  window.addEventListener('keydown', go);
  // Verborgener Tab: Ton anhalten, beim Zurückkehren fortsetzen
  document.addEventListener('visibilitychange', () => {
    if (!ctx) return;
    if (document.hidden) ctx.suspend();
    else ctx.resume();
  });
}
