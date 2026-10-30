package engine

import "math"

// Krieg: Kriegserklärung in der Bauphase, Austragung am Jahresende
// nacheinander. Je Krieg: Diplomatie der übrigen Herrscher, Weg, Aufstellung, Gefecht, Abrechnung.

const (
	PhaseDiplomacy Phase = "diplomacy" // Unbeteiligte wählen ihre Haltung
	PhaseDeploy    Phase = "deploy"    // Kriegsparteien stellen auf (nur taktischer Modus)
)

// Haltungen der Unbeteiligten. Codes wie im Original (H$): 0 neutral, 1 hilft Verteidiger,
// 2 Durchmarsch, 3 Durchmarsch + Hilfe für den Angreifer.
const (
	StanceNeutral      = "neutral"
	StanceHelpDefender = "helpDefender"
	StancePassage      = "passage"
	StanceHelpAttacker = "helpAttacker"
)

var stanceCode = map[string]int{StanceNeutral: 0, StanceHelpDefender: 1, StancePassage: 2, StanceHelpAttacker: 3}

// War ist ein erklärter Krieg dieses Jahres.
type War struct {
	Attacker int                    `json:"attacker"` // Sitz
	Defender int                    `json:"defender"` // Sitz
	Stances  map[int]string         `json:"stances"`
	Route    []int                  `json:"route"` // Länderfolge, leer = Weg versperrt
	Helpers  map[int]int            `json:"helpers"`
	Battle   *Battle                `json:"battle,omitempty"`
	Deploy   map[int][]Placement    `json:"deploy"`
	Result   *WarResult             `json:"result,omitempty"`
	Lent     map[int]map[string]int `json:"lent"` // Leihtruppen je Helfer (Kavallerie/Artillerie/Infanterie)
}

// WarResult ist die Abrechnung nach der Schlacht.
type WarResult struct {
	Blocked        bool          `json:"blocked"`
	Land           int           `json:"land"` // vom Angreifer gewonnen (negativ = verloren)
	Losses         [2]SideLosses `json:"losses"`
	TreasuryLoss   [2]int        `json:"treasuryLoss"`
	PopulationLoss [2]int        `json:"populationLoss"`
	Bonus          int           `json:"bonus"`        // Prämie, die der Angreifer zahlt
	Compensation   map[int]int   `json:"compensation"` // Entschädigung an Helfer (Sitz → Taler)
}

// ErrWar sind Regelverstöße bei der Kriegserklärung.
var (
	ErrTooEarly   = &RuleError{"too_early", "Es ist noch zu früh! Krieg erst ab dem Titel Baron."}
	ErrBadTarget  = &RuleError{"bad_target", "Das geht nicht!"}
	ErrAlreadyWar = &RuleError{"already_war", "Sie haben dieses Jahr bereits einen Krieg erklärt."}
	ErrBadStance  = &RuleError{"bad_stance", "Unbekannte Haltung."}
	ErrBadDeploy  = &RuleError{"bad_deploy", "Ungültige Aufstellung."}
)

// declareWar trägt einen Krieg ein; der Zug des Angreifers endet damit (wie im Original).
func (g *Game) declareWar(seat, target int) error {
	p := g.Seats[seat].Player
	if p.Title < 2 {
		return ErrTooEarly
	}
	if target < 0 || target >= len(g.Seats) || target == seat || !g.Seats[target].Player.Alive() {
		return ErrBadTarget
	}
	for _, w := range g.Wars {
		if w.Attacker == seat {
			return ErrAlreadyWar
		}
	}
	if err := g.warAllowed(seat, target); err != nil {
		return err
	}
	g.Wars = append(g.Wars, &War{Attacker: seat, Defender: target, Stances: map[int]string{}, Deploy: map[int][]Placement{}})
	return nil
}

// current liefert den gerade verhandelten Krieg oder nil.
func (g *Game) current() *War {
	if g.WarIdx < len(g.Wars) {
		return g.Wars[g.WarIdx]
	}
	return nil
}

