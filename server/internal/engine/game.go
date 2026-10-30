package engine

import (
	"encoding/json"
	"fmt"
)

// Phase ist der Abschnitt eines Jahres, den jeder Herrscher gleichzeitig durchläuft.
type Phase string

const (
	PhaseMarket Phase = "market" // Ernte ist eingebracht, Korn- und Landhandel
	PhaseFeed   Phase = "feed"   // Volk versorgen
	PhaseTaxes  Phase = "taxes"  // Steuern und Justiz
	PhaseBuild  Phase = "build"  // Staatskäufe und Militär
	PhaseDone   Phase = "done"   // wartet auf die anderen
)

// Seat verbindet einen Herrscher mit seinem Fortschritt im laufenden Jahr.
type Seat struct {
	Player     *Player `json:"player"`
	Phase      Phase   `json:"phase"`
	PhaseStart int64   `json:"phaseStart"` // Unix-Sekunden
	Overtime   int     `json:"overtime"`   // Sekunden über dem Limit, summiert übers Jahr
	TurnTime   int     `json:"turnTime"`   // gesamte Zugdauer im Jahr (Original-Zeitstrafe)
}

// Event ist eine Meldung für Chronik und Hinweise. Player = -1 heißt „an alle“.
type Event struct {
	Type   string `json:"type"`
	Player int    `json:"player"`
	Year   int    `json:"year"`
	Data   any    `json:"data,omitempty"`
}

// Game ist ein laufendes Spiel. Alle Methoden sind nicht nebenläufig sicher;
// der Server ruft sie aus genau einer Goroutine pro Spiel auf.
type Game struct {
	Settings Settings       `json:"settings"`
	Balance  *Balance       `json:"balance"`
	Year     int            `json:"year"`
	Seats    []*Seat        `json:"seats"`
	Over     bool           `json:"over"`
	Winner   int            `json:"winner"`   // Sitz des Kaisers, sonst -1
	Wars     []*War         `json:"wars"`     // Kriegserklärungen des laufenden Jahres
	WarIdx   int            `json:"warIdx"`   // gerade ausgetragener Krieg
	Bots     map[int]string `json:"bots"`     // Sitz → Spielweise (Computer-Herrscher oder Autopilot)
	CPU      map[int]bool   `json:"cpu"`      // Sitze von Computer-Herrschern (ohne Autopilot menschlicher Spieler)
	Log      []Event        `json:"log"`      // Chronik für Wiedereinstieg (begrenzt, siehe Record)
	Pacts    []Pact         `json:"pacts"`    // Verteidigungsabkommen (buendnis.go)
	Offers   []Pact         `json:"offers"`   // offene Angebote A -> B (verfallen am Jahresende)
	Canceled []Pact         `json:"canceled"` // in diesem Jahr gekündigt (Angriff erst nächstes Jahr)
	Declined []Pact         `json:"declined"` // in diesem Jahr abgelehnte Angebote
	Bonds    []Bond         `json:"bonds"`    // Herrscheranleihen (finanzen.go)
	Loans    []PeerLoan     `json:"loans"`    // Kredite unter Herrschern
	Rng      *Rng           `json:"-"`
	World    *World         `json:"-"` // Routen; vom Server nach New/Restore gesetzt
}

// NewGame legt ein Spiel ohne Spieler an.
func NewGame(s Settings, b *Balance, seed uint64) *Game {
	bb := *b
	bb.Classic = s.Classic
	return &Game{Settings: s, Balance: &bb, Year: bb.Start.StartYear, Rng: NewRng(seed), Winner: -1, CPU: map[int]bool{}}
}

// AddPlayer setzt einen Herrscher an einen freien Platz (Land 1..9).
func (g *Game) AddPlayer(name string, country int, female bool) (int, error) {
	if country < 1 || country > 9 {
		return 0, &RuleError{"invalid_country", "Ungültiges Land."}
	}
	if len(g.Seats) >= MaxSeats {
		return 0, &RuleError{"room_full", "Der Reichstag ist voll."}
	}
	// Remake: bis zu drei Herrscher teilen sich ein Land; Original: eines je Herrscher.
	if g.CountryCount(country) >= g.PerCountry() {
		return 0, &RuleError{"country_taken", "Dieses Land ist schon vergeben."}
	}
	p := NewPlayer(name, country, female, g.Balance, g.Rng)
	// Sterblichkeit aus der Lobby verschiebt die Lebenserwartung
	off, _ := g.Settings.MortalityRule()
	p.DeathYear += off
	g.Seats = append(g.Seats, &Seat{Player: p, Phase: PhaseDone})
	return len(g.Seats) - 1, nil
}

