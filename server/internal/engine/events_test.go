package engine

import "testing"

// Ereignisse: feste Ereignisse wirken sofort, Risiko-Entscheidungen warten auf Antwort,
// „next“ in der Marktphase beantwortet offene Entscheidungen mit „Nein“.
func TestEventsAndRisks(t *testing.T) {
	b := DefaultBalance()
	g := NewGame(DefaultSettings(b), b, 3)
	g.AddPlayer("Micha", 5, false)
	g.Start(0)
	p := g.Seats[0].Player

	seen := map[string]bool{}
	for i := 0; i < 400; i++ {
		before := *p
		rep, risk := g.RollEvent(0)
		switch {
		case rep != nil:
			seen["event"] = true
			if rep.Taler != p.Treasury-before.Treasury {
				t.Fatalf("%s: Taler %d, Kasse änderte sich um %d", rep.ID, rep.Taler, p.Treasury-before.Treasury)
			}
		case risk != "":
			seen["risk"] = true
			if RiskByID(risk) == nil {
				t.Fatalf("unbekanntes Risiko %q", risk)
			}
		default:
			seen["none"] = true
		}
	}
	if !seen["event"] || !seen["risk"] || !seen["none"] {
		t.Fatalf("Verteilung unvollständig: %v", seen)
	}

	p.Year.Risk = "erbe"
	if _, err := g.Apply(0, Action{Type: "risk", Stance: "ja"}, 0); err != nil {
		t.Fatal(err)
	}
	if p.Year.Risk != "" {
		t.Error("Risiko nach Antwort noch offen")
	}
	if _, err := g.Apply(0, Action{Type: "risk", Stance: "ja"}, 0); err == nil {
		t.Error("zweite Antwort ohne offenes Risiko angenommen")
	}
	p.Year.Risk = "spanien"
	if _, err := g.Apply(0, Action{Type: "next"}, 0); err != nil {
		t.Fatal(err)
	}
	if p.Year.Risk != "" {
		t.Error("„next“ hat die offene Entscheidung nicht beantwortet")
	}

	classic := NewGame(DefaultSettings(ClassicBalance()), ClassicBalance(), 3)
	classic.AddPlayer("A", 5, false)
	classic.Start(0)
	for i := 0; i < 50; i++ {
		if rep, risk := classic.RollEvent(0); rep != nil || risk != "" {
			t.Fatal("Original-Regeln dürfen keine Kaiser-II-Ereignisse haben")
		}
	}
}

// Innenpolitik: Volksfest, Opposition (einmal im Jahr) und Gendarmen mit Obergrenze.
func TestInnenpolitik(t *testing.T) {
	b := DefaultBalance()
	g := NewGame(DefaultSettings(b), b, 5)
	g.AddPlayer("Micha", 5, false)
	g.Start(0)
	p := g.Seats[0].Player
	p.Treasury = 100000
	g.Seats[0].Phase = PhaseTaxes
	m := p.Mood
	if _, err := g.Apply(0, Action{Type: "fest"}, 0); err != nil || p.Mood != m+1 {
		t.Fatalf("Volksfest: %v, Stimmung %d", err, p.Mood)
	}
	if _, err := g.Apply(0, Action{Type: "opposition"}, 0); err != nil || p.Mood != min(10, m+4) {
		t.Fatalf("Opposition: %v, Stimmung %d", err, p.Mood)
	}
	if _, err := g.Apply(0, Action{Type: "opposition"}, 0); err == nil {
		t.Error("Opposition zweimal im Jahr verhaftet")
	}
	max := p.Population / GendarmsPer
	if _, err := g.Apply(0, Action{Type: "gendarm", Qty: max + 1}, 0); err == nil {
		t.Error("mehr Gendarmen als erlaubt")
	}
	if _, err := g.Apply(0, Action{Type: "gendarm", Qty: max}, 0); err != nil || p.Gendarmes != max {
		t.Fatalf("Gendarmen: %v, %d", err, p.Gendarmes)
	}
	before := p.Mood
	p.Mood = 0
	if MoodFactor(p) >= 1 {
		t.Error("schlechte Stimmung muss Steuern senken")
	}
	p.Mood = before
}

// Heer: Ausbildung/Material steigern, Grenze 10, Rekruten verdünnen, Schlagkraft-Faktor.
func TestDrill(t *testing.T) {
	b := DefaultBalance()
	p := NewPlayer("A", 5, false, b, NewRng(1))
	p.Treasury = 1_000_000
	for i := 0; i < MaxDrill; i++ {
		if err := Train(p, b); err != nil {
			t.Fatal(err)
		}
		if err := Equip(p, b); err != nil {
			t.Fatal(err)
		}
	}
	if Train(p, b) == nil || Equip(p, b) == nil {
		t.Error("über Stufe 10 gesteigert")
	}
	if s := Strength(p); s < 1.79 || s > 1.81 {
		t.Errorf("Schlagkraft bei 10/10 = %.2f, erwartet 1,8", s)
	}
	dilute(p, 20, 40)
	if p.Training != 5 || p.Equipment != 5 {
		t.Errorf("Verdünnung: %d/%d, erwartet 5/5", p.Training, p.Equipment)
	}
}

// Bündnisse: Angebot, Annahme, kein Angriff auf Verbündete, Kündigung sperrt das Jahr,
// Verbündete helfen dem Angegriffenen automatisch.
func TestPacts(t *testing.T) {
	b := DefaultBalance()
	b.Start.Title = 2
	g := NewGame(DefaultSettings(b), b, 9)
	w, _ := LoadWorld("../../../data/world.json")
	g.World = w
	g.AddPlayer("A", 5, false)
	g.AddPlayer("B", 4, false)
	g.AddPlayer("C", 1, false)
	g.Start(0)
	for i := range g.Seats {
		g.Seats[i].Phase = PhaseBuild
	}
	if _, err := g.Apply(0, Action{Type: "proposePact", Target: 1}, 0); err != nil {
		t.Fatal(err)
	}
	if offers := g.OffersTo(1); len(offers) != 1 || offers[0] != 0 {
		t.Fatalf("Angebot fehlt: %v", offers)
	}
	if _, err := g.Apply(1, Action{Type: "acceptPact", Target: 0}, 0); err != nil || !g.Allied(0, 1) {
		t.Fatalf("Annahme: %v", err)
	}
	if _, err := g.Apply(0, Action{Type: "declareWar", Target: 1}, 0); err == nil {
		t.Error("Angriff auf Verbündeten erlaubt")
	}
	// C greift B an: A steht B automatisch bei
	if _, err := g.Apply(2, Action{Type: "declareWar", Target: 1}, 0); err != nil {
		t.Fatal(err)
	}
	war := g.Wars[0]
	g.advanceWar(war, 0)
	if war.Stances[0] != StanceHelpDefender || g.Seats[0].Phase == PhaseDiplomacy {
		t.Errorf("Verbündeter hilft nicht automatisch: Haltung %q, Phase %s", war.Stances[0], g.Seats[0].Phase)
	}
	if _, err := g.Apply(0, Action{Type: "cancelPact", Target: 1}, 0); err != nil || g.Allied(0, 1) {
		t.Fatalf("Kündigung: %v", err)
	}
	if err := g.warAllowed(0, 1); err == nil {
		t.Error("Angriff direkt nach Kündigung erlaubt")
	}
	g.yearEndPacts()
	if err := g.warAllowed(0, 1); err != nil {
		t.Errorf("im Folgejahr gesperrt: %v", err)
	}
}
