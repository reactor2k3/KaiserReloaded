// Zentraler Client-Zustand mit Svelte-5-Runes. Nachrichten vom Server werden
// hineingemischt statt ersetzt, damit nur geänderte Stellen neu gezeichnet werden.
import type { Field, GameEvent, LobbyPlayer, PublicPlayer, SelfView, ServerMsg, Settings, WarResult, GameStatus, EventReport, RiskQuestion, Pact, JoinRequest } from './types';
import { neueAnfrage } from './benachrichtigung';
import { describeEvent } from './format';
import { sfxBell, sfxChime, sfxError, sfxFanfare } from './audio.svelte';

export interface Toast { id: number; kind: 'info' | 'warn' | 'error'; text: string }
export interface ChronicleEntry { id: number; year: number; text: string }

export const app = $state({
  connection: 'connecting' as 'connecting' | 'open' | 'closed',
  lastAction: '', // zuletzt geschickte Aktion (Fehlermeldung „geht in dieser Phase nicht“)
  gameId: '',
  seat: -1,
  host: false,
  started: false,
  settings: null as Settings | null,
  lobby: [] as LobbyPlayer[],
  preview: null as { gameId: string; started: boolean; joinable: boolean; settings: Settings; players: LobbyPlayer[] } | null,
  pendingJoin: null as { gameId: string; host: string } | null, // eigene Beitrittsanfrage wartet auf den Gastgeber
  joinRequests: [] as JoinRequest[], // als Gastgeber: offene Anfragen anderer
  year: 0,
  over: false,
  winner: -1,
  players: [] as PublicPlayer[],
  self: null as SelfView | null,
  chronicle: [] as ChronicleEntry[],
  // Letzte Schlacht für den Schlachtbericht (Modal); enthält Feld, Einheiten und Schritte für die Wiedergabe.
  lastBattle: null as null | { year: number; historic?: boolean; attacker: number; defender: number; result: WarResult; field: Field; units: unknown[]; steps: unknown[] },
  toasts: [] as Toast[],
  // Eigene Ereignisse für das Ereignis-Modal (Warteschlange) und die offene Risiko-Frage
  ereignisse: [] as EventReport[],
  risiko: null as RiskQuestion | null,
  risikoZeigen: false, // Wunsch, die Risiko-Frage erneut zu öffnen (Knopf in der Aktionsspalte)
  // Hektar je Land: Grundlage für die Besitzanteile auf der Karte
  countryArea: 50000,
  // Verteidigungsabkommen (öffentlich)
  pacts: [] as Pact[],
  // „Meine Spiele“ auf der Startseite; null = noch nicht abgefragt
  myGames: null as GameStatus[] | null,
});

let nextId = 1;

/** Kurzer Hinweis oben rechts, verschwindet nach einigen Sekunden von selbst. */
/** Spielt der Computer gerade für den eigenen Herrscher? Dann öffnen sich keine Hinweisfenster
 *  (Ereignisse, Risiko, Schlachten, Rat); alles steht weiter in der Chronik. */
export const autopilotAn = () => !!app.players[app.seat]?.bot;

export function toast(kind: Toast['kind'], text: string, ms = 5000) {
  const id = nextId++;
  app.toasts.push({ id, kind, text });
  setTimeout(() => {
    const i = app.toasts.findIndex((t) => t.id === id);
    if (i >= 0) app.toasts.splice(i, 1);
  }, ms);
}

/** Überträgt Felder rekursiv in ein bestehendes Objekt, damit Proxys und Bindungen erhalten bleiben. */
function merge(target: any, src: any) {
  for (const [k, v] of Object.entries(src)) {
    if (v && typeof v === 'object' && !Array.isArray(v) && target[k] && typeof target[k] === 'object') {
      merge(target[k], v);
    } else if (target[k] !== v) {
      target[k] = v;
    }
  }
}

