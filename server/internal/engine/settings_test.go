package engine

import "testing"

func TestSettings(t *testing.T) {
	s := DefaultSettings(DefaultBalance())
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := s.Overtime(150); got != 60 {
		t.Errorf("Überzeit bei 90-s-Limit nach 150 s = %d", got)
	}
	s.TimePenalty = false
	if s.Overtime(500) != 0 {
		t.Error("ohne Strafe darf keine Überzeit zählen")
	}
	s.PhaseSeconds = 45
	if s.Validate() == nil {
		t.Error("45 s ist keine erlaubte Zugzeit")
	}
	unlimited := Settings{PhaseSeconds: 0, TimePenalty: true, BattleMode: "quick"}
	if unlimited.Validate() != nil || unlimited.Overtime(10_000) != 0 {
		t.Error("unbegrenzte Zugzeit falsch behandelt")
	}
}

// TestMortality: Unsterbliche sterben nie eines natürlichen Todes, „niedrig“ verlängert das Leben.
func TestMortality(t *testing.T) {
	b := DefaultBalance()
	s := DefaultSettings(b)
	s.Mortality = "unsterblich"
	g := NewGame(s, b, 1)
	g.AddPlayer("A", 1, false)
	p := g.Seats[0].Player
	for y := 1700; y < 1900; y++ {
		if EndYear(p, b, g.Rng, y, 0, 0, 0).Died {
			t.Fatalf("Unsterblicher stirbt %d", y)
		}
	}
	s.Mortality = "niedrig"
	g = NewGame(s, b, 1)
	g.AddPlayer("B", 1, false)
	if dy := g.Seats[0].Player.DeathYear; dy < b.Start.DeathYearMin+15 {
		t.Errorf("Lebenserwartung bei niedriger Sterblichkeit: %d", dy)
	}
	s.Mortality = "ewig"
	if s.Validate() == nil {
		t.Error("ungültige Sterblichkeit angenommen")
	}
}

// TestAsyncMode: lange Zugzeiten nur im asynchronen Modus, dort ohne Lebenszeit-Strafe.
func TestAsyncMode(t *testing.T) {
	s := DefaultSettings(DefaultBalance())
	s.Mode, s.PhaseSeconds, s.TimePenalty = "async", 86400, false
	if err := s.Validate(); err != nil || !s.Async() {
		t.Fatalf("asynchron 24 h: %v", err)
	}
	s.TimePenalty = true
	if s.Validate() == nil {
		t.Error("Lebenszeit-Strafe im asynchronen Spiel erlaubt")
	}
	s.TimePenalty, s.PhaseSeconds = false, 90
	if s.Validate() == nil {
		t.Error("90 s im asynchronen Spiel erlaubt")
	}
	s.Mode, s.PhaseSeconds = "live", 86400
	if s.Validate() == nil {
		t.Error("24 h im Live-Spiel erlaubt")
	}
}
