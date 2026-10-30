package engine

// Computer-Herrscher: entscheiden Schritt für Schritt, damit Zuschauer jeden Zug sehen.
// Jede Entscheidung ist eine normale Aktion über Apply – Bots haben keine Sonderrechte.

// BotStrategy beschreibt das Verhalten eines Computer-Herrschers.
type BotStrategy struct {
	Name      string
	FeedRatio float64 // Anteil am Kornbedarf, der verteilt wird
	Taxes     [4]int  // Zoll, MwSt, Einkommensteuer, Justiz
	Military  bool    // rüstet auf und führt alle vier Jahre Krieg
}

// BotStrategies sind die wählbaren Spielweisen.
var BotStrategies = map[string]BotStrategy{
	"kaufmann": {Name: "Kaufmann", FeedRatio: 1.25, Taxes: [4]int{20, 8, 8, 2}},
	// Feldherr: maßvolle Steuern und etwas mehr Korn, damit Volk und Kasse den Krieg tragen
	"feldherr": {Name: "Feldherr", FeedRatio: 1.2, Taxes: [4]int{24, 10, 8, 2}, Military: true},
}

// BotAction liefert die nächste Aktion eines Computer-Herrschers für seine aktuelle Phase.
// ok = false: nichts zu tun (Phase „done“).
func (g *Game) BotAction(seat int, strategy string) (Action, bool) {
	st, found := BotStrategies[strategy]
	if !found {
		st = BotStrategies["kaufmann"]
	}
	s := g.Seats[seat]
	p := s.Player
	switch s.Phase {
	case PhaseMarket:
		return g.botMarket(seat, st), true
	case PhaseFeed:
		lo, hi := FeedLimits(p)
		return Action{Type: "feed", Qty: max(lo, min(hi, feedTarget(p, st, g.skill(seat))))}, true
	case PhaseTaxes:
		t := st.Taxes
		// Unruhiges Volk: Steuern um 40 % senken und höchstens faire Justiz
		if p.MoodInit && p.Mood <= 3 {
			t = [4]int{t[0] * 6 / 10, t[1] * 6 / 10, t[2] * 6 / 10, min(t[3], 2)}
		}
		if p.Customs != t[0] || p.VAT != t[1] || p.IncomeTax != t[2] || p.Justice != t[3] {
			return Action{Type: "setTaxes", Customs: t[0], VAT: t[1], IncomeTax: t[2], Justice: t[3]}, true
		}
		// Unruhiges Volk beruhigen, solange die Kasse es erlaubt
		if p.MoodInit && p.Mood <= 3 && !p.Year.Opposition && p.Treasury > OppoCostTitle*p.Title*3 {
			return Action{Type: "opposition"}, true
		}
		if p.MoodInit && p.Mood <= 4 && p.Treasury > FestCostTitle*p.Title*6 {
			return Action{Type: "fest"}, true
		}
		// Gendarmen bei unruhigem Volk: bis zur halben Abdeckung
		if p.MoodInit && p.Mood <= 4 && coverage(p) < 0.5 && p.Treasury > 5000 {
			if q := min(p.Population/GendarmsPer/2-p.Gendarmes, p.Treasury/GendarmCost/4); q > 0 {
				return Action{Type: "gendarm", Qty: q}, true
			}
		}
		return Action{Type: "next"}, true
	case PhaseBuild:
		// Planung siehe botplan.go
		if a, ok := g.botBuild(seat, st); ok {
			return a, true
		}
		return Action{Type: "endTurn"}, true
	case PhaseDiplomacy:
		return Action{Type: "stance", Stance: g.botStance(seat, st)}, true
	case PhaseDeploy:
		return Action{Type: "autoDeploy"}, true
	}
	return Action{}, false
}

// power schätzt die Kampfkraft eines Herrschers (Soldaten × Moral).
func power(p *Player) float64 { return float64(p.Soldiers) * p.Morale }

// strongestRival liefert die Kampfkraft des stärksten lebenden Mitspielers.
func (g *Game) strongestRival(seat int) float64 {
	best := 0.0
	for i, s := range g.Seats {
		if i != seat && s.Player.Alive() {
			best = max(best, power(s.Player))
		}
	}
	return best
}

// botStance: Feldherren schließen sich dem Angreifer an; Kaufleute helfen dem schwächeren
// Verteidiger, wenn sie selbst Truppen haben, sonst bleiben sie neutral.
func (g *Game) botStance(seat int, st BotStrategy) string {
	w := g.current()
	if w == nil {
		return StanceNeutral
	}
	att, def := g.Seats[w.Attacker].Player, g.Seats[w.Defender].Player
	me := g.Seats[seat].Player
	switch {
	case st.Military && power(att) > power(def):
		return StanceHelpAttacker
	case !st.Military && me.Soldiers >= 80 && power(def) < power(att):
		return StanceHelpDefender
	case st.Military:
		return StancePassage
	}
	return StanceNeutral
}

// BotFallback beendet die Phase, falls die gewünschte Aktion abgelehnt wurde (z. B. kein Geld).
func (g *Game) BotFallback(seat int) Action {
	p := g.Seats[seat].Player
	switch g.Seats[seat].Phase {
	case PhaseFeed:
		lo, _ := FeedLimits(p)
		return Action{Type: "feed", Qty: lo}
	case PhaseBuild:
		return Action{Type: "endTurn"}
	case PhaseDiplomacy:
		return Action{Type: "stance", Stance: StanceNeutral}
	case PhaseDeploy:
		return Action{Type: "autoDeploy"}
	}
	return Action{Type: "next"}
}

