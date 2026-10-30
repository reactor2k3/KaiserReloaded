package store

import (
	"path/filepath"
	"testing"
)

func TestHallSaveAndOrder(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "h.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.SaveHall(HallEntry{ID: "AAAAAA", Ended: 100, Year: 1750, Winner: "Micha", Players: []HallPlayer{{Name: "Micha", Country: 5, Title: 9, Winner: true}}})
	st.SaveHall(HallEntry{ID: "BBBBBB", Ended: 200, Year: 1770, Players: []HallPlayer{{Name: "Anna", Country: 3, Title: 6}}})
	st.SaveHall(HallEntry{ID: "AAAAAA", Ended: 100, Year: 1751, Winner: "Micha", Players: []HallPlayer{{Name: "Micha", Country: 5, Title: 9, Winner: true}}})
	h, err := st.Hall(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 2 || h[0].ID != "BBBBBB" || h[1].Year != 1751 {
		t.Fatalf("Ruhmeshalle falsch: %+v", h)
	}
}
