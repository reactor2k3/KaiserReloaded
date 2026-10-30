package engine

import "math"

// Finanzen nach dem Vorbild Kaiser II (Markt-Menü „Kredit aufnehmen / zurückzahlen“,
// „Staatsanleihen erwerben / Obligationen verkaufen“, Entwicklungshilfe „Schenkung,
// zinsloser Kredit, normaler Kredit“, Zinsabrechnung, Anleihenauszahlung).
//
// Original:
//   - Bonität KR = (Jahr − 1690) × 1000 + Hektar / 2, Kreditzins 10 %, mit Kredit kein Aufstieg.
//   - Reichsanleihen bringen 18 Taler je Stück und Jahr.
//   - Anleihen anderer Länder: Besitzer bekommen je Stück 1/5000 der Einnahmen des Landes.
//   - Kredite an Mitspieler mit Zins, tilgungsfreier Zeit und Tilgungsfrist.
//
// Remake (nur ohne Classic-Regeln):
//   - Bonität = (Jahr − Startjahr + 10) × 1000 + Hektar / 2, Bankkredit 8 % (günstiger als der
//     Dispo mit 10 %), mit offenem Bankkredit kein Titelaufstieg.
//   - Reichsanleihe: 250 Taler, 18 Taler Zins im Jahr, Rückgabe zum vollen Preis.
//   - Herrscheranleihen: je Stück 1 ‰ der Staatseinnahmen (Steuern, Märkte, Mühlen) des Herrschers, Preis 8 × dieser
//     Ausschüttung (mindestens 50), Rückgabe zu 90 %. Höchstens 100 Stück je Herrscher im Umlauf.
//     Stirbt der Herrscher oder geht ins Exil, sind seine Anleihen wertlos.
//   - Kredit an Mitspieler: 0–15 % Zins, 1–10 Jahre, gleiche Tilgungsraten ab dem nächsten
//     Jahresende; der Schuldner kann jederzeit ganz zurückzahlen. Spenden ohne Gegenleistung.

const (
	LoanRate        = 0.08
	ReichBondPrice  = 250
	ReichBondYield  = 18
	MaxBondsIssuer  = 100
	BondPermille    = 1000 // Ausschüttung je Anleihe = Steuern / 1000
	MaxPeerRate     = 15
	MaxPeerYears    = 10
	bondSellPercent = 90
)

// Bond: Holder besitzt N Anleihen von Issuer.
type Bond struct {
	Holder int `json:"holder"`
	Issuer int `json:"issuer"`
	N      int `json:"n"`
}

// PeerLoan ist ein Kredit zwischen zwei Herrschern; Years = verbleibende Raten.
type PeerLoan struct {
	Lender   int `json:"lender"`
	Borrower int `json:"borrower"`
	Amount   int `json:"amount"`
	Rate     int `json:"rate"`
	Years    int `json:"years"`
}

var (
	errFinClassic  = &RuleError{"fin_classic", "Finanzgeschäfte gibt es nur mit den Remake-Regeln."}
	errBonitaet    = &RuleError{"bonitaet", "So viel leiht Ihnen keine Bank. Ihre Bonität reicht nicht."}
	errNoLoan      = &RuleError{"no_loan", "So hoch ist Ihr Kredit nicht."}
	errNoCash      = &RuleError{"no_cash", "Dafür reicht Ihr Guthaben nicht. Geliehenes Geld kann man nicht weitergeben."}
	errBondsMax    = &RuleError{"bonds_max", "Von diesem Herrscher sind nicht mehr so viele Anleihen zu haben."}
	errBondsNone   = &RuleError{"bonds_none", "So viele Anleihen besitzen Sie nicht."}
	errFinTarget   = &RuleError{"fin_target", "Mit diesem Herrscher geht das nicht."}
	errPeerTerms   = &RuleError{"peer_terms", "Zins 0 bis 15 %, Laufzeit 1 bis 10 Jahre."}
	errPeerOpen    = &RuleError{"peer_open", "Der letzte Kredit an diesen Herrscher ist noch nicht zurückgezahlt."}
	errPeerNone    = &RuleError{"peer_none", "Bei diesem Herrscher haben Sie keine Schulden."}
	errFinBadPhase = &RuleError{"fin_phase", "Finanzgeschäfte gehen nur in der Markt- und der Bauphase."}
)

