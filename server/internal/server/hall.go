package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"

	"kaiserreloaded/server/internal/store"
)

// hallStat fasst die Partien eines Spielernamens zusammen (ohne Accounts: nach Name).
type hallStat struct {
	Name      string `json:"name"`
	Games     int    `json:"games"`
	Crowns    int    `json:"crowns"`
	BestTitle int    `json:"bestTitle"`
}

// HallHandler liefert GET /api/ruhmeshalle: jüngste Partien und Bestenliste der Menschen.
func HallHandler(st *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Nur GET erlaubt.", http.StatusMethodNotAllowed)
			return
		}
		entries, err := st.Hall(200)
		if err != nil {
			slog.Error("ruhmeshalle lesen", "err", err)
			http.Error(w, "Ruhmeshalle nicht verfügbar.", http.StatusInternalServerError)
			return
		}
		stats := map[string]*hallStat{}
		for _, e := range entries {
			for _, p := range e.Players {
				if p.Bot {
					continue
				}
				s := stats[p.Name]
				if s == nil {
					s = &hallStat{Name: p.Name}
					stats[p.Name] = s
				}
				s.Games++
				s.BestTitle = max(s.BestTitle, p.Title)
				if p.Winner {
					s.Crowns++
				}
			}
		}
		best := make([]*hallStat, 0, len(stats))
		for _, s := range stats {
			best = append(best, s)
		}
		sort.Slice(best, func(i, j int) bool {
			if best[i].Crowns != best[j].Crowns {
				return best[i].Crowns > best[j].Crowns
			}
			if best[i].BestTitle != best[j].BestTitle {
				return best[i].BestTitle > best[j].BestTitle
			}
			return best[i].Games > best[j].Games
		})
		if len(best) > 10 {
			best = best[:10]
		}
		recent := entries
		if len(recent) > 12 {
			recent = recent[:12]
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]any{"recent": recent, "best": best})
	})
}
