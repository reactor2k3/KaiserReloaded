// Texte und Zahlenformate für die Oberfläche. Namen und Titel spiegeln game/data/world.json.
import type { GameEvent, PublicPlayer } from './types';

export const COUNTRIES = ['', 'Preußen', 'Hessen', 'Bayern', 'Böhmen', 'Sachsen', 'Mähren', 'Tirol', 'der Pfalz', 'Flandern'];
const TITLES_M = ['', 'Herr', 'Baron', 'Landgraf', 'Markgraf', 'Fürst', 'Herzog', 'Kurfürst', 'König', 'Kaiser'];
const TITLES_F = ['', 'Frau', 'Baronin', 'Landgräfin', 'Markgräfin', 'Fürstin', 'Herzogin', 'Kurfürstin', 'Königin', 'Kaiserin'];
export const WEATHER = ['', 'Dürre – Hungersnot droht', 'Regen – schlechte Ernte', 'Gewöhnliche Ernte', 'Gutes Wetter – reiche Ernte', 'Tolles Wetter – Rekordernte'];
export const JUSTICE = ['', 'Sehr fair', 'Bescheiden', 'Hart', 'Gierig'];
export const PHASES: { id: string; label: string }[] = [
  { id: 'market', label: 'Markt' },
  { id: 'feed', label: 'Volk' },
  { id: 'taxes', label: 'Steuern' },
  { id: 'build', label: 'Bauen' },
  { id: 'diplomacy', label: 'Diplomatie' },
  { id: 'deploy', label: 'Aufstellung' },
  { id: 'done', label: 'Fertig' },
];
export const UNIT_NAMES: Record<string, string> = { cavalry: 'Kavallerie', artillery: 'Artillerie', infantry: 'Infanterie', militia: 'Miliz' };

/** Farbe je Land für Karte und Spielerplaketten (gedeckte Heraldik-Töne). */
export const COUNTRY_COLOR = ['', '#8e8e86', '#c27e72', '#8faccc', '#b0707f', '#9db57f', '#c9a96a', '#b86a6a', '#7fa3a0', '#d4b85e'];

/** Farbe eines Herrschers: Landesfarbe, in geteilten Ländern heller bzw. dunkler abgestuft (nach Sitz). */
export function playerColor(players: { seat: number; country: number }[], seat: number): string {
  const p = players.find((x) => x.seat === seat);
  if (!p) return '#a8987c';
  const idx = players.filter((x) => x.country === p.country && x.seat < seat).length;
  const base = COUNTRY_COLOR[p.country];
  return idx === 0 ? base : idx === 1 ? mix(base, '#ffffff', 0.4) : mix(base, '#2b2118', 0.3);
}

import { mix } from './teilung';

const nf = new Intl.NumberFormat('de-DE');
/** Kontinente der Kolonien (Index wie engine/kolonien.go) und ihre Waren */
export const CONTINENTS = ['Afrika', 'Amerika', 'Asien'];
export const CONTINENT_GOODS = ['Rohstoffe', 'Gold', 'Gewürze'];
export const TRADER_COST = 8000, WARSHIP_COST = 4000, COLONY_PRICE = 1200, COLONY_PARCEL = 1000, COVER_PER_TRADER = 4000;

/** Sterblichkeit in der Lobby: Anzeige und Wirkung wie engine/settings.go */
export const MORTALITY: { value: 'unsterblich' | 'niedrig' | 'normal' | 'hoch'; label: string; info: string }[] = [
  { value: 'unsterblich', label: 'Unsterblich', info: 'kein natürlicher Tod' },
  { value: 'niedrig', label: 'Niedrig', info: '+15 Jahre, 25 %' },
  { value: 'normal', label: 'Normal', info: 'Original, 50 %' },
  { value: 'hoch', label: 'Hoch', info: '−10 Jahre, 75 %' },
];

/** Zugzeiten je Phase (Sekunden) wie engine/settings.go: live und asynchron */
export const LIVE_SECONDS = [0, 60, 90, 120, 180, 300];
export const ASYNC_SECONDS = [3600, 21600, 43200, 86400, 259200];

