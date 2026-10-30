// Typen des WebSocket-Protokolls, gespiegelt aus game/server/internal/server und engine.

export type Phase = 'market' | 'feed' | 'taxes' | 'build' | 'diplomacy' | 'deploy' | 'done';

export interface ArmsSpec {
  /** Landeswappen oder Upload; "builder" kann aus alten Spielständen kommen und gilt als Landeswappen */
  kind: 'country' | 'upload' | 'builder';
  /** Landeswappen: gewählte Variante (0 = Vorgabe) */
  variant?: number;
  media: string;
}

export interface Field { rows: number; cols: number; border: number[]; cells: string[] }
export interface Placement { kind: string; row: number; col: number }
/** Einheit auf dem Schlachtfeld (Wiedergabe und Aufstellung) */
export interface FieldUnit { id: number; side: number; kind: string; row: number; col: number; dead?: boolean }
export interface WarInfo {
  attacker: number; defender: number; side: number;
  route?: number[]; field?: Field; counts?: Record<string, number>;
}
export interface SideLosses { cavalry: number; artillery: number; infantry: number; militia: number; markets: number; mills: number; palace: number; cathedral: number }
export interface WarResult {
  blocked: boolean; land: number; losses: [SideLosses, SideLosses];
  treasuryLoss: [number, number]; populationLoss: [number, number]; bonus: number; compensation: Record<string, number>;
}

export interface Settings {
  phaseSeconds: number;
  timePenalty: boolean;
  battleMode: 'tactical' | 'quick';
  classic: boolean;
  /** Sterblichkeit der Herrscher (engine/settings.go); fehlt bei alten Spielen = normal */
  mortality: Mortality;
  /** Spielstärke der Computer-Herrscher; fehlt bei alten Spielen = schwer */
  difficulty: Difficulty;
  /** Spielmodus: live (kurze Zugzeiten) oder async (Stunden bis Tage); fehlt bei alten Spielen = live */
  mode: 'live' | 'async';
}

export type Difficulty = 'leicht' | 'mittel' | 'schwer';

export type Mortality = 'unsterblich' | 'niedrig' | 'normal' | 'hoch';

export interface Player {
  name: string; country: number; female: boolean;
  land: number; treasury: number; population: number; grain: number; title: number;
  customs: number; vat: number; incomeTax: number; justice: number;
  deathYear: number; suspendedUntil: number; joined?: number;
  markets: number; mills: number; palace: number; cathedral: number;
  cavalry: number; artillery: number; infantry: number; militia: number;
  soldiers: number; mercenaries: number; morale: number;
  nobility: number; clergy: number; burghers: number; trade: number;
  /** Volksstimmung 0–10 und Gendarmen (Kaiser II, engine/stimmung.go) */
  mood: number; moodInit: boolean; gendarmes: number;
  /** Finanzen (Kaiser II): Bankkredit, Reichsanleihen, Steuern des Vorjahres */
  loan?: number; reichBonds?: number; lastTax?: number;
  /** Flotte und Kolonien (Kaiser II): Schiffe insgesamt, je Kontinent Fläche und Kolonialflotte */
  traders?: number; warships?: number; colonies?: Colony[];
  /** Ausbildung und Material des Heeres 0–10 (engine/heer.go) */
  training: number; equipment: number;
  motto: string; avatar: string; arms: ArmsSpec;
  year: {
    weather: number; grainNeed: number; landPrice: number; grainPrice: number;
    grainBought: number; fed: boolean;
    /** offene Risiko-Entscheidung (Kaiser II), Antwort in der Marktphase */
    risk?: string;
    opposition?: boolean;
    shipsBuilt?: number; navalAttack?: boolean;
  };
}

/** Zufallsereignis oder Ausgang einer Risiko-Entscheidung (engine/events.go) */
export interface EventReport {
  id: string; title: string; text: string; risk?: boolean;
  taler?: number; land?: number; soldiers?: number; grain?: number; people?: number; years?: number; ships?: number;
}

/** Kolonie auf einem Kontinent (0 Afrika, 1 Amerika, 2 Asien) samt Kolonialflotte */
export interface Colony { area: number; traders: number; warships: number }

