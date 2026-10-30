// Ratgeber („Der Kanzler rät“): leitet aus dem eigenen Stand ab, was jetzt zu tun ist.
// Reine Funktion ohne Zustand; Schwellen spiegeln die Regeln in engine/yearend.go und build.go.
import type { SelfView, PublicPlayer } from './types';
import { num, title, CONTINENTS, COVER_PER_TRADER, COLONY_PARCEL } from './format';

export interface Tip {
  /** muss = sonst droht Schaden, rat = lohnt sich, info = was die Phase verlangt */
  level: 'muss' | 'rat' | 'info';
  text: string;
}

// Was jede Phase vom Herrscher will (erste Zeile des Ratgebers)
const PHASE_TODO: Record<string, string> = {
  market: 'Korn und Land kaufen oder verkaufen, dann weiter zur Versorgung.',
  feed: 'Legen Sie fest, wie viel Korn das Volk bekommt.',
  taxes: 'Steuersätze und Justiz einstellen und die Steuern eintreiben.',
  build: 'Bauen, Truppen anwerben, gegebenenfalls Krieg erklären und dann den Zug beenden.',
  diplomacy: 'Entscheiden Sie, wie Sie sich zum Krieg Ihrer Nachbarn stellen.',
  deploy: 'Stellen Sie Ihre Truppen auf dem Schlachtfeld auf.',
  done: 'Ihr Jahr ist abgeschlossen. Warten Sie auf die übrigen Herrscher.',
};

/** Bestandteile der Titelwertung (engine TitleScore), je höchstens 17 Punkte. */
function titleParts(p: SelfView['player']) {
  return [
    { label: 'Märkte', v: p.markets }, { label: 'Paläste', v: p.palace }, { label: 'Kathedrale', v: p.cathedral },
    { label: 'Mühlen', v: p.mills }, { label: 'Staatskasse', v: p.treasury / 5000 }, { label: 'Land', v: p.land / 6000 },
    { label: 'Bürger', v: p.burghers / 50 }, { label: 'Adel', v: p.nobility / 5 }, { label: 'Soldaten', v: p.soldiers / 50 },
    { label: 'Klerus', v: p.clergy / 10 }, { label: 'Einwohner', v: p.population / 2000 }, { label: 'Handel', v: p.trade / 5 },
  ].map((x) => ({ ...x, pts: Math.max(0, Math.min(17, Math.floor(x.v))) }));
}

