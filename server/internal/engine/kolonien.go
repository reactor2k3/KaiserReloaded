package engine

import "math"

// Flotten und Kolonien nach dem Vorbild Kaiser II (Werft, Kolonialflotten, Kriegsflotte entsenden,
// Seeschlacht, Kolonialerträge).
//
// Original:
//   - Handelsschiff 8000, Kriegsschiff 4000 Taler, je Jahr höchstens Einwohner / 200 Schiffe,
//     Unterhalt 400 bzw. 200 Taler im Jahr.
//   - Je Kontinent (Afrika, Amerika, Asien) eine Kolonialflotte aus Handels- und Kriegsschiffen.
//     Kolonialland kostet 1000–1250 Taler je 1000 km²; ein Handelsschiff bedient 4000 km².
//   - Ertrag je Kontinent: bediente Fläche / 2 × Zufall 0,7–1,3 (Rohstoffe, Gold, Gewürze).
//   - Seekrieg gegen die Kolonialflotte eines anderen: Runden, in denen jede Seite je 20 Schiffe
//     des Gegners etwa eines versenkt; der Verteidiger flieht unter 70 % seiner Stärke, wenn er
//     unterlegen ist. Danach sind Handelsschiffe verloren und die Flotte segelt heim.
//
// Remake: Preise, Unterhalt, Deckung und Ertrag wie im Original; Kolonialland zu festen
// 1200 Taler je 1000 km². Ein Angriff je Jahr, nicht gegen Verbündete; der Angreifer dreht ab,
// wenn er die Hälfte seiner Schiffe verloren hat und unterlegen ist. Nur mit Remake-Regeln.

const (
	TraderCost      = 8000
	WarshipCost     = 4000
	TraderUpkeep    = 400
	WarshipUpkeep   = 200
	ColonyParcel    = 1000 // km² je Kauf
	ColonyPrice     = 1200 // Taler je Parzelle
	CoverPerTrader  = 4000 // km², die ein Handelsschiff bedient
	ShipsPerPeople  = 200  // je Jahr ein Schiff je 200 Einwohner
	maxBattleRounds = 60
)

// Continents: Anzeige-Namen und Waren (Index = Kontinent).
var Continents = [3]string{"Afrika", "Amerika", "Asien"}

// Colony ist der Besitz auf einem Kontinent samt dort stationierter Flotte.
type Colony struct {
	Area     int `json:"area"`     // km²
	Traders  int `json:"traders"`  // Handelsschiffe der Kolonialflotte
	Warships int `json:"warships"` // Kriegsschiffe der Kolonialflotte
}

var (
	errShipKind    = &RuleError{"ship_kind", "Unbekannter Schiffstyp."}
	errShipLimit   = &RuleError{"ship_limit", "Ihre Werften schaffen dieses Jahr nicht mehr Schiffe."}
	errContinent   = &RuleError{"continent", "Unbekannter Kontinent."}
	errFleetShips  = &RuleError{"fleet_ships", "So viele Schiffe besitzen Sie nicht."}
	errNoTraders   = &RuleError{"no_traders", "Ohne Handelsschiffe lässt sich dort keine Kolonie halten."}
	errNavalOnce   = &RuleError{"naval_once", "Sie haben dieses Jahr schon eine Flotte entsandt."}
	errNoEnemyShip = &RuleError{"no_enemy_fleet", "Dort liegt keine Flotte dieses Herrschers."}
	errNavalTarget = &RuleError{"naval_target", "Gegen diesen Herrscher geht das nicht."}
)

// shipsAtHome liefert die nicht zugeteilten Handels- und Kriegsschiffe.
func shipsAtHome(p *Player) (traders, warships int) {
	traders, warships = p.Traders, p.Warships
	for _, c := range p.Colonies {
		traders -= c.Traders
		warships -= c.Warships
	}
	return
}

// ShipLimit: Schiffe, die die Werften dieses Jahr noch bauen können.
func ShipLimit(p *Player) int { return max(0, max(1, p.Population/ShipsPerPeople)-p.Year.ShipsBuilt) }

// buildShips gibt Schiffe in Auftrag (Bauphase).
func buildShips(p *Player, b *Balance, kind string, qty int) error {
	if qty <= 0 {
		return ErrInvalidAmount
	}
	if qty > ShipLimit(p) {
		return errShipLimit
	}
	cost := 0
	switch kind {
	case "trader":
		cost = TraderCost
	case "warship":
		cost = WarshipCost
	default:
		return errShipKind
	}
	if err := pay(p, b, float64(cost*qty)); err != nil {
		return err
	}
	if kind == "trader" {
		p.Traders += qty
	} else {
		p.Warships += qty
	}
	p.Year.ShipsBuilt += qty
	return nil
}

// setFleet stellt die Kolonialflotte eines Kontinents neu zusammen (absolute Zahlen).
func setFleet(p *Player, cont, traders, warships int) error {
	if cont < 0 || cont >= len(p.Colonies) {
		return errContinent
	}
	if traders < 0 || warships < 0 {
		return ErrInvalidAmount
	}
	ht, hw := shipsAtHome(p)
	c := &p.Colonies[cont]
	if traders > ht+c.Traders || warships > hw+c.Warships {
		return errFleetShips
	}
	if traders == 0 && warships > 0 {
		return errNoTraders // Kriegsschiffe begleiten nur eine Handelsflotte
	}
	c.Traders, c.Warships = traders, warships
	return nil
}

// colonize erwirbt Kolonialland in Parzellen zu 1000 km² (Bauphase).
func colonize(p *Player, b *Balance, cont, parcels int) error {
	if cont < 0 || cont >= len(p.Colonies) {
		return errContinent
	}
	if parcels <= 0 {
		return ErrInvalidAmount
	}
	if p.Colonies[cont].Traders == 0 {
		return errNoTraders
	}
	if err := pay(p, b, float64(parcels*ColonyPrice)); err != nil {
		return err
	}
	p.Colonies[cont].Area += parcels * ColonyParcel
	return nil
}