/** Lesbare Zugzeit: Sekunden, Stunden oder Tage */
export function zugzeit(s: number): string {
  if (!s) return 'unbegrenzt';
  if (s < 3600) return `${s} Sekunden`;
  if (s < 86400) return `${s / 3600} Stunde${s === 3600 ? '' : 'n'}`;
  return `${s / 86400} Tag${s === 86400 ? '' : 'e'}`;
}

/** Restzeit für Uhr und „Meine Spiele“: m:ss, ab einer Stunde „5 h 07 min“, ab einem Tag „2 T 3 h“ */
export function restzeit(s: number): string {
  s = Math.max(0, Math.floor(s));
  if (s < 3600) return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
  if (s < 86400) return `${Math.floor(s / 3600)} h ${String(Math.floor((s % 3600) / 60)).padStart(2, '0')} min`;
  return `${Math.floor(s / 86400)} T ${Math.floor((s % 86400) / 3600)} h`;
}

/** Spielstärke der Computer-Herrscher (engine/botplan.go botSkills) */
export const DIFFICULTY: { value: 'leicht' | 'mittel' | 'schwer'; label: string }[] = [
  { value: 'leicht', label: 'Leicht' },
  { value: 'mittel', label: 'Mittel' },
  { value: 'schwer', label: 'Schwer' },
];

export const num = (n: number) => nf.format(Math.round(n));

export function title(t: number, female: boolean) {
  return (female ? TITLES_F : TITLES_M)[t] ?? '';
}

export function ruler(p: { title: number; female: boolean; name: string; country: number }) {
  return `${title(p.title, p.female)} ${p.name} von ${COUNTRIES[p.country]}`;
}

const name = (p?: PublicPlayer) => (p ? `${title(p.title, p.female)} ${p.name}` : 'Unbekannt');

/** Chroniktext zu einem Ereignis; leer = nicht in die Chronik. */
/** Bezeichnungen der Volksstimmung 0–10 wie in Kaiser II (engine MoodNames). */
export const MOOD_NAMES = ['aufständisch', 'ganz mies', 'unzufrieden', 'schlecht', 'unruhig', 'normal',
  'ruhig', 'zufrieden', 'gut', 'optimistisch', 'euphorisch'];

/** Kosten der Innenpolitik (engine/stimmung.go) */
export const GENDARM_COST = 100, GENDARM_PER = 40, FEST_COST = 800, OPPO_COST = 2500;

/** Folgen eines Ereignisses als kurze Liste („+5.000 Taler“, „−300 ha Land“ …). */
export function fazit(r: { taler?: number; land?: number; soldiers?: number; grain?: number; people?: number; years?: number; ships?: number }): string[] {
  const sig = (n: number) => (n > 0 ? '+' : '−') + num(Math.abs(n));
  const out: string[] = [];
  if (r.taler) out.push(`${sig(r.taler)} Taler`);
  if (r.land) out.push(`${sig(r.land)} ha Land`);
  if (r.soldiers) out.push(`${sig(r.soldiers)} Soldaten`);
  if (r.grain) out.push(`${sig(r.grain)} Maß Korn`);
  if (r.people) out.push(`${sig(r.people)} Einwohner`);
  if (r.years) out.push(`${sig(r.years)} Lebensjahre`);
  if (r.ships) out.push(`${sig(r.ships)} Schiffe`);
  return out;
}

export const UNREST_TITLE: Record<string, string> = { revolution: 'Revolution', putsch: 'Militärputsch', attentat: 'Attentat' };
export const UNREST_RESULT: Record<string, string> = {
  niedergeschlagen: 'niedergeschlagen', exil: 'Sie gehen ins Exil', pluenderung: 'Plünderungen im ganzen Land',
  vereitelt: 'vereitelt', verletzt: 'schwer verletzt',
};