// JoinRunning setzt einen neuen Herrscher in ein laufendes Spiel. Er wartet bis zum nächsten
// Jahr (Phase done) und wird beim Jahresende des Beitrittsjahres übersprungen, damit er nicht
// für ein Jahr abgerechnet wird, das er gar nicht gespielt hat.
func (g *Game) JoinRunning(name string, country int, female bool) (int, error) {
	seat, err := g.AddPlayer(name, country, female)
	if err == nil {
		g.Seats[seat].Player.Joined = g.Year
	}
	return seat, err
}

// Promoted meldet, ob schon ein Herrscher über den Starttitel aufgestiegen ist. Ab dann ist
// kein Beitritt mehr möglich (Neue hätten keine faire Chance mehr).
func (g *Game) Promoted() bool {
	start := max(1, min(8, g.Balance.Start.Title))
	for _, s := range g.Seats {
		if s.Player.Title > start {
			return true
		}
	}
	return false
}

// RemoveSeat entfernt einen Herrscher vor Spielbeginn (Gastgeber wirft Störer hinaus).
// Spätere Sitze rücken auf; Computer-Zuordnungen werden mitverschoben. Vor dem Start gibt
// es noch keine Bündnisse, Kredite oder Kriege, die auf Sitznummern zeigen.
func (g *Game) RemoveSeat(seat int) {
	g.Seats = append(g.Seats[:seat], g.Seats[seat+1:]...)
	shift := func(k int) int {
		if k > seat {
			return k - 1
		}
		return k
	}
	bots, cpu := map[int]string{}, map[int]bool{}
	for k, v := range g.Bots {
		if k != seat {
			bots[shift(k)] = v
		}
	}
	for k, v := range g.CPU {
		if k != seat {
			cpu[shift(k)] = v
		}
	}
	g.Bots, g.CPU = bots, cpu
}

// MaxSeats begrenzt die Herrscher je Partie (Vorgabe 6; Server-Flag -max-players).
var MaxSeats = 6

// PerCountry: wie viele Herrscher sich ein Land teilen dürfen.
func (g *Game) PerCountry() int {
	if g.Balance.Classic {
		return 1
	}
	return 3
}

// CountryCount zählt die Herrscher eines Landes.
func (g *Game) CountryCount(country int) int {
	n := 0
	for _, s := range g.Seats {
		if s.Player.Country == country {
			n++
		}
	}
	return n
}

// FreeLand ist die noch nicht vergebene Fläche eines Landes (Remake). Ohne Grenze
// (Original oder CountryArea 0) liefert sie -1.
func (g *Game) FreeLand(country int) int {
	if g.Balance.Classic || g.Balance.CountryArea <= 0 {
		return -1
	}
	used := 0
	for _, s := range g.Seats {
		if s.Player.Country == country && s.Player.Alive() {
			used += s.Player.Land
		}
	}
	return max(0, g.Balance.CountryArea-used)
}

// MapArea ist die Landesfläche für die Kartendarstellung; alte Spielstände ohne
// countryArea zeigen die Anteile gegen den heutigen Standard (50 000 ha).
func (g *Game) MapArea() int {
	if g.Balance.CountryArea > 0 {
		return g.Balance.CountryArea
	}
	return DefaultBalance().CountryArea
}

// Start beginnt das erste Jahr.
func (g *Game) Start(now int64) []Event {
	return g.startYear(now)
}

// startYear erntet für alle aktiven Herrscher und setzt sie in die Marktphase.
func (g *Game) startYear(now int64) []Event {
	var ev []Event
	for i, s := range g.Seats {
		p := s.Player
		s.Overtime, s.TurnTime = 0, 0
		if !p.Alive() {
			s.Phase = PhaseDone
			continue
		}
		if p.SuspendedUntil > g.Year {
			s.Phase = PhaseDone
			ev = append(ev, Event{"suspended", i, g.Year, nil})
			continue
		}
		rep := Harvest(p, g.Balance, g.Rng)
		DrawTaxFactors(p, g.Rng)
		DrawUnitPrices(p, g.Rng)
		s.Phase, s.PhaseStart = PhaseMarket, now
		ev = append(ev, Event{"harvest", i, g.Year, rep})
		// Zufallsereignis oder Risiko-Entscheidung des Jahres (Kaiser II)
		if er, risk := g.RollEvent(i); er != nil {
			ev = append(ev, Event{"ereignis", i, g.Year, er})
		} else if risk != "" {
			p.Year.Risk = risk
			d := RiskByID(risk)
			ev = append(ev, Event{"risiko", i, g.Year, map[string]string{"id": d.ID, "title": d.Title, "question": d.Question}})
		}
	}
	if g.allDone() {
		ev = append(ev, g.afterAllDone(now)...)
	}
	return ev
}