/** Ratschläge für die aktuelle Phase, wichtigste zuerst. */
export function advise(self: SelfView, year: number, players: PublicPlayer[] = []): Tip[] {
  const p = self.player;
  const tips: Tip[] = [];
  const todo = PHASE_TODO[self.phase];
  if (todo) tips.push({ level: 'info', text: todo });

  if (p.suspendedUntil > year) tips.push({ level: 'muss', text: `Sie sind bis ${p.suspendedUntil} des Amtes enthoben. Kaufen Sie Land, damit es nicht wieder passiert.` });

  // Offene Bündnisangebote (Außenpolitik, Bauphase)
  if ((self.pactOffers ?? []).length && self.phase === 'build') tips.push({ level: 'rat', text: 'Ein Herrscher bietet Ihnen ein Verteidigungsabkommen an. Antworten Sie unter „Außenpolitik“.' });

  // Bankkredit blockiert den Aufstieg (Kaiser II)
  if ((p.loan ?? 0) > 0 && (self.phase === 'market' || self.phase === 'build') && p.treasury > (p.loan ?? 0))
    tips.push({ level: 'rat', text: 'Ihr Bankkredit verhindert den nächsten Titel. Tilgen Sie ihn unter „Finanzen“, die Kasse reicht.' });

  // Flotte und Kolonien (Kaiser II, nur Remake-Regeln): ungenutzte Schiffe, ungenutzte Deckung, Bedrohung
  if (self.finance && self.phase === 'build') {
    const cols = p.colonies ?? [];
    const homeT = (p.traders ?? 0) - cols.reduce((s, c) => s + c.traders, 0);
    if (homeT > 0) tips.push({ level: 'rat', text: `${homeT} Handelsschiff${homeT === 1 ? ' liegt' : 'e liegen'} ungenutzt im Hafen. Teilen Sie sie unter „Kolonien“ einer Kolonialflotte zu.` });
    cols.forEach((c, i) => {
      const frei = Math.floor((COVER_PER_TRADER * c.traders - c.area) / COLONY_PARCEL);
      if (frei > 0) tips.push({ level: 'rat', text: `Ihre Händler in ${CONTINENTS[i]} könnten ${num(frei * COLONY_PARCEL)} km² mehr bedienen. Kolonialland kaufen lohnt sich.` });
    });
    // Stärkster fremder Kriegshafen gegen die schwächste eigene Handelsflotte
    const feind = Math.max(0, ...players.filter((x) => x.seat !== self.seat && x.alive).map((x) => x.warships ?? 0));
    const schwach = cols.findIndex((c) => c.traders > 0 && c.warships * 2 < feind);
    if (schwach >= 0) tips.push({ level: 'rat', text: `Ein Nachbar hat ${feind} Kriegsschiffe, Ihre ${CONTINENTS[schwach]}-Flotte ist schwach bewacht.` });
    if (!(p.traders ?? 0) && p.title >= 2 && p.treasury > 30000) tips.push({ level: 'info', text: 'Mit Handelsschiffen und Kolonien (unter „Kolonien“) erschließen Sie Gold, Gewürze und Rohstoffe.' });
  }
  // Überschüsse bringen in Reichsanleihen mehr als 3 % Guthabenzins
  if (self.finance && (self.phase === 'market' || self.phase === 'build') && p.treasury > 60000 && !(p.reichBonds ?? 0))
    tips.push({ level: 'info', text: 'Ihre Kasse ist prall. Reichsanleihen (unter „Finanzen“) bringen gut 7 % statt 3 % Guthabenzins.' });

  // Volksstimmung (Kaiser II): ab „schlecht“ warnen, bei 0–1 drohen Aufstände
  if (p.moodInit && p.mood <= 1) tips.push({ level: 'muss', text: `Das Volk ist ${p.mood === 0 ? 'aufständisch' : 'ganz mies gestimmt'}. Es droht eine Revolution! Volksfest, Opposition verhaften, Gendarmen und niedrigere Steuern helfen (Steuerphase).` });
  else if (p.moodInit && p.mood <= 3) tips.push({ level: 'rat', text: 'Die Stimmung im Volk ist schlecht. Mehr Korn, niedrigere Steuern oder ein Volksfest heben sie wieder.' });

  // Landmangel: weniger Hektar als Einwohner führt nach den Steuern zur Amtsenthebung
  const landGap = p.population - p.land;
  if (['market', 'feed', 'taxes'].includes(self.phase)) {
    if (landGap > 0) tips.push({ level: 'muss', text: `Zu wenig Land: ${num(p.population)} Einwohner auf ${num(p.land)} ha. Kaufen Sie mindestens ${num(landGap)} ha, sonst droht die Amtsenthebung.` });
    else if (p.land < p.population * 1.15) tips.push({ level: 'rat', text: `Das Land wird knapp (${num(p.land)} ha für ${num(p.population)} Einwohner). Bald etwas Land dazukaufen.` });
  }

  if (self.phase === 'market') {
    const need = p.year.grainNeed;
    if (p.grain < need) tips.push({ level: 'muss', text: `Die Reserve reicht nicht für das Volk: Es fehlen ${num(need - p.grain)} Maß Korn. Kaufen Sie nach.` });
    // Verteilt werden dürfen nur 80 % der Reserve: für Einwanderer (etwa 150 % des Bedarfs) rechtzeitig zukaufen
    else if (p.grain * 0.8 < need * 1.5) tips.push({ level: 'rat', text: `Für Einwanderer sollten Sie etwa 150 % des Bedarfs verteilen. Da nur 80 % der Reserve verteilt werden dürfen, fehlen dafür ${num(Math.ceil(need * 1.5 / 0.8 - p.grain))} Maß – jetzt zukaufen.` });
    else if (p.grain > need * 3) tips.push({ level: 'rat', text: `Sie lagern ${num(p.grain)} Maß, das Dreifache des Bedarfs. Ein Teil verdirbt jedes Jahr – Überschuss lässt sich verkaufen.` });
  }

  if (self.phase === 'feed') {
    tips.push({ level: 'rat', text: `Mindestens ${num(self.feedMin)} Maß verteilen, sonst verhungern Menschen. Bis ${num(self.feedMax)} Maß lockt Einwanderer an.` });
  }

  // Bankrott droht unter etwa −8 000 Taler je Titelstufe (−10 000 × Titel × 0,8)
  const bankruptAt = -8000 * p.title;
  if (p.treasury < 0) {
    tips.push({
      level: p.treasury < bankruptAt * 0.6 ? 'muss' : 'rat',
      text: p.treasury < bankruptAt * 0.6
        ? `Die Kasse steht bei ${num(p.treasury)} Talern. Ab etwa ${num(bankruptAt)} droht der Bankrott – Steuern erhöhen und Ausgaben stoppen.`
        : `Die Kasse ist im Minus (${num(p.treasury)} Taler); im Minus gibt es keinen neuen Titel.`,
    });
  }

  if (self.phase === 'taxes' && p.treasury >= 0 && p.customs + p.vat + p.incomeTax > 60) {
    tips.push({ level: 'rat', text: 'Hohe Steuern vertreiben Bürger und Händler. Die Vorschau zeigt, was die Sätze einbringen.' });
  }

  if (self.phase === 'build') {
    const plots = Math.floor(p.land / 1000) - 1;
    if (p.markets <= plots && p.treasury >= 1000 && p.markets < 10) tips.push({ level: 'rat', text: 'Märkte (1.000 Taler) bringen jedes Jahr Einnahmen und Handel – ein guter Anfang.' });
    if (p.mills <= plots && p.treasury >= 2000 && p.mills < p.markets / 2) tips.push({ level: 'rat', text: 'Mühlen (2.000 Taler) senken den Kornverderb und zählen für den Titel.' });
    if (p.soldiers < p.population / 100) tips.push({ level: 'rat', text: 'Ihr Heer ist sehr klein. Ein paar Soldaten schrecken Angreifer ab.' });
    if (p.title === 7 && p.palace < 16) tips.push({ level: 'muss', text: `Für den Königstitel braucht es einen vollendeten Palast (16 Abschnitte, Sie haben ${p.palace}).` });
    if (p.title === 8) tips.push({ level: 'muss', text: 'Kaiser wird nur, wer 14 Kathedralen-Abschnitte, 15 Mühlen, 25 Märkte und 100.000 Taler besitzt.' });

    // Titelwertung: die schwächsten Bereiche nennen, in denen noch Punkte zu holen sind
    if (p.title < 9) {
      // Palast erst ab 13.000 ha, Kathedrale ab 25.000 ha baubar (build.go) – vorher nicht empfehlen
      const blocked = new Set([...(p.land < 13000 ? ['Paläste'] : []), ...(p.land < 25000 ? ['Kathedrale'] : [])]);
      const weak = titleParts(p).filter((x) => x.pts < 17 && !blocked.has(x.label)).sort((a, b) => a.pts - b.pts).slice(0, 3).map((x) => x.label);
      tips.push({ level: 'info', text: `Für den Titel ${title(p.title + 1, p.female)} zählt der Durchschnitt aller Bereiche. Am meisten fehlt bei: ${weak.join(', ')}.` });
    }
  }

  // muss vor rat vor info, Phasenaufgabe bleibt bei info vorne
  const order = { muss: 0, rat: 1, info: 2 };
  return tips.sort((a, b) => order[a.level] - order[b.level]);
}
