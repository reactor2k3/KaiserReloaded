package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Country ist eines der neun Länder.
type Country struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// World ist die feste Spielwelt aus game/data/world.json.
type World struct {
	Countries []Country `json:"countries"`
	Titles    struct {
		Male   []string `json:"male"`
		Female []string `json:"female"`
	} `json:"titles"`
	// Routes: Angreifer-ID → Wege als Länderfolgen (erstes = Angreifer, letztes = Ziel).
	Routes map[int][][]int `json:"-"`
}

// LoadWorld liest world.json und wandelt die Ziffern-Routen in Länderfolgen um.
func LoadWorld(path string) (*World, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("world lesen: %w", err)
	}
	var w World
	var aux struct {
		Routes map[string][]string `json:"routes"`
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, fmt.Errorf("world parsen: %w", err)
	}
	if err := json.Unmarshal(raw, &aux); err != nil {
		return nil, fmt.Errorf("world-Routen parsen: %w", err)
	}
	w.Routes = map[int][][]int{}
	for k, list := range aux.Routes {
		from, err := strconv.Atoi(k)
		if err != nil {
			return nil, fmt.Errorf("world: ungültiger Angreifer %q", k)
		}
		for _, s := range list {
			route := make([]int, 0, len(s))
			for _, ch := range s {
				if ch < '1' || ch > '9' {
					return nil, fmt.Errorf("world: ungültige Route %q", s)
				}
				route = append(route, int(ch-'0'))
			}
			if len(route) < 2 || route[0] != from {
				return nil, fmt.Errorf("world: Route %q passt nicht zu Angreifer %d", s, from)
			}
			w.Routes[from] = append(w.Routes[from], route)
		}
	}
	return &w, nil
}

// TitleName liefert den Titel in passender Form, z. B. „Fürstin“.
func (w *World) TitleName(title int, female bool) string {
	list := w.Titles.Male
	if female {
		list = w.Titles.Female
	}
	if title < 1 || title > len(list) {
		return ""
	}
	return list[title-1]
}

// Neighbors liefert die direkt angreifbaren Länder (Routen der Länge 2).
func (w *World) Neighbors(country int) []int {
	var out []int
	for _, r := range w.Routes[country] {
		if len(r) == 2 {
			out = append(out, r[1])
		}
	}
	return out
}