// Action ist ein Spielzug vom Client. Nur die zur Art passenden Felder zählen.
type Action struct {
	Type       string      `json:"type"`
	Qty        int         `json:"qty,omitempty"`
	Customs    int         `json:"customs,omitempty"`
	VAT        int         `json:"vat,omitempty"`
	IncomeTax  int         `json:"incomeTax,omitempty"`
	Justice    int         `json:"justice,omitempty"`
	Building   string      `json:"building,omitempty"`
	Unit       string      `json:"unit,omitempty"`
	Mercenary  bool        `json:"mercenary,omitempty"`
	Target     int         `json:"target,omitempty"`     // Sitz des Angegriffenen
	Stance     string      `json:"stance,omitempty"`     // Haltung in der Diplomatie
	Placements []Placement `json:"placements,omitempty"` // taktische Aufstellung
	Motto      string      `json:"motto,omitempty"`      // Profil
	Arms       *ArmsSpec   `json:"arms,omitempty"`       // Profil: Landeswappen (Upload separat)
	Rate       int         `json:"rate,omitempty"`       // Kredit an Mitspieler: Jahreszins in %
	Years      int         `json:"years,omitempty"`      // Kredit an Mitspieler: Laufzeit
	Continent  int         `json:"continent,omitempty"`  // Kolonien: 0 Afrika, 1 Amerika, 2 Asien
	Warships   int         `json:"warships,omitempty"`   // Kolonialflotte / Seekrieg: Kriegsschiffe
}

// ErrWrongPhase meldet eine Aktion zur falschen Zeit.
var ErrWrongPhase = &RuleError{"wrong_phase", "Das geht in dieser Phase nicht."}

