package engine

import "testing"

// TestColonies prüft Werft, Kolonialflotten, Landerwerb, Erträge und Seekrieg.
func TestColonies(t *testing.T) {
	b := DefaultBalance()
	g := NewGame(DefaultSettings(b), b, 5)
	w, _ := LoadWorld("../../../data/world.json")
	g.World = w
	g.AddPlayer("A", 5, false)
	g.AddPlayer("B", 4, false)
	g.Start(0)
	a, d := g.Seats[0].Player, g.Seats[1].Player
	for i, p := range []*Player{a, d} {
		g.Seats[i].Phase = PhaseBuild
		p.Treasury, p.Population = 200000, 4000
	}
	do := func(seat int, act Action) error { _, err := g.Apply(seat, act, 0); return err }

	// Werft: höchstens Einwohner / 200 Schiffe im Jahr
	if err := do(0, Action{Type: "buildShip", Unit: "trader", Qty: 21}); err == nil {
		t.Error("Werftgrenze überschritten")
	}
	if err := do(0, Action{Type: "buildShip", Unit: "warship", Qty: 10}); err != nil || a.Warships != 10 {
		t.Fatalf("Kriegsschiffe: %v", err)
	}
	if err := do(1, Action{Type: "buildShip", Unit: "trader", Qty: 4}); err != nil {
		t.Fatal(err)
	}
	if err := do(1, Action{Type: "buildShip", Unit: "warship", Qty: 2}); err != nil {
		t.Fatal(err)
	}
	// Kolonie in Amerika: ohne Händler kein Land, Kriegsschiffe nur mit Händlern
	if err := do(1, Action{Type: "colonize", Continent: 1, Qty: 4}); err == nil {
		t.Error("Kolonie ohne Handelsschiffe")
	}
	if err := do(1, Action{Type: "fleet", Continent: 1, Qty: 0, Warships: 2}); err == nil {
		t.Error("Kriegsflotte ohne Händler")
	}
	if err := do(1, Action{Type: "fleet", Continent: 1, Qty: 4, Warships: 2}); err != nil {
		t.Fatal(err)
	}
	if err := do(1, Action{Type: "colonize", Continent: 1, Qty: 20}); err != nil || d.Colonies[1].Area != 20000 {
		t.Fatalf("Landerwerb: %v", err)
	}
	// Ertrag: 4 Händler bedienen 16 000 km² -> 8000 × 0,7–1,3; Unterhalt 4×400 + 2×200
	r := colonyIncome(d, g.Rng)
	if r.Income[1] < 5600 || r.Income[1] > 10400 || r.Upkeep != 2000 {
		t.Errorf("Kolonialertrag %+v", r)
	}

	// Seekrieg: 10 gegen 2 Kriegsschiffe, Verteidiger flieht, Kolonialflotte heim
	if err := do(0, Action{Type: "navalAttack", Target: 1, Continent: 0, Warships: 5}); err == nil {
		t.Error("Angriff auf leere Kolonie")
	}
	ev, err := g.Apply(0, Action{Type: "navalAttack", Target: 1, Continent: 1, Warships: 10}, 0)
	if err != nil {
		t.Fatal(err)
	}
	rep := ev[0].Data.(*NavalReport)
	if !rep.Won || d.Colonies[1].Traders != 0 || d.Traders != 4-rep.TradersHit || a.Warships != 10-rep.AttLost {
		t.Errorf("Seeschlacht: %+v, Verteidiger %+v", rep, d.Colonies[1])
	}
	if err := do(0, Action{Type: "navalAttack", Target: 1, Continent: 1, Warships: 1}); err == nil {
		t.Error("zweiter Angriff im selben Jahr")
	}

	// Schiffsereignisse nur mit passenden Schiffen
	c := NewGame(DefaultSettings(b), b, 1)
	c.AddPlayer("C", 1, false)
	c.Start(0)
	for _, ev := range Events {
		if (ev.Kind == EvShipPlus || ev.Kind == EvShipStorm || ev.Kind == EvFleetLoss) && c.eventFits(0, ev) {
			t.Errorf("Schiffsereignis %s ohne Schiffe", ev.ID)
		}
	}
	a.Traders, a.Colonies[0] = 3, Colony{Area: 4000, Traders: 3, Warships: 2}
	g.applyEvent(0, EventDef{"piraten", EvFleetLoss, "", ""})
	if fleetSize(a) != 0 || a.Traders < 0 || a.Warships < 0 {
		t.Errorf("Piraten: %+v", a.Colonies)
	}
}

// TestLastRulerPlaysOn: Stirbt der Computer-Herrscher, spielt der Mensch weiter bis zur Krone;
// stirbt der letzte Mensch, ist das Spiel aus.
func TestLastRulerPlaysOn(t *testing.T) {
	b := DefaultBalance()
	g := NewGame(DefaultSettings(b), b, 2)
	g.AddPlayer("Mensch", 5, false)
	g.AddBot("kaufmann")
	g.Start(0)
	g.Seats[1].Player.DeathYear = 0
	g.endYear(0)
	if g.Over {
		t.Fatal("Spiel endet, obwohl der Mensch lebt")
	}
	g.Seats[0].Player.DeathYear = 0
	g.endYear(0)
	if !g.Over {
		t.Error("Spiel läuft ohne lebenden Menschen weiter")
	}
}