/** Verarbeitet eine Server-Nachricht. */
export function receive(msg: ServerMsg) {
  switch (msg.t) {
    case 'welcome':
      app.pendingJoin = null;
      app.gameId = msg.gameId;
      app.seat = msg.seat;
      app.host = msg.host;
      return msg;
    case 'preview':
      app.preview = { gameId: msg.gameId, started: msg.started, joinable: msg.joinable, settings: msg.settings, players: msg.players };
      return;
    case 'lobby':
      app.gameId = msg.gameId;
      app.started = msg.started;
      app.settings = msg.settings;
      app.lobby = msg.players;
      return;
    case 'public':
      app.year = msg.year;
      app.over = msg.over;
      app.winner = msg.winner;
      if (msg.countryArea) app.countryArea = msg.countryArea;
      app.pacts = msg.pacts ?? [];
      if (msg.full) app.players = msg.players;
      else for (const p of msg.players) {
        if (app.players[p.seat]) merge(app.players[p.seat], p);
        else app.players[p.seat] = p;
      }
      return;
    case 'self':
      if (app.self) {
        merge(app.self, msg.self);
        // Jahreswerte und Wappen vollständig übernehmen: leere Felder (erledigte Entscheidung) dürfen nicht alt stehen bleiben
        app.self.player.year = msg.self.player.year;
        app.self.player.arms = msg.self.player.arms;
        if (!msg.self.war) app.self.war = undefined; // Kriegsausschnitt entfällt nach Diplomatie/Aufstellung
      } else app.self = msg.self;
      return;
    case 'events':
      for (const e of msg.events) addEvent(e);
      return;
    case 'history':
      // Wiedereinstieg: Chronik neu aufbauen, ohne Hinweise oder Schlachtbericht erneut zu zeigen
      app.chronicle = [];
      app.lastBattle = null;
      for (const e of msg.events ?? []) addEvent(e, true);
      return;
    case 'pending':
      app.pendingJoin = { gameId: msg.gameId, host: msg.host };
      return;
    case 'requests':
      // Neue Anfrage: Ton und (bei verstecktem Tab) Browser-Meldung für den Gastgeber
      if (msg.requests.some((q) => !app.joinRequests.some((o) => o.id === q.id))) {
        sfxChime();
        neueAnfrage(msg.requests[msg.requests.length - 1].name);
      }
      app.joinRequests = msg.requests;
      return;
    case 'games':
      app.myGames = msg.games;
      return;
    case 'error':
      // Kein Fehlerton für stille Hintergrundfälle (alter Platz beim Wiederverbinden ungültig)
      if (!['bad_token', 'no_game'].includes(msg.code)) sfxError();
      // Anfrage abgelehnt oder Beitritt nicht mehr möglich: Wartezustand beenden
      if (['denied', 'not_joinable', 'banned', 'kicked'].includes(msg.code)) app.pendingJoin = null;
      // Phasenfehler entstehen meist, wenn die Zugzeit abläuft, während man noch klickt: Aktion nennen
      if (msg.code === 'wrong_phase') {
        const was = ACTION_LABEL[app.lastAction] ?? app.lastAction;
        toast('error', `„${was}“ geht jetzt nicht mehr: Ihre Phase ist schon weiter (Zugzeit abgelaufen?).`);
        return;
      }
      toast('error', msg.message);
      return;
  }
}

// Texte der Umstürze (Kaiser II, frei nacherzählt): [Titel, Text]
const UNREST_TEXT: Record<string, [string, string]> = {
  'revolution/niedergeschlagen': ['Revolution!', 'Armut und Unmut über Ihre Politik ließen revolutionäre Ideen aufkeimen. Mit Hilfe der Gendarmerie konnten Sie den Aufstand eindämmen und die Anführer festsetzen. Reformieren Sie Ihre Innenpolitik, bevor es erneut brennt.'],
  'revolution/pluenderung': ['Revolution!', 'Der Aufstand ließ sich nicht verhindern. Plündernde Haufen zogen durchs Land, viele Untertanen flohen, und die Staatskasse wurde geplündert. Nur knapp konnten Sie sich an der Macht halten.'],
  'revolution/exil': ['Revolution!', 'Die Stimmung im Volk hat einen bedenklichen Punkt erreicht, und Ihre Gegner haben es genutzt. Selbst Gendarmerie und Militär konnten den Volksaufstand nicht zerschlagen. Sie müssen ins Exil gehen, für Sie ist das Spiel zu Ende.'],
  'putsch/niedergeschlagen': ['Aufstand der Militärs', 'Unzufrieden mit Sold und Moral haben Teile der Armee geputscht. Den treuen Truppen gelang es unter großen Verlusten, die Aufrührer zurückzuschlagen. Halten Sie künftig die Moral der Armee hoch.'],
  'putsch/exil': ['Militärputsch!', 'Die Generäle haben die Macht übernommen. Unter Androhung der Todesstrafe zwingen sie Sie ins Exil. Für Sie ist das Spiel zu Ende. Gehabt Euch wohl …'],
  'attentat/vereitelt': ['Ein Attentat', 'Bei einem Ball zieht einer Ihrer Minister ein Messer. Ein Gendarm ist sofort zur Stelle und vereitelt den Anschlag. Die Verschwörer wurden verhaftet, ihre Ämter neu besetzt.'],
  'attentat/verletzt': ['Ein Attentat!', 'Bei Ihrer Geburtstagsparade springt ein Offizier auf die Tribüne und sticht Ihnen einen Dolch in die Brust. Sie überleben schwer verletzt, doch die Wunde wird Sie Lebensjahre kosten.'],
};