// Bonitaet ist der höchste Bankkredit, den ein Herrscher insgesamt bekommt.
func (g *Game) Bonitaet(p *Player) int {
	return max(0, 1000*(g.Year-g.Balance.Start.StartYear+10)+p.Land/2)
}

// bondDividend: Ausschüttung je Anleihe aus den Staatseinnahmen des Herrschers. Vor der ersten
// Steuer gilt eine Schätzung aus Steuervorschau und mittleren Markt- und Mühlenerträgen.
func bondDividend(p *Player) int {
	tax := p.LastTax
	if tax <= 0 {
		tax = Taxes(p).Total + p.Markets*190 + p.Mills*375
	}
	return max(0, tax/BondPermille)
}

// BondPrice ist der Kaufpreis einer Herrscheranleihe.
func BondPrice(p *Player) int { return max(50, 8*bondDividend(p)) }

// bondsIssued zählt die Anleihen eines Herrschers im Besitz anderer.
func (g *Game) bondsIssued(issuer int) int {
	n := 0
	for _, b := range g.Bonds {
		if b.Issuer == issuer {
			n += b.N
		}
	}
	return n
}

// bondIndex liefert den Eintrag holder/issuer (-1 = keiner).
func (g *Game) bondIndex(holder, issuer int) int {
	for i, b := range g.Bonds {
		if b.Holder == holder && b.Issuer == issuer {
			return i
		}
	}
	return -1
}

// validPeer prüft einen anderen, lebenden Herrscher als Gegenüber.
func (g *Game) validPeer(seat, target int) bool {
	return target >= 0 && target < len(g.Seats) && target != seat && g.Seats[target].Player.Alive()
}

// spend zieht Geld ab, aber nur aus echtem Guthaben (kein Dispo für Geldgeschäfte).
func spend(p *Player, amount int) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if p.Treasury < amount {
		return errNoCash
	}
	p.Treasury -= amount
	return nil
}