// Namen für Computer-Herrscher (frei erfunden, zeitgemäß klingend).
var BotNames = []struct {
	Name   string
	Female bool
}{
	{"Leopold", false}, {"Augusta", true}, {"Friedrich", false}, {"Sophia", true},
	{"Maximilian", false}, {"Charlotte", true}, {"Ottokar", false}, {"Henriette", true},
	{"Wilhelm", false}, {"Amalia", true}, {"Ludwig", false}, {"Theresia", true},
}

// AddBot setzt einen Computer-Herrscher in das am wenigsten besetzte Land
// (bei Gleichstand das erste), damit sich Computer-Herrscher verteilen.
func (g *Game) AddBot(strategy string) (int, error) {
	if _, ok := BotStrategies[strategy]; !ok {
		return 0, &RuleError{"bad_strategy", "Unbekannte Spielweise."}
	}
	best := 0
	for c := 1; c <= 9; c++ {
		if n := g.CountryCount(c); n < g.PerCountry() && (best == 0 || n < g.CountryCount(best)) {
			best = c
		}
	}
	if best == 0 || len(g.Seats) >= MaxSeats {
		return 0, &RuleError{"room_full", "Der Reichstag ist voll."}
	}
	n := BotNames[len(g.Seats)%len(BotNames)]
	seat, err := g.AddPlayer(n.Name, best, n.Female)
	if err != nil {
		return 0, err
	}
	g.SetBot(seat, strategy)
	if g.CPU == nil {
		g.CPU = map[int]bool{}
	}
	g.CPU[seat] = true
	return seat, nil
}

// isCPU meldet einen Computer-Herrscher. Ältere Spielstände ohne CPU-Liste nehmen die Bots.
func (g *Game) isCPU(seat int) bool {
	if g.CPU != nil {
		return g.CPU[seat]
	}
	_, ok := g.Bots[seat]
	return ok
}

// SetBot schaltet den Autopiloten eines Sitzes ein (strategy) oder aus (leer).
func (g *Game) SetBot(seat int, strategy string) {
	if g.Bots == nil {
		g.Bots = map[int]string{}
	}
	if strategy == "" {
		delete(g.Bots, seat)
		return
	}
	g.Bots[seat] = strategy
}

// botColonies: Kaufleute bauen Handelsflotten und Kolonien auf, Feldherren halten Kriegsschiffe
// und überfallen schwach bewachte Kolonialflotten (Kaiser II). Jede Aktion ändert den Zustand,
// damit der Bot nicht in einer Schleife hängt.
func (g *Game) botColonies(seat int, st BotStrategy) (Action, bool) {
	p := g.Seats[seat].Player
	if g.Balance.Classic {
		return Action{}, false
	}
	ht, hw := shipsAtHome(p)
	// Freie Handelsschiffe zur Kolonie mit den wenigsten Händlern, freie Kriegsschiffe zur größten Flotte
	if ht > 0 {
		i := 0
		for j, c := range p.Colonies {
			if c.Traders < p.Colonies[i].Traders {
				i = j
			}
		}
		c := p.Colonies[i]
		return Action{Type: "fleet", Continent: i, Qty: c.Traders + ht, Warships: c.Warships}, true
	}
	if hw > 0 && !st.Military {
		i := 0
		for j, c := range p.Colonies {
			if c.Traders > p.Colonies[i].Traders {
				i = j
			}
		}
		if c := p.Colonies[i]; c.Traders > 0 {
			return Action{Type: "fleet", Continent: i, Qty: c.Traders, Warships: c.Warships + hw}, true
		}
	}
	// Kolonialland bis zur Deckung der Handelsschiffe
	for i, c := range p.Colonies {
		if need := (CoverPerTrader*c.Traders - c.Area) / ColonyParcel; need > 0 && p.Treasury > 8000 {
			if n := min(need, (p.Treasury-6000)/ColonyPrice); n > 0 {
				return Action{Type: "colonize", Continent: i, Qty: n}, true
			}
		}
	}
	if ShipLimit(p) > 0 {
		switch {
		case !st.Military && p.Title >= 2 && p.Treasury > 30000 && p.Traders < 6:
			return Action{Type: "buildShip", Unit: "trader", Qty: 1}, true
		case !st.Military && p.Traders >= 2 && p.Warships < p.Traders/2 && p.Treasury > 20000:
			return Action{Type: "buildShip", Unit: "warship", Qty: 1}, true
		case st.Military && p.Title >= 2 && p.Treasury > 14000 && p.Warships < 6:
			return Action{Type: "buildShip", Unit: "warship", Qty: 1}, true
		}
	}
	// Feldherren: Kaperfahrt gegen schwach bewachte Handelsflotten, jedes zweite Jahr
	if st.Military && hw >= 4 && !p.Year.NavalAttack && g.Year%2 == 0 {
		for other, o := range g.Seats {
			if other == seat || !o.Player.Alive() || g.warAllowed(seat, other) != nil {
				continue
			}
			for i, c := range o.Player.Colonies {
				if c.Traders > 0 && c.Warships*2 < hw {
					return Action{Type: "navalAttack", Target: other, Continent: i, Warships: hw}, true
				}
			}
		}
	}
	return Action{}, false
}
