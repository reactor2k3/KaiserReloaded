package engine

import (
	"errors"
	"testing"
)

// playYear spielt für jeden Sitz ein Jahr mit Standardentscheidungen durch.
func playYear(t *testing.T, g *Game, now int64) []Event {
	t.Helper()
	var all []Event
	for i, s := range g.Seats {
		if s.Phase == PhaseDone {
			continue
		}
		steps := []Action{
			{Type: "next"},
			{Type: "feed", Qty: max(FeedLimitsLo(s.Player), min(FeedLimitsHi(s.Player), s.Player.Year.GrainNeed))},
			{Type: "next"},
			{Type: "endTurn"},
		}
		for _, a := range steps {
			if g.Seats[i].Phase == PhaseDone {
				break
			}
			if a.Type == "feed" { // Menge erst jetzt berechnen: Ereignisse können die Reserve verändert haben
				a.Qty = max(FeedLimitsLo(s.Player), min(FeedLimitsHi(s.Player), s.Player.Year.GrainNeed))
			}
			ev, err := g.Apply(i, a, now)
			if err != nil {
				t.Fatalf("Sitz %d, %s in Phase %s: %v", i, a.Type, g.Seats[i].Phase, err)
			}
			all = append(all, ev...)
		}
	}
	return all
}

func FeedLimitsLo(p *Player) int { lo, _ := FeedLimits(p); return lo }
func FeedLimitsHi(p *Player) int { _, hi := FeedLimits(p); return hi }

func TestGameYearCycle(t *testing.T) {
	b := DefaultBalance()
	g := NewGame(DefaultSettings(b), b, 1)
	if _, err := g.AddPlayer("Micha", 5, false); err != nil {
		t.Fatal(err)
	}
	if _, err := g.AddPlayer("Lena", 0, true); err == nil {
		t.Fatal("ungültiges Land erlaubt")
	}
	if _, err := g.AddPlayer("Lena", 2, true); err != nil {
		t.Fatal(err)
	}
	g.Start(1000)
	if g.Seats[0].Phase != PhaseMarket || g.Seats[1].Phase != PhaseMarket {
		t.Fatal("nach Start nicht in der Marktphase")
	}
	if _, err := g.Apply(0, Action{Type: "endTurn"}, 1000); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("Zug beenden in Marktphase: %v", err)
	}
	ev := playYear(t, g, 1010)
	if g.Year != 1701 {
		t.Fatalf("Jahr nicht weitergezählt: %d", g.Year)
	}
	newYear := false
	for _, e := range ev {
		if e.Type == "newYear" {
			newYear = true
		}
	}
	if !newYear || g.Seats[0].Phase != PhaseMarket {
		t.Error("kein neues Jahr gestartet")
	}
	if v := g.Self(0); v.Player.Year.TaxFactors != (TaxFactors{}) {
		t.Error("verdeckte Steuerfaktoren gehen an den Client")
	}
}

func TestGameTimeouts(t *testing.T) {
	b := DefaultBalance()
	s := DefaultSettings(b)
	s.PhaseSeconds, s.TimePenalty = 60, false
	g := NewGame(s, b, 2)
	g.AddPlayer("Micha", 5, false)
	g.AddPlayer("Anna", 3, true)
	g.Start(0)
	// Ohne Strafe: nach 60 s wird automatisch weitergeschaltet, viermal bis zum Jahresende.
	now := int64(0)
	for i := 0; i < 4 && g.Year == 1700; i++ {
		now += 61
		g.Timeouts(now)
	}
	if g.Year != 1701 {
		t.Fatalf("Zeitlimit schaltet nicht durch, Jahr %d, Phase %s", g.Year, g.Seats[0].Phase)
	}
}

func TestGameRunsManyYears(t *testing.T) {
	// Rauchtest: 60 Jahre mit Standardzügen dürfen weder hängen noch abstürzen.
	b := DefaultBalance()
	g := NewGame(DefaultSettings(b), b, 7)
	g.AddPlayer("A", 1, false)
	g.AddPlayer("B", 4, true)
	g.Start(0)
	for y := 0; y < 60 && !g.Over; y++ {
		playYear(t, g, int64(y))
	}
	if !g.Over && g.Year < 1760 {
		t.Errorf("nach 60 Runden erst Jahr %d", g.Year)
	}
	if _, err := g.Snapshot(); err != nil {
		t.Fatal(err)
	}
}

// Geteilte Länder: bis zu drei Herrscher je Land, Landkauf nur aus freier Fläche,
// Krieg im selben Land ohne Durchmarsch.
func TestSharedCountry(t *testing.T) {
	g := NewGame(DefaultSettings(DefaultBalance()), DefaultBalance(), 7)
	w, err := LoadWorld("../../../data/world.json")
	if err != nil {
		t.Fatal(err)
	}
	g.World = w
	for i := 0; i < 3; i++ {
		if _, err := g.AddPlayer("Herr", 5, false); err != nil {
			t.Fatalf("Herrscher %d in Sachsen abgelehnt: %v", i, err)
		}
	}
	if _, err := g.AddPlayer("Vierter", 5, false); err == nil {
		t.Fatal("vierter Herrscher im selben Land angenommen")
	}
	if free := g.FreeLand(5); free != g.Balance.CountryArea-3*g.Balance.Start.Land {
		t.Errorf("freie Fläche = %d", free)
	}
	g.Seats[0].Player.Land = g.Balance.CountryArea - 2*g.Balance.Start.Land - 100
	if free := g.FreeLand(5); free != 100 {
		t.Fatalf("freie Fläche nach Landgewinn = %d, erwartet 100", free)
	}
	g.Start(0)
	// Ereignisse beim Jahresstart können Land verändert haben: Ausgangslage neu setzen
	for i := 1; i < 3; i++ {
		g.Seats[i].Player.Land = g.Balance.Start.Land
	}
	g.Seats[0].Player.Land = g.Balance.CountryArea - 2*g.Balance.Start.Land - 100
	g.Seats[0].Player.Treasury = 1_000_000
	if _, err := g.Apply(0, Action{Type: "buyLand", Qty: 500}, 0); err == nil {
		t.Error("Landkauf über die freie Fläche hinaus erlaubt")
	}
	if _, err := g.Apply(0, Action{Type: "buyLand", Qty: 100}, 0); err != nil {
		t.Errorf("Landkauf im Rahmen der freien Fläche abgelehnt: %v", err)
	}
	war := &War{Attacker: 0, Defender: 1, Stances: map[int]string{}}
	if r := g.findRoute(war); len(r) != 1 || r[0] != 5 {
		t.Errorf("Krieg im selben Land braucht keinen Weg, bekommen %v", r)
	}

	classic := NewGame(DefaultSettings(ClassicBalance()), ClassicBalance(), 7)
	classic.AddPlayer("A", 5, false)
	if _, err := classic.AddPlayer("B", 5, false); err == nil {
		t.Error("Original-Regeln: zweiter Herrscher im selben Land angenommen")
	}
}