export function describeEvent(e: GameEvent, players: PublicPlayer[], me: number): string {
  const p = players[e.player];
  const who = e.player === me ? 'Sie' : p ? `${title(p.title, p.female)} ${p.name}` : '';
  const d = e.data ?? {};
  switch (e.type) {
    case 'newYear': return `Ein neues Jahr beginnt: ${e.year}.`;
    case 'ereignis': return e.player === me ? `${d.title}: ${fazit(d).join(', ') || 'ohne Folgen'}.` : `${who}: ${d.title}.`;
    case 'risiko': return e.player === me ? `Eine Entscheidung steht an: ${d.title}.` : '';
    case 'stimmung': {
      const unruhe = d.unrest ? ` – ${UNREST_TITLE[d.unrest] ?? d.unrest}: ${UNREST_RESULT[d.result] ?? d.result}` : '';
      if (e.player === me) return `Stimmung im Volk: ${d.after} (${d.name})${unruhe}.`;
      return d.unrest ? `${who}${unruhe}.` : '';
    }
    case 'finanz': {
      const t = players[d.target]?.name ?? '?';
      const betrag = `${num(d.amount)} Taler`;
      if (d.kind === 'gift') return e.player === me ? `Sie schenken ${t} ${betrag}.` : d.target === me ? `${who} schenkt Ihnen ${betrag}.` : `${who} schenkt ${t} ${betrag}.`;
      if (d.kind === 'lend') return e.player === me ? `Sie leihen ${t} ${betrag} zu ${d.rate} % auf ${d.years} Jahre.`
        : d.target === me ? `${who} leiht Ihnen ${betrag} zu ${d.rate} % auf ${d.years} Jahre.` : '';
      if (d.kind === 'repay') return e.player === me ? `Sie zahlen ${t} ${betrag} Schulden zurück.` : d.target === me ? `${who} zahlt Ihnen ${betrag} Schulden zurück.` : '';
      return '';
    }
    case 'kolonien': {
      if (e.player !== me) return '';
      const ein = (d.income as number[]).reduce((s, x) => s + x, 0);
      return `Kolonien: Erträge +${num(ein)} Taler (${CONTINENT_GOODS.map((g, i) => `${g} ${num(d.income[i])}`).join(', ')}), Flottenunterhalt −${num(d.upkeep)} Taler.`;
    }
    case 'seekrieg': {
      const att = players[d.attacker]?.name ?? '?', def = players[d.defender]?.name ?? '?';
      const wo = CONTINENTS[d.continent];
      const verlauf = `${d.attLost} gegen ${d.defLost} versenkte Kriegsschiffe`;
      if (d.attacker === me) return d.won ? `Ihre Marine schlägt die ${wo}-Flotte von ${def} in die Flucht (${verlauf}), ${d.tradersHit} Handelsschiffe versenkt.`
        : `Ihre Marine scheitert an der ${wo}-Flotte von ${def} (${verlauf}).`;
      if (d.defender === me) return d.won ? `${att} überfällt Ihre ${wo}-Flotte (${verlauf}). ${d.tradersHit} Handelsschiffe verloren, die Flotte segelt heim.`
        : `Ihre ${wo}-Flotte wehrt einen Angriff von ${att} ab (${verlauf}).`;
      return `Seeschlacht vor ${wo}: ${att} gegen ${def}, ${d.won ? 'der Angreifer siegt' : 'der Angriff scheitert'}.`;
    }
    case 'anleihen': return e.player === me ? `${num(d.paid)} Taler gehen an die Besitzer Ihrer Anleihen.` : '';
    case 'finanzen': {
      if (e.player !== me) return '';
      const teile = [
        d.loanInterest && `Kreditzins −${num(d.loanInterest)}`,
        d.reichYield && `Reichsanleihen +${num(d.reichYield)}`,
        d.dividends && `Anleihen +${num(d.dividends)}`,
        d.peerIn && `von Schuldnern +${num(d.peerIn)}`,
        d.peerOut && `an Gläubiger −${num(d.peerOut)}`,
      ].filter(Boolean);
      return teile.length ? `Geldgeschäfte: ${teile.join(', ')} Taler.` : '';
    }
    case 'pactOffer': return e.player === me ? `Sie bieten ${players[d.target]?.name ?? '?'} ein Verteidigungsabkommen an.`
      : d.target === me ? `${who} bietet Ihnen ein Verteidigungsabkommen an.` : '';
    case 'pactAnswer': {
      const from = players[d.from];
      if (d.from === me) return `${who} ${d.accept ? 'nimmt Ihr Abkommen an. Sie sind nun verbündet' : 'lehnt Ihr Abkommen ab'}.`;
      if (e.player === me) return `Sie ${d.accept ? 'sind nun verbündet mit' : 'lehnen das Abkommen ab von'} ${from?.name ?? '?'}.`;
      return d.accept ? `${who} und ${from?.name ?? '?'} schließen ein Verteidigungsabkommen.` : '';
    }
    case 'exil': return `${who} ${e.player === me ? 'müssen' : 'muss'} ins Exil gehen. Das Spiel ist für ${e.player === me ? 'Sie' : 'diesen Herrscher'} zu Ende.`;
    case 'harvest': return e.player === me ? `${WEATHER[d.weather]}. ${num(d.rotPercent)} % der Reserven sind verfault.` : '';
    case 'fed':
      if (e.player !== me) return '';
      return `${num(d.births)} Geburten, ${num(d.deaths)} Todesfälle` +
        (d.immigrants ? `, ${num(d.immigrants)} Einwanderer` : '') +
        (d.emigrants ? `, ${num(d.emigrants)} verließen das Land` : '') + '.';
    case 'income': return e.player === me ? `Märkte ${num(d.markets)}, Mühlen ${num(d.mills)}, Sold −${num(d.pay)} Taler.` : '';
    case 'taxes': return e.player === me ? `Staatseinnahmen: ${num(d.total)} Taler.` : '';
    case 'checks':
      if (d.bankrupt) return `${who} ${e.player === me ? 'sind' : 'ist'} bankrott – Gläubiger pfänden große Teile des Besitzes!`;
      if (d.suspended) return `${who} wegen schlechter Landpolitik ein Jahr des Amtes enthoben.`;
      if (d.lostMarkets || d.lostMills || d.lostPalace || d.lostCathedral) return `${who}: Gebäude wegen Landmangel verloren.`;
      return '';
    case 'yearEnd':
      if (d.emperor) return `${who} wurde zum Kaiser gekrönt!`;
      if (d.promoted) return `${who} ${e.player === me ? 'tragen' : 'trägt'} nun den Titel ${p ? title(d.newTitle, p.female) : ''}.`;
      if (d.timePenalty && e.player === me) return `Zu lange gezögert: Ihre Lebenserwartung sinkt um ${d.timePenalty} Jahr(e).`;
      return '';
    case 'died': return `Sehr schlechte Nachricht: ${who} ${e.player === me ? 'sind' : 'ist'} verstorben.`;
    case 'timeout': return e.player === me ? 'Die Zeit ist abgelaufen – der Zug wurde automatisch fortgesetzt.' : '';
    case 'suspended': return e.player === me ? 'Sie sind dieses Jahr Ihres Amtes enthoben.' : '';
    case 'gameOver': return 'Das Spiel ist zu Ende.';
    case 'warPlanned': return e.player === me ? `Sie rüsten zum Krieg gegen ${name(players[d.target])}.` : '';
    case 'warDeclared': return `${name(players[d.attacker])} erklärt ${name(players[d.defender])} den Krieg!`;
    case 'warBlocked': return `Alle Wege von ${name(players[d.attacker])} nach ${COUNTRIES[players[d.defender]?.country ?? 0]} sind versperrt.`;
    case 'deploy': return `Die Heere von ${name(players[d.attacker])} und ${name(players[d.defender])} stellen sich auf.`;
    case 'battle': {
      const r = d.result as { land: number };
      const a = name(players[d.attacker]), v = name(players[d.defender]);
      if (r.land > 0) return `Schlacht: ${a} erobert ${num(r.land)} Hektar von ${v}.`;
      if (r.land < 0) return `Schlacht: ${v} schlägt ${a} zurück und gewinnt ${num(-r.land)} Hektar.`;
      return `Schlacht zwischen ${a} und ${v} ohne Landgewinn.`;
    }
  }
  return '';
}
