// WebSocket-Verbindung zum Spielserver mit automatischem Wiederverbinden.
// Die Sitzung (Spiel-ID + Token) liegt im sessionStorage: ein Neuladen landet wieder am
// selben Platz, mehrere Tabs können aber getrennte Herrscher spielen.
// Zusätzlich merkt sich der Browser alle eigenen Plätze im localStorage („Meine Spiele“).
import { app, receive, toast } from './state.svelte';
import type { Action, ServerMsg, Settings } from './types';
import { sfxClick, sfxCoin } from './audio.svelte';

const SESSION_KEY = 'kaiser.session';
const GAMES_KEY = 'kaiser.games';
const MAX_GAMES = 20; // wie maxGameRefs auf dem Server
let ws: WebSocket | null = null;
let retry = 0;

interface Session { gameId: string; token: string }

function loadSession(): Session | null {
  try {
    const raw = sessionStorage.getItem(SESSION_KEY);
    return raw ? (JSON.parse(raw) as Session) : null;
  } catch {
    return null;
  }
}

function saveSession(s: Session | null) {
  try {
    if (s) sessionStorage.setItem(SESSION_KEY, JSON.stringify(s));
    else sessionStorage.removeItem(SESSION_KEY);
  } catch {
    /* privater Modus: Wiedereinstieg nach Reload dann nicht möglich */
  }
}

/** Gemerkte Plätze, neueste zuerst. */
function loadGames(): Session[] {
  try {
    const list = JSON.parse(localStorage.getItem(GAMES_KEY) ?? '[]');
    return Array.isArray(list) ? list.filter((g) => typeof g?.gameId === 'string' && typeof g?.token === 'string') : [];
  } catch {
    return [];
  }
}

function saveGames(list: Session[]) {
  try {
    localStorage.setItem(GAMES_KEY, JSON.stringify(list.slice(0, MAX_GAMES)));
  } catch {
    /* Speicher gesperrt: Liste bleibt leer */
  }
}

function rememberGame(s: Session) {
  saveGames([s, ...loadGames().filter((g) => g.gameId !== s.gameId)]);
}

/** Fragt den Stand aller gemerkten Spiele ab (Antwort: app.myGames). */
export function requestGames() {
  const games = loadGames();
  if (!games.length) {
    app.myGames = [];
    return;
  }
  if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ t: 'games', games }));
}

/** Nimmt ein gemerktes Spiel wieder auf: Sitzung setzen und Platz einnehmen. */
export function resumeGame(gameId: string) {
  const g = loadGames().find((x) => x.gameId === gameId);
  if (!g) return;
  saveSession(g);
  send({ t: 'rejoin', gameId: g.gameId, token: g.token });
}

/** Entfernt ein Spiel aus „Meine Spiele“ (nur in diesem Browser; der Platz bleibt bestehen). */
export function forgetGame(gameId: string) {
  saveGames(loadGames().filter((g) => g.gameId !== gameId));
  app.myGames = app.myGames?.filter((g) => g.gameId !== gameId) ?? null;
}

// Platz-Übergabe per Link (#platz=SPIELID.TOKEN), z. B. um eine laufende Partie in einem
// anderen Browser zu übernehmen. Der Hash wird sofort entfernt, damit das Token nicht in der Adresszeile bleibt.
const handover = /^#platz=([A-Z2-9]{6})\.([0-9a-f]{32})$/.exec(location.hash);
if (handover) {
  saveSession({ gameId: handover[1], token: handover[2] });
  history.replaceState(null, '', location.pathname + location.search);
}

