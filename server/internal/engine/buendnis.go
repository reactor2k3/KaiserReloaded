package engine

// Verteidigungsabkommen nach dem Vorbild Kaiser II (Außenpolitik: „Verteidigungsabkommen
// abschließen oder kündigen“, „Risiko eines Krieges mit … und seinen Verbündeten“).
//
// Remake:
//   - Ein Herrscher bietet in seiner Bauphase einem anderen ein Abkommen an; der nimmt in seiner
//     Bauphase an oder lehnt ab. Angebote verfallen am Jahresende.
//   - Verbündete greifen einander nicht an. Wer kündigt, darf den bisherigen Partner im selben
//     Jahr noch nicht angreifen.
//   - Wird ein Herrscher angegriffen, stehen ihm seine Verbündeten automatisch bei (Haltung
//     „Hilfe für den Angegriffenen“), sofern sie einen Weg zu ihm haben.

// Pact ist ein Abkommen (A < B) oder ein Angebot (A bietet B an).
type Pact struct {
	A int `json:"a"`
	B int `json:"b"`
}

func pair(a, b int) Pact {
	if a > b {
		a, b = b, a
	}
	return Pact{a, b}
}

// Allied meldet, ob zwei Herrscher verbündet sind.
func (g *Game) Allied(a, b int) bool {
	p := pair(a, b)
	for _, x := range g.Pacts {
		if x == p {
			return true
		}
	}
	return false
}

// Allies liefert die Verbündeten eines Herrschers.
func (g *Game) Allies(seat int) []int {
	var out []int
	for _, x := range g.Pacts {
		if x.A == seat {
			out = append(out, x.B)
		} else if x.B == seat {
			out = append(out, x.A)
		}
	}
	return out
}

var (
	errPactSelf     = &RuleError{"pact_bad", "Mit diesem Herrscher geht das nicht."}
	errPactExists   = &RuleError{"pact_exists", "Sie sind bereits verbündet."}
	errPactNoOffer  = &RuleError{"pact_no_offer", "Es liegt kein Angebot vor."}
	errPactNone     = &RuleError{"pact_none", "Sie sind nicht verbündet."}
	errAttackAlly   = &RuleError{"attack_ally", "Sie sind mit diesem Herrscher verbündet. Kündigen Sie zuerst das Abkommen."}
	errJustCanceled = &RuleError{"pact_canceled", "Das Abkommen wurde eben erst gekündigt. Ein Angriff ist frühestens nächstes Jahr möglich."}
)

// proposePact bietet einem anderen Herrscher ein Abkommen an.
func (g *Game) proposePact(seat, target int) error {
	if target < 0 || target >= len(g.Seats) || target == seat || !g.Seats[target].Player.Alive() {
		return errPactSelf
	}
	if g.Allied(seat, target) {
		return errPactExists
	}
	o := Pact{seat, target}
	for _, x := range g.Offers {
		if x == o {
			return nil // schon angeboten
		}
	}
	for _, x := range g.Declined {
		if x == o {
			return &RuleError{"pact_declined", "Dieses Angebot wurde in diesem Jahr bereits abgelehnt."}
		}
	}
	// Gegenseitiges Angebot gilt als Annahme
	for i, x := range g.Offers {
		if x == (Pact{target, seat}) {
			g.Offers = append(g.Offers[:i], g.Offers[i+1:]...)
			g.Pacts = append(g.Pacts, pair(seat, target))
			return nil
		}
	}
	g.Offers = append(g.Offers, o)
	return nil
}

// answerPact nimmt ein Angebot von from an oder lehnt es ab.
func (g *Game) answerPact(seat, from int, accept bool) error {
	for i, x := range g.Offers {
		if x == (Pact{from, seat}) {
			g.Offers = append(g.Offers[:i], g.Offers[i+1:]...)
			if accept && !g.Allied(seat, from) {
				g.Pacts = append(g.Pacts, pair(seat, from))
			} else if !accept {
				g.Declined = append(g.Declined, x) // im selben Jahr nicht erneut anbieten
			}
			return nil
		}
	}
	return errPactNoOffer
}

// cancelPact kündigt ein Abkommen; Angriff auf den Ex-Partner erst im nächsten Jahr.
func (g *Game) cancelPact(seat, target int) error {
	p := pair(seat, target)
	for i, x := range g.Pacts {
		if x == p {
			g.Pacts = append(g.Pacts[:i], g.Pacts[i+1:]...)
			g.Canceled = append(g.Canceled, p)
			return nil
		}
	}
	return errPactNone
}

// warAllowed prüft Bündnisregeln vor einer Kriegserklärung.
func (g *Game) warAllowed(seat, target int) error {
	if g.Allied(seat, target) {
		return errAttackAlly
	}
	p := pair(seat, target)
	for _, x := range g.Canceled {
		if x == p {
			return errJustCanceled
		}
	}
	return nil
}

// OffersTo liefert die Sitze, die dem Herrscher ein Abkommen angeboten haben.
func (g *Game) OffersTo(seat int) []int {
	var out []int
	for _, x := range g.Offers {
		if x.B == seat {
			out = append(out, x.A)
		}
	}
	return out
}

// yearEndPacts entfernt Abkommen und Angebote ausgeschiedener Herrscher und leert die
// Jahreslisten (Angebote, Kündigungen) am Jahresende.
func (g *Game) yearEndPacts() {
	alive := func(i int) bool { return g.Seats[i].Player.Alive() }
	var keep []Pact
	for _, x := range g.Pacts {
		if alive(x.A) && alive(x.B) {
			keep = append(keep, x)
		}
	}
	g.Pacts, g.Offers, g.Canceled, g.Declined = keep, nil, nil, nil
}

// offered meldet ein bereits abgegebenes oder abgelehntes Angebot von a an b (dieses Jahr).
func (g *Game) offered(a, b int) bool {
	for _, x := range append(append([]Pact{}, g.Offers...), g.Declined...) {
		if x == (Pact{a, b}) {
			return true
		}
	}
	return false
}

// strongestRivalSeat: Sitz des stärksten anderen lebenden Herrschers (-1 = keiner).
func (g *Game) strongestRivalSeat(seat int) int {
	best, bestV := -1, 0.0
	for i, s := range g.Seats {
		if i == seat || !s.Player.Alive() {
			continue
		}
		if v := power(s.Player); best < 0 || v > bestV {
			best, bestV = i, v
		}
	}
	return best
}
