package engine

import "testing"

func TestRecordCapsAndSlimsBattles(t *testing.T) {
	g := &Game{}
	big := map[string]any{"attacker": 0, "defender": 1, "result": "r", "field": "f", "units": "u", "steps": "s"}
	g.Record([]Event{{Type: "battle", Data: big}})
	g.Record([]Event{{Type: "battle", Data: map[string]any{"attacker": 1, "defender": 0, "result": "r2", "steps": "s2"}}})
	first := g.Log[0].Data.(map[string]any)
	if _, ok := first["steps"]; ok {
		t.Error("ältere Schlacht behält Wiedergabedaten")
	}
	if last := g.Log[1].Data.(map[string]any); last["steps"] != "s2" {
		t.Error("letzte Schlacht verliert Wiedergabedaten")
	}
	for i := 0; i < LogLimit+20; i++ {
		g.Record([]Event{{Type: "newYear", Year: i}})
	}
	if len(g.Log) != LogLimit || g.Log[len(g.Log)-1].Year != LogLimit+19 {
		t.Errorf("Chronik nicht begrenzt: %d Einträge", len(g.Log))
	}
}