/** Offene Risiko-Entscheidung */
export interface RiskQuestion { id: string; title: string; question: string }

/** Verteidigungsabkommen zweier Sitze (engine/buendnis.go) */
export interface Pact { a: number; b: number }

export interface TaxReport { customs: number; vat: number; incomeTax: number; justice: number; total: number }

export interface SelfView {
  seat: number;
  player: Player;
  phase: Phase;
  deadline: number;
  feedMin: number;
  feedMax: number;
  /** freie Fläche im eigenen (geteilten) Land, -1 = unbegrenzt */
  freeLand: number;
  /** Sitze, die ein Abkommen anbieten */
  pactOffers?: number[] | null;
  /** eigene offene Angebote */
  pactSent?: number[] | null;
  /** Geldgeschäfte, nur mit Remake-Regeln */
  finance?: FinanceView;
  /** in diesem Jahr noch baubare Schiffe */
  shipLimit?: number;
  taxes: TaxReport;
  unitCosts: UnitCosts;
  mercCosts: UnitCosts;
  war?: WarInfo;
}

/** Kredit zwischen zwei Herrschern (engine/finanzen.go); years = verbleibende Raten */
export interface PeerLoan { lender: number; borrower: number; amount: number; rate: number; years: number }

/** Finanzausschnitt der eigenen Sicht; Arrays je Sitz */
export interface FinanceView {
  bonitaet: number; bondPrice: number[]; bondDiv: number[]; bondFree: number[]; holdings: number[];
  issuedOwn: number; dividendPA: number; loans?: PeerLoan[] | null;
}

export interface UnitCosts { Cavalry: number; Artillery: number; Infantry: number }

export interface PublicPlayer {
  seat: number; name: string; country: number; female: boolean; title: number; alive: boolean;
  phase: Phase; land: number; treasury: number; population: number; soldiers: number; nobility: number;
  motto: string; avatar: string; arms: ArmsSpec; bot: string;
  markets: number; mills: number; palace: number; cathedral: number;
  colonies?: Colony[]; traders?: number; warships?: number;
}

/** Offene Beitrittsanfrage, nur für den Gastgeber sichtbar. */
export interface JoinRequest { id: string; name: string; country: number; female: boolean }

/** Eintrag der öffentlichen Spieleliste (GET /api/games). */
export interface GameListing {
  gameId: string; host: string; hostCountry: number; players: number; maxPlayers: number;
  started: boolean; year: number; mode: string; phaseSeconds: number; classic: boolean; joinable: boolean;
}

export interface LobbyPlayer { name: string; country: number; female: boolean; online: boolean; bot: string }

export interface GameEvent { type: string; player: number; year: number; data?: any }

export type ServerMsg =
  | { t: 'welcome'; gameId: string; token: string; seat: number; host: boolean }
  | { t: 'lobby' | 'preview'; gameId: string; started: boolean; joinable: boolean; settings: Settings; players: LobbyPlayer[] }
  | { t: 'pending'; gameId: string; token: string; host: string }
  | { t: 'requests'; gameId: string; requests: JoinRequest[] }
  | { t: 'public'; year: number; over: boolean; winner: number; full: boolean; players: PublicPlayer[]; countryArea?: number; pacts?: Pact[] | null }
  | { t: 'self'; self: SelfView }
  | { t: 'events' | 'history'; events: GameEvent[] }
  | { t: 'games'; games: GameStatus[] }
  | { t: 'error'; code: string; message: string };

/** Stand eines gemerkten Spiels für „Meine Spiele“ (Server: gameStatus). */
export interface GameStatus {
  gameId: string; found: boolean; valid: boolean; pending: boolean; started: boolean; over: boolean; year?: number;
  name?: string; country?: number; female?: boolean; title?: number; alive: boolean; host: boolean;
  myTurn: boolean; deadline: number; winner?: string; players: string[] | null;
}

export interface Action {
  type: string;
  qty?: number;
  customs?: number; vat?: number; incomeTax?: number; justice?: number;
  building?: string; unit?: string; mercenary?: boolean;
  target?: number; stance?: string; placements?: Placement[];
  rate?: number; years?: number; continent?: number; warships?: number;
  motto?: string; arms?: ArmsSpec;
}