/** Baut die Verbindung auf; nach Abbruch erneut mit wachsendem Abstand (max. 10 s). */
export function connect() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  ws = new WebSocket(`${proto}://${location.host}/ws`);
  app.connection = 'connecting';

  ws.onopen = () => {
    app.connection = 'open';
    retry = 0;
    const s = loadSession();
    if (s) send({ t: 'rejoin', gameId: s.gameId, token: s.token });
    else requestGames();
  };
  ws.onmessage = (ev) => {
    let msg: ServerMsg;
    try {
      msg = JSON.parse(ev.data);
    } catch {
      return;
    }
    // Anfrage gestellt: Token merken, damit man nach Neuladen weiter wartet und nach der
    // Zulassung mit genau diesem Token am Platz sitzt
    if (msg.t === 'pending') {
      saveSession({ gameId: msg.gameId, token: msg.token });
      rememberGame({ gameId: msg.gameId, token: msg.token });
    }
    if (msg.t === 'error' && ['denied', 'not_joinable', 'banned'].includes(msg.code) && app.pendingJoin) {
      saveSession(null);
      forgetGame(app.pendingJoin.gameId);
    }
    if (msg.t === 'welcome') {
      saveSession({ gameId: msg.gameId, token: msg.token });
      rememberGame({ gameId: msg.gameId, token: msg.token });
    }
    // Vom Gastgeber aus der Lobby entfernt: Platz vergessen und zurück zur Startseite
    if (msg.t === 'error' && msg.code === 'kicked') {
      saveSession(null);
      forgetGame(app.gameId);
      app.gameId = '';
      app.seat = -1;
      app.host = false;
      app.lobby = [];
    }
    if (msg.t === 'error' && msg.code === 'bad_token') {
      saveSession(null);
      requestGames();
    }
    receive(msg);
  };
  ws.onclose = () => {
    app.connection = 'closed';
    retry = Math.min(retry + 1, 5);
    setTimeout(connect, retry * 2000);
  };
}

function send(msg: object) {
  if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify(msg));
  else toast('error', 'Keine Verbindung zum Server. Es wird neu verbunden …');
}

export const createGame = (name: string, country: number, female: boolean, settings: Settings) =>
  send({ t: 'create', name, country, female, settings });
export const joinGame = (gameId: string, name: string, country: number, female: boolean) =>
  send({ t: 'join', gameId: gameId.trim().toUpperCase(), name, country, female });
export const peekGame = (gameId: string) => send({ t: 'peek', gameId: gameId.trim().toUpperCase() });
export const startGame = () => send({ t: 'start' });
export const addBot = (strategy: string) => send({ t: 'addBot', strategy });
/** Gastgeber lässt eine Beitrittsanfrage zu oder lehnt sie ab. */
export function decideJoin(request: string, accept: boolean) {
  send({ t: 'decide', request, accept });
  app.joinRequests = app.joinRequests.filter((q) => q.id !== request);
}
/** Eigene Anfrage zurückziehen und den gemerkten Platz vergessen. */
export function cancelJoin() {
  send({ t: 'cancelJoin' });
  saveSession(null);
  if (app.pendingJoin) forgetGame(app.pendingJoin.gameId);
  app.pendingJoin = null;
}
/** Gastgeber entfernt einen Herrscher (Mensch oder Computer) vor dem Start aus der Lobby. */
export const kickPlayer = (seat: number) => send({ t: 'kick', seat });
/** Autopilot für den eigenen Herrscher; leere Spielweise schaltet ihn ab. */
export const setAutopilot = (strategy: string) => send({ t: 'autopilot', strategy });
export const act = (action: Action) => {
  // Hörbare Rückmeldung: Münzen bei Käufen, sonst ein leises Klicken
  if (['buyGrain', 'buyLand', 'build', 'recruit', 'sellGrain', 'sellLand'].includes(action.type)) sfxCoin();
  else sfxClick();
  // Für verständliche Fehlermeldungen: welche Aktion zuletzt geschickt wurde
  app.lastAction = action.type;
  send({ t: 'act', action });
};

/** Lädt Spielerbild oder Wappen hoch; der Server prüft, säubert und übernimmt es direkt. */
export async function upload(kind: 'avatar' | 'arms', file: File): Promise<boolean> {
  const s = loadSession();
  if (!s) return false;
  if (file.size > 1024 * 1024) {
    toast('error', 'Die Datei ist zu groß (höchstens 1 MB).');
    return false;
  }
  const form = new FormData();
  form.set('gameId', s.gameId);
  form.set('token', s.token);
  form.set('kind', kind);
  form.set('file', file);
  try {
    const res = await fetch('/upload', { method: 'POST', body: form });
    const body = await res.json().catch(() => ({}));
    if (!res.ok) {
      toast('error', body.message ?? 'Hochladen fehlgeschlagen.');
      return false;
    }
    toast('info', kind === 'avatar' ? 'Spielerbild übernommen.' : 'Wappen übernommen.');
    return true;
  } catch {
    toast('error', 'Keine Verbindung zum Server.');
    return false;
  }
}

/** Verlässt das Spiel lokal (Sitz bleibt auf dem Server erhalten). */
export function leaveGame() {
  saveSession(null);
  location.reload();
}
