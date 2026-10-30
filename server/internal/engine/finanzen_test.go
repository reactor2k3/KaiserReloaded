package engine

import "testing"

// TestFinance prüft Bankkredit, Anleihen, Kredite unter Herrschern und die Jahresabrechnung.
func TestFinance(t *testing.T) {
	b := DefaultBalance()
	g := NewGame(DefaultSettings(b), b, 3)
	g.AddPlayer("A", 5, false)
	g.AddPlayer("B", 4, false)
	g.Start(0)
	for i := range g.Seats {
		g.Seats[i].Phase = PhaseMarket
	}
	a, bp := g.Seats[0].Player, g.Seats[1].Player
	do := func(seat int, act Action) error { _, err := g.Apply(seat, act, 0); return err }

	// Bankkredit bis zur Bonität, mit Kredit kein Aufstieg
	limit := g.Bonitaet(a)
	if err := do(0, Action{Type: "takeLoan", Qty: limit + 1}); err == nil {
		t.Error("Kredit über Bonität erlaubt")
	}
	cash := a.Treasury
	if err := do(0, Action{Type: "takeLoan", Qty: 10000}); err != nil || a.Loan != 10000 || a.Treasury != cash+10000 {
		t.Fatalf("Kredit: %v %+v", err, a.Loan)
	}
	if canPromote(a) {
		t.Error("Aufstieg trotz Bankkredit")
	}

	// Reichsanleihen und Anleihen von B
	if err := do(0, Action{Type: "buyReichBonds", Qty: 10}); err != nil || a.ReichBonds != 10 {
		t.Fatalf("Reichsanleihen: %v", err)
	}
	if err := do(0, Action{Type: "buyBonds", Target: 1, Qty: MaxBondsIssuer + 1}); err == nil {
		t.Error("mehr Anleihen als im Umlauf erlaubt")
	}
	if err := do(0, Action{Type: "buyBonds", Target: 0, Qty: 1}); err == nil {
		t.Error("eigene Anleihen kaufbar")
	}
	if err := do(0, Action{Type: "buyBonds", Target: 1, Qty: 20}); err != nil {
		t.Fatal(err)
	}
	before := a.Treasury
	if paid := g.payDividends(1, 30000); paid != 20*30 || a.Treasury != before+600 || a.Year.Dividends != 600 {
		t.Errorf("Ausschüttung %d, Kasse %d", paid, a.Treasury-before)
	}

	// Kredit an B: 10 % Zins, 2 Jahre
	g.Seats[0].Phase, g.Seats[1].Phase = PhaseBuild, PhaseBuild
	if err := do(0, Action{Type: "lend", Target: 1, Qty: 4000, Rate: 20, Years: 2}); err == nil {
		t.Error("Wucherzins erlaubt")
	}
	bCash := bp.Treasury
	if err := do(0, Action{Type: "lend", Target: 1, Qty: 4000, Rate: 10, Years: 2}); err != nil || bp.Treasury != bCash+4000 {
		t.Fatalf("Kredit an B: %v", err)
	}
	if err := do(0, Action{Type: "lend", Target: 1, Qty: 100, Rate: 0, Years: 1}); err == nil {
		t.Error("zweiter Kredit vor Rückzahlung erlaubt")
	}

	// Jahresabrechnung: Bankzins 800, Reichsanleihen 180, B zahlt 400 Zins + 2000 Rate
	aCash, bCash := a.Treasury, bp.Treasury
	reps := g.yearEndFinance()
	if reps[0].LoanInterest != 800 || reps[0].ReichYield != 180 || reps[0].PeerIn != 2400 || reps[1].PeerOut != 2400 {
		t.Errorf("Abrechnung: %+v %+v", reps[0], reps[1])
	}
	if a.Treasury != aCash-800+180+2400 || bp.Treasury != bCash-2400 {
		t.Errorf("Kassen nach Abrechnung: %d %d", a.Treasury-aCash, bp.Treasury-bCash)
	}
	if len(g.Loans) != 1 || g.Loans[0].Amount != 2000 || g.Loans[0].Years != 1 {
		t.Fatalf("Restkredit: %+v", g.Loans)
	}
	// B zahlt den Rest vorzeitig zurück
	if err := do(1, Action{Type: "repayPeer", Target: 0}); err != nil || len(g.Loans) != 0 {
		t.Fatalf("Rückzahlung: %v", err)
	}
	// Bankkredit tilgen, Spende
	if err := do(0, Action{Type: "repayLoan", Qty: 10000}); err != nil || a.Loan != 0 {
		t.Fatalf("Tilgung: %v", err)
	}
	if err := do(0, Action{Type: "gift", Target: 1, Qty: a.Treasury + 1}); err == nil {
		t.Error("Spende über Guthaben erlaubt")
	}

	// Classic-Regeln kennen keine Finanzgeschäfte
	cb := ClassicBalance()
	cg := NewGame(DefaultSettings(cb), cb, 1)
	cg.AddPlayer("C", 1, false)
	cg.Start(0)
	if _, err := cg.Apply(0, Action{Type: "takeLoan", Qty: 100}, 0); err == nil {
		t.Error("Classic erlaubt Kredit")
	}
}
