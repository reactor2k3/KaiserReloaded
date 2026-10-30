package store

import (
	"path/filepath"
	"reflect"
	"testing"

	"kaiserreloaded/server/internal/engine"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	b := engine.DefaultBalance()
	g := engine.NewGame(engine.DefaultSettings(b), b, 5)
	g.AddPlayer("Micha", 5, false)
	g.AddPlayer("Anna", 3, true)
	g.Start(100)
	snap, err := g.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	rec := Record{ID: "ABCDEF", Started: true, Tokens: []string{"t1", "t2"}, Snapshot: snap}
	if err := st.Save(rec); err != nil {
		t.Fatal(err)
	}
	rec.Tokens = []string{"t1", "t2", "t3"} // zweites Speichern überschreibt
	if err := st.Save(rec); err != nil {
		t.Fatal(err)
	}

	all, err := st.LoadAll()
	if err != nil || len(all) != 1 {
		t.Fatalf("LoadAll: %v, %d Einträge", err, len(all))
	}
	if !reflect.DeepEqual(all[0].Tokens, rec.Tokens) || !all[0].Started {
		t.Errorf("Datensatz verändert: %+v", all[0])
	}
	g2, err := engine.RestoreGame(all[0].Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.Public(), g2.Public()) || g.Year != g2.Year {
		t.Error("wiederhergestelltes Spiel weicht ab")
	}
	// Gleicher Zufallszustand: dieselbe Aktion führt zum selben Ergebnis.
	a := engine.Action{Type: "next"}
	g.Apply(0, a, 110)
	g2.Apply(0, a, 110)
	f1, _ := g.Apply(0, engine.Action{Type: "feed", Qty: g.Self(0).FeedMin}, 120)
	f2, _ := g2.Apply(0, engine.Action{Type: "feed", Qty: g2.Self(0).FeedMin}, 120)
	if !reflect.DeepEqual(f1, f2) {
		t.Errorf("Zufall nach Wiederherstellung verschieden:\n%v\n%v", f1, f2)
	}
}
