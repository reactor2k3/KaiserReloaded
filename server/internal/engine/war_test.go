package engine

import (
	"errors"
	"path/filepath"
	"testing"
)

// warGame baut ein Spiel mit Welt und lässt alle bis zur Bauphase spielen.
func warGame(t *testing.T, mode string, players ...int) *Game {
	t.Helper()
	b := DefaultBalance()
	s := DefaultSettings(b)
	s.BattleMode = mode
	g := NewGame(s, b, 77)
	w, err := LoadWorld(filepath.Join(dataDir, "world.json"))
	if err != nil {
		t.Fatal(err)
	}
	g.World = w
	for i, c := range players {
		g.AddPlayer(string(rune('A'+i)), c, false)
	}
	g.Start(0)
	for i := range g.Seats {
		p := g.Seats[i].Player
		p.Title, p.Cavalry, p.Infantry, p.Markets = 3, 3, 5, 4
		p.RecalcArmy()
		for _, a := range []Action{{Type: "next"}, {Type: "feed", Qty: FeedLimitsLo(p)}, {Type: "next"}} {
			if _, err := g.Apply(i, a, 1); err != nil {
				t.Fatalf("Sitz %d %s: %v", i, a.Type, err)
			}
		}
	}
	return g
}

func findEvent(ev []Event, typ string) *Event {
	for i := range ev {
		if ev[i].Type == typ {
			return &ev[i]
		}
	}
	return nil
}

func TestDeclareWarRules(t *testing.T) {
	g := warGame(t, "quick", 5, 3)
	g.Seats[0].Player.Title = 1
	if _, err := g.Apply(0, Action{Type: "declareWar", Target: 1}, 2); !errors.Is(err, ErrTooEarly) {
		t.Errorf("Herr darf keinen Krieg erklären: %v", err)
	}
	g.Seats[0].Player.Title = 2
	if _, err := g.Apply(0, Action{Type: "declareWar", Target: 0}, 2); !errors.Is(err, ErrBadTarget) {
		t.Errorf("Krieg gegen sich selbst: %v", err)
	}
}

