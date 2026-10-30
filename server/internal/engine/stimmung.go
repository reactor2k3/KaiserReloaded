package engine

// Volksstimmung, Gendarmen und Umstürze nach dem Vorbild Kaiser II (Stimmung 0–10, Gendarmen,
// Revolution/Putsch/Attentat). Die Regeln sind für das Remake
// vereinfacht und hier vollständig beschrieben:
//
// Stimmung 0 (aufständisch) bis 10 (euphorisch), Start 5. Jährliche Änderung nach den Steuern:
//   Versorgung: unter 90 % des Bedarfs −2, unter 100 % −1, ab 120 % +1, ab 150 % +2
//   Steuern (Zoll+MwSt+Einkommen): bis 25 % +1, über 60 % −1, über 90 % −2
//   Justiz: sehr fair +1, fair 0, hart −1, gierig −2;  Krieg im Jahr −1;  Zufall −1..+1
//   Volksfest (Steuerphase): +1 je Fest;  Opposition verhaften: +3, einmal im Jahr
//   Gewöhnung: unter 4 mit 50 % Chance +1
// Wirkung: Steuereinnahmen ×(0,85 + 0,03 × Stimmung), also 85 % bei 0 bis 115 % bei 10.
// Aufstand bei Stimmung ≤ 1: 15 % bei Stimmung 1, 30 % bei 0. Niederschlagen gelingt
// mit 35 % + 60 % × Gendarmenabdeckung (Abdeckung 1 = ein Gendarm je 40 Einwohner).
// Militärputsch bei Moral unter 25 % und über 100 Soldaten: 15 % je Jahr.
// Attentat: 1,5 % je Jahr, vereitelt mit 40 % + 60 % × Abdeckung, sonst schwere Verletzung.
// Scheitert ein Herrscher an einer Revolution bei Stimmung 0 (halbe Chance, sonst Plünderung)
// oder an einem Putsch, geht er ins Exil:
// Für ihn ist das Spiel zu Ende.

// Kosten der Innenpolitik
const (
	GendarmCost   = 100 // Taler je Gendarm (Anwerbung)
	GendarmPay    = 20  // Sold je Gendarm und Jahr
	GendarmsPer   = 40  // höchstens ein Gendarm je 40 Einwohner
	FestCostTitle = 800 // Volksfest: 800 Taler je Titelstufe
	OppoCostTitle = 2500
)

// MoodNames: Bezeichnungen der Stimmung wie in Kaiser II (Index = Stimmung).
var MoodNames = []string{"aufständisch", "ganz mies", "unzufrieden", "schlecht", "unruhig", "normal",
	"ruhig", "zufrieden", "gut", "optimistisch", "euphorisch"}

// ensureMood setzt die Startstimmung für Spielstände aus der Zeit vor der Stimmung.
func ensureMood(p *Player) {
	if !p.MoodInit {
		p.Mood, p.MoodInit = 5, true
	}
}

// MoodFactor ist der Faktor auf die Steuereinnahmen.
func MoodFactor(p *Player) float64 {
	if !p.MoodInit {
		return 1
	}
	return 0.85 + 0.03*float64(p.Mood)
}

// coverage: Anteil der Gendarmen am Bedarf (1 = ein Gendarm je 40 Einwohner), höchstens 1.
func coverage(p *Player) float64 {
	need := float64(max(1, p.Population/GendarmsPer))
	return min(1, float64(p.Gendarmes)/need)
}

// MoodReport ist die jährliche Bilanz der Innenpolitik.
type MoodReport struct {
	Before int    `json:"before"`
	After  int    `json:"after"`
	Name   string `json:"name"`
	Unrest string `json:"unrest,omitempty"` // revolution, putsch, attentat
	Result string `json:"result,omitempty"` // niedergeschlagen, exil, vereitelt, verletzt
	People int    `json:"people,omitempty"` // Verluste an Einwohnern
	Troops int    `json:"troops,omitempty"` // Verluste an Soldaten (Mann)
	Years  int    `json:"years,omitempty"`  // Lebensjahre (Attentat)
}

func clampMood(m int) int { return max(0, min(10, m)) }