// Apply führt eine Aktion eines Sitzes aus und liefert die entstandenen Ereignisse.
func (g *Game) Apply(seat int, a Action, now int64) ([]Event, error) {
	if g.Over {
		return nil, &RuleError{"game_over", "Das Spiel ist beendet."}
	}
	if seat < 0 || seat >= len(g.Seats) {
		return nil, fmt.Errorf("ungültiger Sitz %d", seat)
	}
	s := g.Seats[seat]
	p, b := s.Player, g.Balance
	var ev []Event
	emit := func(t string, data any) { ev = append(ev, Event{t, seat, g.Year, data}) }

	// Bündnisangebote beantworten geht in jeder Phase (das Angebot gilt bis Jahresende).
	if a.Type == "acceptPact" || a.Type == "declinePact" {
		if err := g.answerPact(seat, a.Target, a.Type == "acceptPact"); err != nil {
			return nil, err
		}
		emit("pactAnswer", map[string]any{"from": a.Target, "accept": a.Type == "acceptPact"})
		return ev, nil
	}
	// Profiländerungen sind unabhängig von der Phase; hochgeladene Wappen setzt der Server direkt.
	if a.Type == "setProfile" {
		if a.Arms != nil && a.Arms.Kind == "upload" {
			return nil, ErrBadProfile
		}
		return nil, setProfile(p, a.Motto, a.Arms)
	}

	// Geldgeschäfte (Kaiser II) in der Markt- und der Bauphase
	if isFinance(a.Type) {
		if s.Phase != PhaseMarket && s.Phase != PhaseBuild {
			return nil, errFinBadPhase
		}
		data, err := g.finance(seat, a)
		if err != nil {
			return nil, err
		}
		if data != nil {
			emit("finanz", data)
		}
		return ev, nil
	}

	switch s.Phase {
	case PhaseMarket:
		var err error
		switch a.Type {
		case "buyGrain":
			err = BuyGrain(p, b, a.Qty)
		case "sellGrain":
			err = SellGrain(p, a.Qty)
		case "buyLand":
			// Geteilte Länder: nur freie Fläche des eigenen Landes ist käuflich
			if free := g.FreeLand(p.Country); free >= 0 && a.Qty > free {
				err = &RuleError{"no_free_land", "In Ihrem Land ist nicht mehr so viel Fläche frei."}
				break
			}
			err = BuyLand(p, b, a.Qty)
		case "sellLand":
			err = SellLand(p, a.Qty)
		case "risk": // Antwort auf die Risiko-Entscheidung: Stance "ja" oder "nein"
			if p.Year.Risk == "" {
				return nil, ErrWrongPhase
			}
			if rep := g.AnswerRisk(seat, a.Stance == "ja"); rep != nil {
				emit("ereignis", rep)
			}
		case "next":
			// Unbeantwortete Risiko-Entscheidung gilt als „Nein“
			if p.Year.Risk != "" {
				if rep := g.AnswerRisk(seat, false); rep != nil {
					emit("ereignis", rep)
				}
			}
			g.advance(s, PhaseFeed, now)
		default:
			return nil, ErrWrongPhase
		}
		if err != nil {
			return nil, err
		}
	case PhaseFeed:
		if a.Type != "feed" {
			return nil, ErrWrongPhase
		}
		rep, err := Feed(p, b, g.Rng, a.Qty)
		if err != nil {
			return nil, err
		}
		emit("fed", rep)
		inc := CollectIncome(p, g.Rng)
		p.Year.Earnings = inc.Markets + inc.Mills
		emit("income", inc)
		g.advance(s, PhaseTaxes, now)
	case PhaseTaxes:
		switch a.Type {
		case "setTaxes":
			if err := SetTaxes(p, a.Customs, a.VAT, a.IncomeTax, a.Justice); err != nil {
				return nil, err
			}
		case "fest": // Volksfest: Stimmung +1
			if err := holdFest(p, b); err != nil {
				return nil, err
			}
		case "opposition": // Opposition verhaften: Stimmung +3, einmal im Jahr
			if err := arrestOpposition(p, b); err != nil {
				return nil, err
			}
		case "gendarm": // Gendarmen anwerben (Innenpolitik)
			if err := hireGendarmes(p, b, a.Qty); err != nil {
				return nil, err
			}
		case "next":
			// Kolonialerträge und Flottenunterhalt (Kaiser II) vor der Steuer
			col := 0
			if !b.Classic && (p.Traders > 0 || p.Warships > 0) {
				cr := colonyIncome(p, g.Rng)
				col = cr.Income[0] + cr.Income[1] + cr.Income[2]
				emit("kolonien", cr)
			}
			tr := CollectTaxes(p)
			emit("taxes", tr)
			// Besitzer eigener Anleihen bekommen ihren Anteil an den Steuern
			if paid := g.payDividends(seat, tr.Total+p.Year.Earnings+col); paid > 0 {
				emit("anleihen", map[string]int{"paid": paid})
			}
			chk := AfterTaxChecks(p, b, g.Rng, g.Year)
			emit("checks", chk)
			next := PhaseBuild
			if chk.Bankrupt || chk.Suspended {
				next = PhaseDone
			}
			g.advance(s, next, now)
		default:
			return nil, ErrWrongPhase
		}
	case PhaseBuild:
		var err error
		switch a.Type {
		case "build":
			err = Build(p, b, g.Rng, Building(a.Building))
		case "recruit":
			err = Recruit(p, b, Unit(a.Unit), a.Mercenary)
		case "maneuver":
			err = Maneuver(p, b)
		case "buildShip", "fleet", "colonize", "navalAttack":
			if b.Classic {
				return nil, errFinClassic
			}
			switch a.Type {
			case "buildShip":
				err = buildShips(p, b, a.Unit, a.Qty)
			case "fleet":
				err = setFleet(p, a.Continent, a.Qty, a.Warships)
			case "colonize":
				err = colonize(p, b, a.Continent, a.Qty)
			case "navalAttack":
				var rep *NavalReport
				if rep, err = g.navalAttack(seat, a.Target, a.Continent, a.Warships); err == nil {
					emit("seekrieg", rep)
				}
			}
		case "train":
			err = Train(p, b)
		case "equip":
			err = Equip(p, b)
		case "endTurn":
			g.advance(s, PhaseDone, now)
		case "proposePact":
			if err = g.proposePact(seat, a.Target); err == nil {
				emit("pactOffer", map[string]int{"target": a.Target})
				// Computer-Herrscher antworten sofort (sonst verfiele das Angebot, wenn sie schon fertig sind)
				if _, bot := g.Bots[a.Target]; bot && g.offered(seat, a.Target) {
					accept := g.strongestRivalSeat(a.Target) != seat
					if g.answerPact(a.Target, seat, accept) == nil {
						ev = append(ev, Event{"pactAnswer", a.Target, g.Year, map[string]any{"from": seat, "accept": accept}})
					}
				}
			}
		case "cancelPact":
			err = g.cancelPact(seat, a.Target)
		case "declareWar":
			if err = g.declareWar(seat, a.Target); err == nil {
				emit("warPlanned", map[string]int{"target": a.Target})
				g.advance(s, PhaseDone, now)
			}
		default:
			return nil, ErrWrongPhase
		}
		if err != nil {
			return nil, err
		}
	case PhaseDiplomacy:
		w := g.current()
		if a.Type != "stance" || w == nil {
			return nil, ErrWrongPhase
		}
		if _, ok := stanceCode[a.Stance]; !ok {
			return nil, ErrBadStance
		}
		w.Stances[seat] = a.Stance
		g.advance(s, PhaseDone, now)
	case PhaseDeploy:
		var pl []Placement
		switch a.Type {
		case "deploy":
			pl = a.Placements
		case "autoDeploy":
		default:
			return nil, ErrWrongPhase
		}
		if err := g.setDeployment(seat, pl); err != nil {
			return nil, err
		}
		g.advance(s, PhaseDone, now)
	default:
		return nil, ErrWrongPhase
	}

	if g.allDone() {
		ev = append(ev, g.afterAllDone(now)...)
	}
	return ev, nil
}