// finance führt ein Finanzgeschäft aus (Markt- und Bauphase). data beschreibt es für die Chronik
// (nil = kein Eintrag).
func (g *Game) finance(seat int, a Action) (data map[string]any, err error) {
	if g.Balance.Classic {
		return nil, errFinClassic
	}
	p := g.Seats[seat].Player
	switch a.Type {
	case "takeLoan":
		if a.Qty <= 0 {
			return nil, ErrInvalidAmount
		}
		if p.Loan+a.Qty > g.Bonitaet(p) {
			return nil, errBonitaet
		}
		p.Loan += a.Qty
		p.Treasury += a.Qty
	case "repayLoan":
		if a.Qty > p.Loan {
			return nil, errNoLoan
		}
		if err := spend(p, a.Qty); err != nil {
			return nil, err
		}
		p.Loan -= a.Qty
	case "buyReichBonds":
		if a.Qty <= 0 {
			return nil, ErrInvalidAmount
		}
		if err := spend(p, a.Qty*ReichBondPrice); err != nil {
			return nil, err
		}
		p.ReichBonds += a.Qty
	case "sellReichBonds":
		if a.Qty <= 0 || a.Qty > p.ReichBonds {
			return nil, errBondsNone
		}
		p.ReichBonds -= a.Qty
		p.Treasury += a.Qty * ReichBondPrice
	case "buyBonds":
		if !g.validPeer(seat, a.Target) {
			return nil, errFinTarget
		}
		if a.Qty <= 0 {
			return nil, ErrInvalidAmount
		}
		if g.bondsIssued(a.Target)+a.Qty > MaxBondsIssuer {
			return nil, errBondsMax
		}
		if err := spend(p, a.Qty*BondPrice(g.Seats[a.Target].Player)); err != nil {
			return nil, err
		}
		if i := g.bondIndex(seat, a.Target); i >= 0 {
			g.Bonds[i].N += a.Qty
		} else {
			g.Bonds = append(g.Bonds, Bond{seat, a.Target, a.Qty})
		}
	case "sellBonds":
		i := g.bondIndex(seat, a.Target)
		if i < 0 || a.Qty <= 0 || a.Qty > g.Bonds[i].N {
			return nil, errBondsNone
		}
		p.Treasury += a.Qty * BondPrice(g.Seats[a.Target].Player) * bondSellPercent / 100
		if g.Bonds[i].N -= a.Qty; g.Bonds[i].N == 0 {
			g.Bonds = append(g.Bonds[:i], g.Bonds[i+1:]...)
		}
	case "gift":
		if !g.validPeer(seat, a.Target) {
			return nil, errFinTarget
		}
		if err := spend(p, a.Qty); err != nil {
			return nil, err
		}
		g.Seats[a.Target].Player.Treasury += a.Qty
		return map[string]any{"kind": "gift", "target": a.Target, "amount": a.Qty}, nil
	case "lend":
		if !g.validPeer(seat, a.Target) {
			return nil, errFinTarget
		}
		if a.Rate < 0 || a.Rate > MaxPeerRate || a.Years < 1 || a.Years > MaxPeerYears {
			return nil, errPeerTerms
		}
		for _, l := range g.Loans {
			if l.Lender == seat && l.Borrower == a.Target {
				return nil, errPeerOpen
			}
		}
		if err := spend(p, a.Qty); err != nil {
			return nil, err
		}
		g.Seats[a.Target].Player.Treasury += a.Qty
		g.Loans = append(g.Loans, PeerLoan{seat, a.Target, a.Qty, a.Rate, a.Years})
		return map[string]any{"kind": "lend", "target": a.Target, "amount": a.Qty, "rate": a.Rate, "years": a.Years}, nil
	case "repayPeer": // Schuldner zahlt den Rest an Target (Gläubiger) auf einmal zurück
		for i, l := range g.Loans {
			if l.Borrower == seat && l.Lender == a.Target {
				if err := spend(p, l.Amount); err != nil {
					return nil, err
				}
				g.Seats[l.Lender].Player.Treasury += l.Amount
				g.Loans = append(g.Loans[:i], g.Loans[i+1:]...)
				return map[string]any{"kind": "repay", "target": a.Target, "amount": l.Amount}, nil
			}
		}
		return nil, errPeerNone
	default:
		return nil, ErrWrongPhase
	}
	return nil, nil
}

// isFinance meldet, ob ein Aktionstyp ein Finanzgeschäft ist.
func isFinance(t string) bool {
	switch t {
	case "takeLoan", "repayLoan", "buyReichBonds", "sellReichBonds", "buyBonds", "sellBonds", "gift", "lend", "repayPeer":
		return true
	}
	return false
}

// payDividends: Nach der Steuer zahlt ein Herrscher die Ausschüttung an die Besitzer seiner Anleihen;
// tax sind die Staatseinnahmen des Jahres (Steuern + Märkte + Mühlen).
// Liefert die Gesamtsumme (0 = keine Anleihen im Umlauf).
func (g *Game) payDividends(issuer int, tax int) int {
	p := g.Seats[issuer].Player
	p.LastTax = tax
	per := max(0, tax/BondPermille)
	total := 0
	for _, b := range g.Bonds {
		if b.Issuer != issuer {
			continue
		}
		x := b.N * per
		g.Seats[b.Holder].Player.Year.Dividends += x
		g.Seats[b.Holder].Player.Treasury += x
		total += x
	}
	p.Treasury -= total
	return total
}

// FinanceReport ist die Jahresabrechnung der Geldgeschäfte eines Herrschers.
type FinanceReport struct {
	LoanInterest int `json:"loanInterest"` // Bankkredit (Aufwand)
	ReichYield   int `json:"reichYield"`   // Reichsanleihen (Ertrag)
	Dividends    int `json:"dividends"`    // Herrscheranleihen (Ertrag, schon bei der Steuer gutgeschrieben)
	PeerIn       int `json:"peerIn"`       // Zins und Tilgung von Schuldnern
	PeerOut      int `json:"peerOut"`      // Zins und Tilgung an Gläubiger
}

