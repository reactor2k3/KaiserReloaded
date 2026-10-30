package engine

import (
	"path/filepath"
	"testing"
)

// Zwei Computer-Herrscher spielen nur über BotAction/BotFallback eine Partie durch.
func TestBotsPlayWholeGame(t *testing.T) {
	b := DefaultBalance()
	s := DefaultSettings(b)
	s.BattleMode = "tactical" // auch die Aufstellung muss der Bot erledigen
	s.Difficulty = "schwer"   // volle Spielstärke: der Feldherr muss Krieg führen
	g := NewGame(s, b, 42)
	w, err := LoadWorld(filepath.Join(dataDir, "world.json"))
	if err != nil {
		t.Fatal(err)
	}
	g.World = w
	if _, err := g.AddBot("kaufmann"); err != nil {
		t.Fatal(err)
	}
	if _, err := g.AddBot("feldherr"); err != nil {
		t.Fatal(err)
	}
	g.Start(0)
	battles := 0
	for step := 0; step < 20000 && !g.Over && g.Year < 1790; step++ {
		moved := false
		for seat := range g.Seats {
			a, ok := g.BotAction(seat, g.Bots[seat])
			if !ok {
				continue
			}
			ev, err := g.Apply(seat, a, int64(step))
			if err != nil {
				ev, err = g.Apply(seat, g.BotFallback(seat), int64(step))
				if err != nil {
					t.Fatalf("Jahr %d, Sitz %d, Phase %s: %v", g.Year, seat, g.Seats[seat].Phase, err)
				}
			}
			for _, e := range ev {
				if e.Type == "battle" {
					battles++
				}
			}
			moved = true
		}
		if !moved {
			t.Fatalf("Spiel hängt in Jahr %d", g.Year)
		}
	}
	if !g.Over && g.Year < 1790 {
		t.Fatalf("Partie nicht beendet, Jahr %d", g.Year)
	}
	if battles == 0 {
		t.Error("Feldherr hat nie Krieg geführt")
	}
}