// afterAllDone: Wenn alle fertig sind, folgen erst die Kriege, dann das Jahresende.
func (g *Game) afterAllDone(now int64) []Event {
	for g.WarIdx < len(g.Wars) {
		w := g.Wars[g.WarIdx]
		if w.Result != nil {
			g.WarIdx++
			continue
		}
		ev, waiting := g.advanceWar(w, now)
		if waiting {
			return ev
		}
		g.WarIdx++ // Ergebnis melden und gleich den nächsten Krieg angehen
		return append(ev, g.afterAllDone(now)...)
	}
	g.Wars, g.WarIdx = nil, 0
	return g.endYear(now)
}

// advanceWar führt einen Krieg so weit wie möglich; waiting = es wird auf Spieler gewartet.
func (g *Game) advanceWar(w *War, now int64) ([]Event, bool) {
	var ev []Event
	// 1. Diplomatie: alle lebenden Unbeteiligten, die noch nicht gewählt haben
	if w.Route == nil && !g.diplomacyDone(w) {
		if !g.anyInPhase(PhaseDiplomacy) {
			ev = append(ev, Event{"warDeclared", -1, g.Year, map[string]int{"attacker": w.Attacker, "defender": w.Defender}})
			for i, s := range g.Seats {
				if i != w.Attacker && i != w.Defender && s.Player.Alive() {
					// Verbündete des Angegriffenen stehen ihm ohne Rückfrage bei (Verteidigungsabkommen)
					if g.Allied(i, w.Defender) {
						w.Stances[i] = StanceHelpDefender
						continue
					}
					s.Phase, s.PhaseStart = PhaseDiplomacy, now
				}
			}
		}
		if !g.diplomacyDone(w) {
			return ev, true
		}
	}
	// 2. Weg und Helfer
	if w.Route == nil {
		w.Route = g.findRoute(w)
		if len(w.Route) == 0 {
			w.Result = &WarResult{Blocked: true}
			return append(ev, Event{"warBlocked", -1, g.Year, map[string]int{"attacker": w.Attacker, "defender": w.Defender}}), false
		}
		g.collectHelpers(w)
		att := g.Seats[w.Attacker].Player
		att.Morale *= 2 / float64(len(w.Route)) // langer Weg zehrt an der Moral
		w.Battle = &Battle{Field: NewField(g.Rng, att, g.Seats[w.Defender].Player)}
		if g.Settings.BattleMode == "tactical" {
			for _, s := range []int{w.Attacker, w.Defender} {
				g.Seats[s].Phase, g.Seats[s].PhaseStart = PhaseDeploy, now
			}
			ev = append(ev, Event{"deploy", -1, g.Year, map[string]int{"attacker": w.Attacker, "defender": w.Defender}})
			return ev, true
		}
	}
	// 3. Aufstellung abwarten (taktisch)
	if g.Seats[w.Attacker].Phase == PhaseDeploy || g.Seats[w.Defender].Phase == PhaseDeploy {
		return ev, true
	}
	// 4. Gefecht und Abrechnung
	g.fight(w)
	return append(ev, Event{"battle", -1, g.Year, map[string]any{
		"attacker": w.Attacker, "defender": w.Defender, "result": w.Result,
		"field": w.Battle.Field, "units": w.Battle.Start, "steps": w.Battle.Steps, // für die Wiedergabe im Client
	}}), false
}

func (g *Game) diplomacyDone(w *War) bool {
	for i, s := range g.Seats {
		if i == w.Attacker || i == w.Defender || !s.Player.Alive() {
			continue
		}
		if _, ok := w.Stances[i]; !ok {
			return false
		}
	}
	return true
}

func (g *Game) anyInPhase(ph Phase) bool {
	for _, s := range g.Seats {
		if s.Phase == ph {
			return true
		}
	}
	return false
}

// codesOf: Haltungscodes aller Herrscher eines Landes für die Wegsuche. Ein geteiltes Land
// ist nur passierbar, wenn alle dort zustimmen; liegt der Angreifer selbst darin, zieht er
// durch eigenes Gebiet (2). Unbesetzte Länder sind im Original gesperrt (0), im Remake frei (-1).
func (g *Game) codesOf(w *War, country int) []int {
	var codes []int
	for i, s := range g.Seats {
		if s.Player.Country != country {
			continue
		}
		switch {
		case i == w.Attacker:
			return []int{2}
		case i == w.Defender || !s.Player.Alive():
			codes = append(codes, 0)
		default:
			codes = append(codes, stanceCode[w.Stances[i]])
		}
	}
	if len(codes) > 0 {
		return codes
	}
	if g.Balance.Classic {
		return []int{0}
	}
	return []int{-1}
}