// empty meldet einen Bericht ohne Buchungen.
func (r FinanceReport) empty() bool { return r == FinanceReport{} }

// yearEndFinance rechnet Zinsen, Ausschüttungen und Tilgungen ab und räumt Geschäfte mit
// ausgeschiedenen Herrschern ab. Liefert je Sitz den Bericht.
func (g *Game) yearEndFinance() []FinanceReport {
	reps := make([]FinanceReport, len(g.Seats))
	alive := func(i int) bool { return g.Seats[i].Player.Alive() }
	for i, s := range g.Seats {
		p := s.Player
		if !alive(i) {
			continue
		}
		if p.Loan > 0 {
			x := int(math.Ceil(float64(p.Loan) * LoanRate))
			p.Treasury -= x
			reps[i].LoanInterest = x
		}
		if p.ReichBonds > 0 {
			x := p.ReichBonds * ReichBondYield
			p.Treasury += x
			reps[i].ReichYield = x
		}
		reps[i].Dividends = p.Year.Dividends
	}
	// Kredite unter Herrschern: Zins und Rate; mit Ausgeschiedenen verfällt der Kredit.
	var keep []PeerLoan
	for _, l := range g.Loans {
		if !alive(l.Lender) || !alive(l.Borrower) {
			continue
		}
		rate := (l.Amount + l.Years - 1) / l.Years
		x := l.Amount*l.Rate/100 + rate
		g.Seats[l.Borrower].Player.Treasury -= x
		g.Seats[l.Lender].Player.Treasury += x
		reps[l.Borrower].PeerOut += x
		reps[l.Lender].PeerIn += x
		l.Amount -= rate
		l.Years--
		if l.Amount > 0 && l.Years > 0 {
			keep = append(keep, l)
		}
	}
	g.Loans = keep
	// Anleihen ausgeschiedener Herrscher sind wertlos
	var bonds []Bond
	for _, b := range g.Bonds {
		if alive(b.Holder) && alive(b.Issuer) {
			bonds = append(bonds, b)
		}
	}
	g.Bonds = bonds
	return reps
}

// FinanceView ist der Finanzausschnitt der eigenen Sicht.
type FinanceView struct {
	Bonitaet   int        `json:"bonitaet"`
	BondPrice  []int      `json:"bondPrice"`  // Preis je Herrscher (Index = Sitz)
	BondDiv    []int      `json:"bondDiv"`    // Ausschüttung je Anleihe (Vorschau)
	BondFree   []int      `json:"bondFree"`   // noch zu habende Anleihen je Herrscher
	Holdings   []int      `json:"holdings"`   // eigene Anleihen je Herrscher
	IssuedOwn  int        `json:"issuedOwn"`  // eigene Anleihen im Besitz anderer
	DividendPA int        `json:"dividendPA"` // Ausschüttung je eigener Anleihe (Vorschau)
	Loans      []PeerLoan `json:"loans"`      // Kredite, an denen man beteiligt ist
}

// financeView baut den Finanzausschnitt für einen Sitz.
func (g *Game) financeView(seat int) FinanceView {
	p := g.Seats[seat].Player
	v := FinanceView{Bonitaet: g.Bonitaet(p), IssuedOwn: g.bondsIssued(seat), DividendPA: bondDividend(p)}
	for i, s := range g.Seats {
		v.BondPrice = append(v.BondPrice, BondPrice(s.Player))
		v.BondDiv = append(v.BondDiv, bondDividend(s.Player))
		v.BondFree = append(v.BondFree, MaxBondsIssuer-g.bondsIssued(i))
		n := 0
		if j := g.bondIndex(seat, i); j >= 0 {
			n = g.Bonds[j].N
		}
		v.Holdings = append(v.Holdings, n)
	}
	for _, l := range g.Loans {
		if l.Lender == seat || l.Borrower == seat {
			v.Loans = append(v.Loans, l)
		}
	}
	return v
}