// ColonyReport: Kolonialerträge je Kontinent und Unterhalt der ganzen Flotte.
type ColonyReport struct {
	Income [3]int `json:"income"`
	Upkeep int    `json:"upkeep"`
}

// colonyIncome bucht Erträge und Unterhalt (zur Steuer). Ertrag = bediente Fläche / 2 × 0,7–1,3.
func colonyIncome(p *Player, rng *Rng) ColonyReport {
	var r ColonyReport
	for i, c := range p.Colonies {
		served := min(c.Area, CoverPerTrader*c.Traders)
		if served > 0 {
			r.Income[i] = int(float64(served) / 2 * (0.7 + 0.3*(rng.Float()+rng.Float())))
		}
		p.Treasury += r.Income[i]
	}
	r.Upkeep = p.Traders*TraderUpkeep + p.Warships*WarshipUpkeep
	p.Treasury -= r.Upkeep
	return r
}

// NavalReport beschreibt eine Seeschlacht um eine Kolonialflotte.
type NavalReport struct {
	Attacker   int  `json:"attacker"`
	Defender   int  `json:"defender"`
	Continent  int  `json:"continent"`
	Sent       int  `json:"sent"`       // eingesetzte Kriegsschiffe des Angreifers
	Guards     int  `json:"guards"`     // Kriegsschiffe des Verteidigers vor Ort
	Traders    int  `json:"traders"`    // Handelsschiffe des Verteidigers vor Ort
	AttLost    int  `json:"attLost"`    // versenkte Schiffe des Angreifers
	DefLost    int  `json:"defLost"`    // versenkte Kriegsschiffe des Verteidigers
	TradersHit int  `json:"tradersHit"` // verlorene Handelsschiffe des Verteidigers
	Won        bool `json:"won"`        // Verteidiger geflohen
}

// fnu: Zufallsfaktor 1 ± x % (wie im Vorbild).
func (g *Game) fnu(x float64) float64 {
	return 1 + ((g.Rng.Float()+g.Rng.Float())*x-x)/100
}

// navalAttack schickt Kriegsschiffe gegen die Kolonialflotte eines anderen (Bauphase).
func (g *Game) navalAttack(seat, target, cont, sent int) (*NavalReport, error) {
	att := g.Seats[seat].Player
	if !g.validPeer(seat, target) {
		return nil, errNavalTarget
	}
	if err := g.warAllowed(seat, target); err != nil {
		return nil, err
	}
	if cont < 0 || cont >= len(att.Colonies) {
		return nil, errContinent
	}
	if att.Year.NavalAttack {
		return nil, errNavalOnce
	}
	if _, hw := shipsAtHome(att); sent <= 0 || sent > hw {
		return nil, errFleetShips
	}
	def := g.Seats[target].Player
	c := &def.Colonies[cont]
	if c.Traders == 0 && c.Warships == 0 {
		return nil, errNoEnemyShip
	}
	att.Year.NavalAttack = true
	r := &NavalReport{Attacker: seat, Defender: target, Continent: cont, Sent: sent, Guards: c.Warships, Traders: c.Traders}
	ag, vt := sent, c.Warships
	// Runden wie im Original: jede Seite versenkt je 20 Schiffe des Gegners etwa eines
	for round := 0; round < maxBattleRounds; round++ {
		if vt == 0 || (float64(vt)/float64(max(1, r.Guards)) < 0.7 && float64(vt) < float64(ag)*0.9) {
			r.Won = true
			break
		}
		ag -= min(ag, int(math.Ceil(float64(vt)/20*g.fnu(20))))
		if ag == 0 || (ag*2 < sent && ag < vt) {
			break // Angreifer versenkt oder dreht ab
		}
		vt -= min(vt, int(math.Ceil(float64(ag)/20*g.fnu(20))))
	}
	r.AttLost, r.DefLost = sent-ag, r.Guards-vt
	att.Warships -= r.AttLost
	def.Warships -= r.DefLost
	if r.Won {
		// Verlorene Handelsschiffe, die Kolonialflotte segelt heim
		hs := float64(r.Traders)
		vh := int(math.Ceil(hs * float64(ag-vt) / (hs + float64(vt)) / 8 * (1 + g.Rng.Float()/2)))
		if float64(vh) > hs/2 {
			vh = int(hs / 2 * g.fnu(20))
		}
		r.TradersHit = max(0, min(vh, r.Traders))
		def.Traders -= r.TradersHit
		c.Traders, c.Warships = 0, 0
	} else {
		c.Warships = vt
	}
	return r, nil
}

// fixFleet hält die Bestände nach Verlusten stimmig: nie mehr Schiffe zugeteilt als vorhanden,
// Kriegsschiffe nur bei einer Handelsflotte.
func fixFleet(p *Player) {
	p.Traders, p.Warships = max(0, p.Traders), max(0, p.Warships)
	ht, hw := shipsAtHome(p)
	for i := range p.Colonies {
		c := &p.Colonies[i]
		if ht < 0 {
			d := min(c.Traders, -ht)
			c.Traders -= d
			ht += d
		}
		if hw < 0 {
			d := min(c.Warships, -hw)
			c.Warships -= d
			hw += d
		}
		if c.Traders == 0 {
			c.Warships = 0
		}
	}
}

// fleetSize: alle Schiffe in Kolonialflotten (für Schiffsereignisse).
func fleetSize(p *Player) (colonial int) {
	for _, c := range p.Colonies {
		colonial += c.Traders + c.Warships
	}
	return
}