// routeFor sucht den ersten Weg von from nach to, dessen Transitländer ok(code) erfüllen.
func (g *Game) routeFor(w *War, from, to int, ok func(int) bool) []int {
	// Nachbarn im selben Land: direkter Weg ohne Transit
	if from == to {
		return []int{from}
	}
	for _, r := range g.World.Routes[from] {
		if r[len(r)-1] != to {
			continue
		}
		free := true
		for _, c := range r[1 : len(r)-1] {
			for _, code := range g.codesOf(w, c) {
				if !ok(code) {
					free = false
				}
			}
		}
		if free {
			return r
		}
	}
	return nil
}

func (g *Game) findRoute(w *War) []int {
	from, to := g.Seats[w.Attacker].Player.Country, g.Seats[w.Defender].Player.Country
	return g.routeFor(w, from, to, func(c int) bool { return c >= 2 || c == -1 })
}

// collectHelpers: Helfer brauchen einen eigenen freien Weg zum Angegriffenen (446–458)
// und leihen Kavallerie, Artillerie und Infanterie (keine Miliz).
func (g *Game) collectHelpers(w *War) {
	w.Helpers, w.Lent = map[int]int{}, map[int]map[string]int{}
	to := g.Seats[w.Defender].Player.Country
	for i, st := range w.Stances {
		side := -1
		var ok func(int) bool
		switch st {
		case StanceHelpAttacker:
			side, ok = 0, func(c int) bool { return c >= 2 || c == -1 }
		case StanceHelpDefender:
			side, ok = 1, func(c int) bool { return c < 2 }
		default:
			continue
		}
		h := g.Seats[i].Player
		if g.routeFor(w, h.Country, to, ok) == nil {
			continue
		}
		w.Helpers[i] = side
		w.Lent[i] = map[string]int{"cavalry": h.Cavalry, "artillery": h.Artillery, "infantry": h.Infantry}
	}
}

// sideCounts: Einheiten einer Kriegspartei inklusive Leihtruppen.
func (g *Game) sideCounts(w *War, side int) map[string]int {
	seat := w.Attacker
	if side == 1 {
		seat = w.Defender
	}
	p := g.Seats[seat].Player
	c := map[string]int{"cavalry": p.Cavalry, "artillery": p.Artillery, "infantry": p.Infantry, "militia": p.Militia}
	for h, s := range w.Helpers {
		if s == side {
			for k, v := range w.Lent[h] {
				c[k] += v
			}
		}
	}
	return c
}

// SetDeployment übernimmt eine taktische Aufstellung; fehlende Einheiten stellt der Feldherr auf.
func (g *Game) setDeployment(seat int, pl []Placement) error {
	w := g.current()
	if w == nil || w.Battle == nil || (seat != w.Attacker && seat != w.Defender) {
		return ErrWrongPhase
	}
	side := 0
	if seat == w.Defender {
		side = 1
	}
	left := g.sideCounts(w, side)
	taken := map[[2]int]bool{}
	var ok []Placement
	for _, p := range pl {
		if left[p.Kind] <= 0 || taken[[2]int{p.Row, p.Col}] || !w.Battle.Field.validPlacement(side, p.Row, p.Col) {
			return ErrBadDeploy
		}
		left[p.Kind]--
		taken[[2]int{p.Row, p.Col}] = true
		ok = append(ok, p)
	}
	w.Deploy[side] = append(ok, AutoPlace(g.Rng, &w.Battle.Field, side, left, taken)...)
	return nil
}