// yearMood rechnet Stimmung und Umstürze eines Herrschers am Jahresende.
func (g *Game) yearMood(seat int) *MoodReport {
	p := g.Seats[seat].Player
	if g.Balance.Classic || !p.Alive() {
		return nil
	}
	ensureMood(p)
	r := &MoodReport{Before: p.Mood}
	d := 0
	switch fr := p.Year.FedRatio; {
	case !p.Year.Fed:
	case fr < 0.9:
		d -= 2
	case fr < 1:
		d--
	case fr >= 1.5:
		d += 2
	case fr >= 1.2:
		d++
	}
	switch t := p.Customs + p.VAT + p.IncomeTax; {
	case t <= 25:
		d++
	case t > 90:
		d -= 2
	case t > 60:
		d--
	}
	d += []int{0, 1, 0, -1, -2}[max(0, min(4, p.Justice))]
	for _, w := range g.Wars {
		if w.Attacker == seat || w.Defender == seat {
			d--
			break
		}
	}
	d += g.Rng.IntN(3) - 1
	// Gewöhnung: ein sehr unzufriedenes Volk beruhigt sich mit der Zeit etwas (halbe Chance auf +1)
	if p.Mood < 4 && g.Rng.Float() < 0.5 {
		d++
	}
	p.Mood = clampMood(p.Mood + d)

	cov := coverage(p)
	switch {
	case p.Mood <= 1 && g.Rng.Float() < 0.15*float64(2-p.Mood):
		r.Unrest = "revolution"
		if g.Rng.Float() < 0.35+0.6*cov {
			r.Result = "niedergeschlagen"
			r.People = int(float64(p.Population) * (0.05 + 0.10*g.Rng.Float()))
			p.Population = max(g.Balance.MinPopulation, p.Population-r.People)
			p.Mood = clampMood(p.Mood + 2)
		} else if p.Mood == 0 && g.Rng.Float() < 0.5 {
			r.Result = "exil"
			p.DeathYear = 0
		} else {
			r.Result = "pluenderung"
			r.People = int(float64(p.Population) * 0.15)
			p.Population = max(g.Balance.MinPopulation, p.Population-r.People)
			p.Treasury -= p.Treasury / 5
			p.Mood = clampMood(p.Mood + 1)
		}
	case p.Morale < 0.25 && p.Soldiers > 100 && g.Rng.Float() < 0.15:
		r.Unrest = "putsch"
		if g.Rng.Float() < 0.8+0.2*cov {
			r.Result = "niedergeschlagen"
			r.Troops = p.Soldiers / 2
			p.Cavalry, p.Artillery, p.Infantry = p.Cavalry/2, p.Artillery/2, p.Infantry/2
			p.Morale = 0.5
			p.RecalcArmy()
		} else {
			r.Result = "exil"
			p.DeathYear = 0
		}
	case g.Rng.Float() < 0.015:
		r.Unrest = "attentat"
		if g.Rng.Float() < 0.4+0.6*cov {
			r.Result = "vereitelt"
		} else {
			r.Result = "verletzt"
			r.Years = -(2 + g.Rng.IntN(3))
			p.DeathYear = max(g.Year+1, p.DeathYear+r.Years)
		}
	}
	r.After = p.Mood
	r.Name = MoodNames[p.Mood]
	return r
}

// hireGendarmes wirbt Gendarmen an (Steuerphase, Innenpolitik), höchstens einer je 40 Einwohner.
func hireGendarmes(p *Player, b *Balance, qty int) error {
	if qty <= 0 {
		return ErrInvalidAmount
	}
	if p.Gendarmes+qty > p.Population/GendarmsPer {
		return &RuleError{"too_many_gendarmes", "So viele Gendarmen braucht Ihr Land nicht (höchstens einer je 40 Einwohner)."}
	}
	if err := pay(p, b, float64(qty*GendarmCost)); err != nil {
		return err
	}
	p.Gendarmes += qty
	return nil
}

// holdFest (Steuerphase): Volksfest, Stimmung +1, je Titelstufe 800 Taler.
func holdFest(p *Player, b *Balance) error {
	ensureMood(p)
	if p.Mood >= 10 {
		return &RuleError{"mood_max", "Ihr Volk ist bereits euphorisch."}
	}
	if err := pay(p, b, float64(FestCostTitle*p.Title)); err != nil {
		return err
	}
	p.Mood++
	return nil
}

// arrestOpposition (Steuerphase): Stimmung +3, einmal je Jahr, je Titelstufe 2 500 Taler.
func arrestOpposition(p *Player, b *Balance) error {
	ensureMood(p)
	if p.Year.Opposition {
		return &RuleError{"no_opposition", "Es gibt keine Opposition mehr."}
	}
	if err := pay(p, b, float64(OppoCostTitle*p.Title)); err != nil {
		return err
	}
	p.Year.Opposition = true
	p.Mood = clampMood(p.Mood + 3)
	return nil
}