// advance wechselt die Phase und verbucht die Überzeit der alten Phase.
func (g *Game) advance(s *Seat, next Phase, now int64) {
	used := int(now - s.PhaseStart)
	s.Overtime += g.Settings.Overtime(used)
	s.TurnTime += used
	s.Phase, s.PhaseStart = next, now
}

// Deadline liefert das Ende der laufenden Phase (Unix-Sekunden) oder 0 ohne Limit.
func (g *Game) Deadline(seat int) int64 {
	s := g.Seats[seat]
	if g.Settings.PhaseSeconds == 0 || s.Phase == PhaseDone {
		return 0
	}
	return s.PhaseStart + int64(g.Settings.PhaseSeconds)
}

// Timeouts beendet abgelaufene Phasen automatisch, damit niemand das Spiel aufhält.
// Ohne Strafe wird hart nach dem Limit weitergeschaltet, mit Strafe nach doppelter Zeit.
func (g *Game) Timeouts(now int64) []Event {
	if g.Settings.PhaseSeconds == 0 || g.Over {
		return nil
	}
	grace := int64(g.Settings.PhaseSeconds)
	if g.Settings.TimePenalty {
		grace *= 2
	}
	var ev []Event
	for i, s := range g.Seats {
		if s.Phase == PhaseDone || now-s.PhaseStart < grace {
			continue
		}
		ev = append(ev, Event{"timeout", i, g.Year, s.Phase})
		var a Action
		switch s.Phase {
		case PhaseMarket:
			a = Action{Type: "next"}
		case PhaseFeed:
			// Automatisch der Bedarf, begrenzt auf die erlaubte Spanne.
			lo, hi := FeedLimits(s.Player)
			a = Action{Type: "feed", Qty: max(lo, min(hi, s.Player.Year.GrainNeed))}
		case PhaseTaxes:
			a = Action{Type: "next"}
		case PhaseBuild:
			a = Action{Type: "endTurn"}
		case PhaseDiplomacy:
			a = Action{Type: "stance", Stance: StanceNeutral}
		case PhaseDeploy:
			a = Action{Type: "autoDeploy"}
		}
		more, _ := g.Apply(i, a, now)
		ev = append(ev, more...)
	}
	return ev
}

func (g *Game) allDone() bool {
	for _, s := range g.Seats {
		if s.Phase != PhaseDone {
			return false
		}
	}
	return true
}