// fight stellt fehlende Aufstellungen automatisch, kämpft und rechnet ab.
func (g *Game) fight(w *War) {
	b := w.Battle
	for side := 0; side < 2; side++ {
		if _, done := w.Deploy[side]; !done {
			w.Deploy[side] = AutoPlace(g.Rng, &b.Field, side, g.sideCounts(w, side), map[[2]int]bool{})
		}
	}
	id := 1 // IDs ab 1, damit Ziel 0 „kein Ziel“ bedeutet
	for side := 0; side < 2; side++ {
		for _, p := range w.Deploy[side] {
			b.Units = append(b.Units, &BattleUnit{ID: id, Side: side, Kind: p.Kind, Row: p.Row, Col: p.Col})
			id++
		}
	}
	// Startaufstellung sichern: Fight bewegt die Einheiten, die Wiedergabe braucht den Anfang.
	for _, u := range b.Units {
		b.Start = append(b.Start, *u)
	}
	att, def := g.Seats[w.Attacker].Player, g.Seats[w.Defender].Player
	// Ausbildung und Material (Kaiser II) verstärken die Moral als Kampfkraft-Faktor
	b.Fight(g.Rng, [2]float64{att.Morale * Strength(att), def.Morale * Strength(def)}, [2]int{def.Land / 800, att.Land / 800})
	w.Result = g.settle(w)
}

// settle verrechnet Land, Verluste, Kasse, Einwohner, Prämie und Leihtruppen (304–352).
func (g *Game) settle(w *War) *WarResult {
	b := w.Battle
	att, def := g.Seats[w.Attacker].Player, g.Seats[w.Defender].Player
	r := &WarResult{Losses: b.Losses, Compensation: map[int]int{}}
	land := max(1-att.Land, min(def.Land-1, b.Land))
	r.Land = land

	// Verluste an Einheiten: zuerst die eigenen, Leihtruppen werden danach zurückgegeben.
	for side, p := range []*Player{att, def} {
		l := b.Losses[side]
		lostOwn := func(have *int, lost int) int {
			n := min(*have, lost)
			*have -= n
			return lost - n // Rest trifft Leihtruppen
		}
		rest := map[string]int{
			"cavalry":   lostOwn(&p.Cavalry, l.Cavalry),
			"artillery": lostOwn(&p.Artillery, l.Artillery),
			"infantry":  lostOwn(&p.Infantry, l.Infantry),
		}
		price := map[string]int{"cavalry": 2000, "artillery": 1300, "infantry": 900}
		for h, s := range w.Helpers {
			if s != side {
				continue
			}
			hp := g.Seats[h].Player
			for k, v := range rest {
				n := min(v, w.Lent[h][k])
				if n == 0 {
					continue
				}
				rest[k] -= n
				switch k {
				case "cavalry":
					hp.Cavalry -= n
				case "artillery":
					hp.Artillery -= n
				case "infantry":
					hp.Infantry -= n
				}
				pay := n * price[k]
				p.Treasury -= pay
				hp.Treasury += pay
				r.Compensation[h] += pay
			}
			hp.RecalcArmy()
		}
		p.Markets = max(0, p.Markets-l.Markets)
		p.Mills = max(0, p.Mills-l.Mills)
		p.Palace = max(0, p.Palace-int(l.Palace))
		p.Cathedral = max(0, p.Cathedral-int(l.Cathedral))
	}

	// Verlierer zahlt aus der Kasse und verliert Einwohner im Verhältnis zum verlorenen Land.
	loser, lost := def, land
	li := 1
	if land < 0 {
		loser, lost, li = att, -land, 0
	}
	if lost > 0 && loser.Land > 0 {
		if loser.Treasury > 0 {
			r.TreasuryLoss[li] = int(float64(loser.Treasury) / float64(loser.Land) * float64(lost))
			loser.Treasury -= r.TreasuryLoss[li]
		}
		r.PopulationLoss[li] = int(float64(loser.Population) / 2 / float64(loser.Land) * float64(lost))
		loser.Population -= r.PopulationLoss[li]
	}
	att.Land += land
	def.Land -= land
	if land > 0 { // Siegprämie für die Überlebenden (314)
		r.Bonus = att.Soldiers * int(math.Log10(float64(land)+1)) * 2
		att.Treasury -= r.Bonus
	}
	att.RecalcArmy()
	def.RecalcArmy()
	return r
}
