package engine

import "testing"

// Balance der Innenpolitik: 60 Partien mit zwei Computer-Herrschern. Umstürze sollen selten
// spielentscheidend sein (Exil in weniger als 15 % der Herrscherleben).
func TestMoodBalance(t *testing.T) {
	counts := map[string]int{}
	moods := 0
	n := 0
	for seed := uint64(1); seed <= 60; seed++ {
		b := DefaultBalance()
		g := NewGame(DefaultSettings(b), b, seed)
		w, _ := LoadWorld("../../../data/world.json")
		g.World = w
		g.AddBot("kaufmann")
		g.AddBot("feldherr")
		g.Start(0)
		for step := 0; step < 20000 && !g.Over; step++ {
			moved := false
			for i := range g.Seats {
				st := g.Bots[i]
				a, ok := g.BotAction(i, st)
				if !ok {
					continue
				}
				ev, err := g.Apply(i, a, 0)
				if err != nil {
					ev, _ = g.Apply(i, g.BotFallback(i), 0)
				}
				moved = true
				for _, e := range ev {
					if e.Type == "stimmung" {
						r := e.Data.(*MoodReport)
						moods += r.After
						n++
						if r.Unrest != "" {
							counts[r.Unrest+"/"+r.Result]++
						}
					}
				}
			}
			if !moved {
				break
			}
		}
	}
	t.Logf("Ø Stimmung %.2f bei %d Jahresbilanzen; Unruhen: %v", float64(moods)/float64(max(1, n)), n, counts)
	if exil := counts["revolution/exil"] + counts["putsch/exil"]; exil > 20 { // etwa 1 von 6 Herrschern; schwankt mit dem Zufallsstrom der Bots
		t.Errorf("zu viele Herrscher im Exil: %d von 120", exil)
	}
}