// endYear rechnet das Jahresende für alle ab und startet das nächste Jahr.
func (g *Game) endYear(now int64) []Event {
	var ev []Event
	fin := g.yearEndFinance()
	for i, s := range g.Seats {
		p := s.Player
		if !p.Alive() || p.Joined >= g.Year {
			continue // neu Beigetretene starten erst im nächsten Jahr
		}
		if !fin[i].empty() {
			ev = append(ev, Event{"finanzen", i, g.Year, fin[i]})
		}
		// Volksstimmung und Umstürze (Kaiser II); ein Herrscher im Exil scheidet aus
		if mr := g.yearMood(i); mr != nil {
			ev = append(ev, Event{"stimmung", i, g.Year, mr})
			if !p.Alive() {
				ev = append(ev, Event{"exil", i, g.Year, mr})
				continue
			}
		}
		p.Gendarmes = min(p.Gendarmes, p.Population/GendarmsPer)
		// Ausbildung verfällt ohne Übung (Kaiser II)
		if p.Training > 0 && g.Rng.Float() < 0.3 {
			p.Training--
		}
		turn := s.TurnTime
		if !g.Settings.TimePenalty || g.Settings.PhaseSeconds == 0 {
			turn = 0 // Original-Strafe nur, wenn das Spiel überhaupt ein Zeitlimit hat
		}
		_, chance := g.Settings.MortalityRule()
		rep := EndYear(p, g.Balance, g.Rng, g.Year, turn, s.Overtime, chance)
		ev = append(ev, Event{"yearEnd", i, g.Year, rep})
		if rep.Emperor {
			g.Over, g.Winner = true, i
		}
		if rep.Died {
			ev = append(ev, Event{"died", i, g.Year, nil})
		}
	}
	// Das Spiel läuft, bis jemand Kaiser wird oder kein menschlicher Herrscher mehr lebt; wer
	// als Letzter übrig bleibt, kann weiter nach der Krone streben. Reine Computer-Partien
	// (Simulation) enden, wenn alle tot sind.
	humans, humansAlive, alive := 0, 0, 0
	for i, s := range g.Seats {
		if s.Player.Alive() {
			alive++
		}
		if !g.isCPU(i) {
			humans++
			if s.Player.Alive() {
				humansAlive++
			}
		}
	}
	if alive == 0 || (humans > 0 && humansAlive == 0) {
		g.Over = true
	}
	if g.Over {
		ev = append(ev, Event{"gameOver", -1, g.Year, map[string]int{"winner": g.Winner}})
		return ev
	}
	g.yearEndPacts()
	g.Year++
	ev = append(ev, Event{"newYear", -1, g.Year, nil})
	return append(ev, g.startYear(now)...)
}

// PublicPlayer ist das, was alle über einen Herrscher sehen (wie die Spielstand-Tabelle im Original).
type PublicPlayer struct {
	Seat       int      `json:"seat"`
	Name       string   `json:"name"`
	Country    int      `json:"country"`
	Female     bool     `json:"female"`
	Title      int      `json:"title"`
	Alive      bool     `json:"alive"`
	Phase      Phase    `json:"phase"`
	Land       int      `json:"land"`
	Treasury   int      `json:"treasury"`
	Population int      `json:"population"`
	Soldiers   int      `json:"soldiers"`
	Nobility   int      `json:"nobility"`
	Motto      string   `json:"motto"`
	Avatar     string   `json:"avatar"`
	Arms       ArmsSpec `json:"arms"`
	Bot        string   `json:"bot"` // Spielweise, wenn der Computer spielt
	// Bauten für die Reichskarte (Symbole im Gebiet des Herrschers)
	Markets   int `json:"markets"`
	Mills     int `json:"mills"`
	Palace    int `json:"palace"`
	Cathedral int `json:"cathedral"`
	// Kolonien und Kolonialflotten sind öffentlich (Kaiser II zeigt sie vor dem Seekrieg)
	Colonies [3]Colony `json:"colonies"`
	Traders  int       `json:"traders"`
	Warships int       `json:"warships"`
}

// Public liefert die öffentliche Übersicht aller Herrscher.
func (g *Game) Public() []PublicPlayer {
	out := make([]PublicPlayer, len(g.Seats))
	for i, s := range g.Seats {
		p := s.Player
		out[i] = PublicPlayer{i, p.Name, p.Country, p.Female, p.Title, p.Alive(), s.Phase,
			p.Land, p.Treasury, p.Population, p.Soldiers, p.Nobility, p.Motto, p.Avatar, p.Arms, g.Bots[i],
			p.Markets, p.Mills, p.Palace, p.Cathedral, p.Colonies, p.Traders, p.Warships}
	}
	return out
}