func TestQuickWarBetweenNeighbours(t *testing.T) {
	g := warGame(t, "quick", 5, 3) // Sachsen gegen Bayern, direkte Nachbarn
	landBefore := g.Seats[0].Player.Land + g.Seats[1].Player.Land
	if _, err := g.Apply(0, Action{Type: "declareWar", Target: 1}, 2); err != nil {
		t.Fatal(err)
	}
	ev, err := g.Apply(1, Action{Type: "endTurn"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	be := findEvent(ev, "battle")
	if be == nil {
		t.Fatalf("keine Schlacht, Ereignisse: %+v", ev)
	}
	res := be.Data.(map[string]any)["result"].(*WarResult)
	if len(g.Wars) != 0 || g.Year != 1701 {
		t.Errorf("nach der Schlacht kein Jahreswechsel: Jahr %d, Kriege %d", g.Year, len(g.Wars))
	}
	if sum := g.Seats[0].Player.Land + g.Seats[1].Player.Land; sum != landBefore {
		t.Errorf("Land nicht erhalten: vorher %d, nachher %d (Ergebnis %+v)", landBefore, sum, res)
	}
	if res.Losses[0].Cavalry+res.Losses[0].Infantry+res.Losses[1].Cavalry+res.Losses[1].Infantry == 0 {
		t.Errorf("Schlacht ohne Verluste: %+v", res)
	}
}

func TestTacticalWarWithDiplomacy(t *testing.T) {
	g := warGame(t, "tactical", 5, 3, 1) // Sachsen greift Bayern an, Preußen entscheidet
	g.Apply(0, Action{Type: "declareWar", Target: 1}, 2)
	g.Apply(1, Action{Type: "endTurn"}, 2)
	ev, _ := g.Apply(2, Action{Type: "endTurn"}, 2)
	if g.Seats[2].Phase != PhaseDiplomacy || findEvent(ev, "warDeclared") == nil {
		t.Fatalf("Preußen nicht in der Diplomatie: %s", g.Seats[2].Phase)
	}
	if v := g.Self(2); v.War == nil || v.War.Field != nil {
		t.Error("Unbeteiligter sieht kein Kriegsbild oder das Schlachtfeld")
	}
	if _, err := g.Apply(2, Action{Type: "stance", Stance: StanceHelpAttacker}, 3); err != nil {
		t.Fatal(err)
	}
	if g.Seats[0].Phase != PhaseDeploy || g.Seats[1].Phase != PhaseDeploy {
		t.Fatalf("keine Aufstellungsphase: %s/%s", g.Seats[0].Phase, g.Seats[1].Phase)
	}
	if _, ok := g.current().Helpers[2]; !ok {
		t.Error("Preußen (Nachbar von Bayern) sollte helfen können")
	}
	v := g.Self(0)
	if v.War == nil || v.War.Field == nil || v.War.Counts["cavalry"] != 6 {
		t.Fatalf("Aufstellungssicht falsch: %+v", v.War)
	}
	// Ungültig: Feld auf gegnerischer Seite
	f := v.War.Field
	bad := Placement{"cavalry", 0, f.Border[0] + 3}
	if _, err := g.Apply(0, Action{Type: "deploy", Placements: []Placement{bad}}, 4); !errors.Is(err, ErrBadDeploy) {
		t.Errorf("Aufstellung im Feindesland erlaubt: %v", err)
	}
	var good []Placement
	for r := 0; r < f.Rows && len(good) < 2; r++ {
		if f.validPlacement(0, r, f.Border[r]-3) {
			good = append(good, Placement{"cavalry", r, f.Border[r] - 3})
		}
	}
	if _, err := g.Apply(0, Action{Type: "deploy", Placements: good}, 4); err != nil {
		t.Fatal(err)
	}
	ev, err := g.Apply(1, Action{Type: "autoDeploy"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if findEvent(ev, "battle") == nil || g.Year != 1701 {
		t.Fatalf("keine Schlacht nach Aufstellung, Jahr %d", g.Year)
	}
}

func TestWarTimeoutsAutoResolve(t *testing.T) {
	g := warGame(t, "tactical", 5, 3, 1)
	g.Settings.PhaseSeconds, g.Settings.TimePenalty = 60, false
	g.Apply(0, Action{Type: "declareWar", Target: 1}, 2)
	g.Apply(1, Action{Type: "endTurn"}, 2)
	g.Apply(2, Action{Type: "endTurn"}, 2)
	now := int64(2)
	for i := 0; i < 4 && g.Year == 1700; i++ {
		now += 61
		g.Timeouts(now)
	}
	if g.Year != 1701 {
		t.Fatalf("Zeitlimit löst Krieg nicht auf: Phasen %s/%s/%s", g.Seats[0].Phase, g.Seats[1].Phase, g.Seats[2].Phase)
	}
}

func TestBlockedRouteClassic(t *testing.T) {
	// Flandern → Bayern führt nur über andere Länder; im Original sind unbesetzte Länder gesperrt.
	g := warGame(t, "quick", 9, 3)
	g.Balance.Classic = true
	g.Apply(0, Action{Type: "declareWar", Target: 1}, 2)
	ev, _ := g.Apply(1, Action{Type: "endTurn"}, 2)
	if findEvent(ev, "warBlocked") == nil {
		t.Error("Original: Weg über unbesetzte Länder müsste versperrt sein")
	}
	g2 := warGame(t, "quick", 9, 3)
	g2.Apply(0, Action{Type: "declareWar", Target: 1}, 2)
	ev, _ = g2.Apply(1, Action{Type: "endTurn"}, 2)
	if findEvent(ev, "battle") == nil {
		t.Error("Remake: unbesetzte Länder sind passierbar, Schlacht erwartet")
	}
}