/** Anzeigenamen der Aktionen für Fehlermeldungen */
const ACTION_LABEL: Record<string, string> = {
  buyGrain: 'Korn kaufen', sellGrain: 'Korn verkaufen', buyLand: 'Land kaufen', sellLand: 'Land verkaufen',
  risk: 'Entscheidung', next: 'Weiter', feed: 'Korn verteilen', setTaxes: 'Steuern ändern', fest: 'Volksfest',
  opposition: 'Opposition verhaften', gendarm: 'Gendarmen anwerben', build: 'Bauen', recruit: 'Truppen anwerben',
  maneuver: 'Manöver', train: 'Ausbildung', equip: 'Material', endTurn: 'Jahr beenden', declareWar: 'Krieg erklären',
  proposePact: 'Abkommen anbieten', cancelPact: 'Abkommen kündigen', stance: 'Haltung im Krieg', deploy: 'Aufstellung',
  takeLoan: 'Kredit aufnehmen', repayLoan: 'Kredit tilgen', buyReichBonds: 'Reichsanleihen', sellReichBonds: 'Reichsanleihen',
  buyBonds: 'Anleihen', sellBonds: 'Anleihen', gift: 'Spende', lend: 'Kredit geben', repayPeer: 'Schulden zurückzahlen',
  buildShip: 'Schiffbau', fleet: 'Flotte', colonize: 'Kolonialland', navalAttack: 'Kriegsflotte',
};

/** Ereignisse landen in der Chronik; eigene wichtige Ereignisse zusätzlich als Hinweis. */
function addEvent(e: GameEvent, historic = false) {
  // Nur Schlachten mit Wiedergabedaten taugen für Bericht und Heerzug auf der Karte
  if (e.type === 'battle' && e.data?.steps) app.lastBattle = { year: e.year, historic, ...e.data };
  // Kaiser-II-Ereignisse: eigene als Modal, offene Risiko-Frage merken (auch aus der Chronik nach Neuladen)
  if (e.player === app.seat && e.type === 'risiko') app.risiko = e.data as RiskQuestion;
  if (e.player === app.seat && e.type === 'ereignis' && !historic) app.ereignisse.push(e.data as EventReport);
  if (e.player === app.seat && e.type === 'stimmung' && e.data?.unrest && !historic) {
    const t = UNREST_TEXT[`${e.data.unrest}/${e.data.result}`];
    if (t) app.ereignisse.push({ id: e.data.unrest, title: t[0], text: t[1], people: -(e.data.people ?? 0) || undefined,
      soldiers: -(e.data.troops ?? 0) || undefined, years: e.data.years || undefined });
  }
  const text = describeEvent(e, app.players, app.seat);
  if (!text) return;
  app.chronicle.unshift({ id: nextId++, year: e.year, text });
  if (app.chronicle.length > 150) app.chronicle.length = 150;
  if (historic) return;
  // Klänge zu wichtigen Ereignissen
  if (e.type === 'newYear') sfxChime();
  if (e.type === 'died') sfxBell();
  if (e.type === 'yearEnd' && e.data?.promoted && e.player === app.seat) sfxFanfare();
  if (e.type === 'yearEnd' && e.data?.emperor) sfxFanfare();
  if (e.player === app.seat && ['died', 'checks', 'timeout'].includes(e.type)) toast('warn', text, 8000);
  if (e.type === 'yearEnd' && e.player === app.seat && e.data?.promoted) toast('info', text, 8000);
}