// SelfView ist der volle Zustand für den eigenen Herrscher inkl. Vorschauwerten.
type SelfView struct {
	Seat      int          `json:"seat"`
	Player    *Player      `json:"player"`
	Phase     Phase        `json:"phase"`
	Deadline  int64        `json:"deadline"`
	FeedMin   int          `json:"feedMin"`
	FeedMax   int          `json:"feedMax"`
	Taxes     TaxReport    `json:"taxes"`
	UnitCosts unitCosts    `json:"unitCosts"`         // Rekruten
	MercCosts unitCosts    `json:"mercCosts"`         // Söldner (+50 % des Grundpreises)
	War       *WarInfo     `json:"war,omitempty"`     // nur in Diplomatie- und Aufstellungsphase
	FreeLand  int          `json:"freeLand"`          // freie Fläche im eigenen Land, -1 = unbegrenzt
	Offers    []int        `json:"pactOffers"`        // Herrscher, die ein Abkommen anbieten
	Sent      []int        `json:"pactSent"`          // eigene offene Angebote
	Finance   *FinanceView `json:"finance,omitempty"` // Geldgeschäfte (nur Remake-Regeln)
	ShipLimit int          `json:"shipLimit"`         // in diesem Jahr noch baubare Schiffe
}

// WarInfo ist der Kriegsausschnitt, den ein beteiligter Spieler sehen darf.
type WarInfo struct {
	Attacker int            `json:"attacker"`
	Defender int            `json:"defender"`
	Side     int            `json:"side"` // 0 Angreifer, 1 Verteidiger, -1 unbeteiligt
	Route    []int          `json:"route,omitempty"`
	Field    *Field         `json:"field,omitempty"`
	Counts   map[string]int `json:"counts,omitempty"`
}

type unitCosts struct {
	Cavalry, Artillery, Infantry int
}

// Self baut die Sicht eines Sitzes. Die verdeckten Steuerfaktoren gehen nicht mit hinaus.
func (g *Game) Self(seat int) SelfView {
	s := g.Seats[seat]
	cp := *s.Player
	cp.Year.TaxFactors = TaxFactors{}
	v := SelfView{Seat: seat, Player: &cp, Phase: s.Phase, Deadline: g.Deadline(seat), Taxes: Taxes(s.Player)}
	v.FeedMin, v.FeedMax = FeedLimits(s.Player)
	v.FreeLand = g.FreeLand(s.Player.Country)
	v.Offers = g.OffersTo(seat)
	v.ShipLimit = ShipLimit(s.Player)
	if !g.Balance.Classic {
		fv := g.financeView(seat)
		v.Finance = &fv
	}
	for _, x := range g.Offers {
		if x.A == seat {
			v.Sent = append(v.Sent, x.B)
		}
	}
	if w := g.current(); w != nil && (s.Phase == PhaseDiplomacy || s.Phase == PhaseDeploy) {
		info := &WarInfo{Attacker: w.Attacker, Defender: w.Defender, Side: -1, Route: w.Route}
		if seat == w.Attacker || seat == w.Defender {
			info.Side = 0
			if seat == w.Defender {
				info.Side = 1
			}
			if w.Battle != nil {
				info.Field = &w.Battle.Field
				info.Counts = g.sideCounts(w, info.Side)
			}
		}
		v.War = info
	}
	for _, merc := range []bool{false, true} {
		c := unitCosts{UnitCost(s.Player, Cavalry, merc), UnitCost(s.Player, Artillery, merc), UnitCost(s.Player, Infantry, merc)}
		if merc {
			v.MercCosts = c
		} else {
			v.UnitCosts = c
		}
	}
	return v
}

// RestoreGame stellt ein Spiel aus einem Snapshot wieder her.
func RestoreGame(data []byte) (*Game, error) {
	var aux struct {
		Game
		RngState []byte `json:"rng"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return nil, fmt.Errorf("snapshot lesen: %w", err)
	}
	g := aux.Game
	g.Rng = &Rng{}
	if err := g.Rng.UnmarshalBinary(aux.RngState); err != nil {
		return nil, fmt.Errorf("zufallszustand: %w", err)
	}
	if g.Balance == nil {
		return nil, fmt.Errorf("snapshot ohne balance")
	}
	return &g, nil
}

// Snapshot serialisiert das Spiel samt Zufallszustand (für Persistenz).
func (g *Game) Snapshot() ([]byte, error) {
	rs, err := g.Rng.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		*Game
		RngState []byte `json:"rng"`
	}{g, rs})
}
