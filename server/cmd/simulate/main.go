// Command simulate lässt Computer-Herrscher viele Partien über die echte Engine spielen
// und fasst das Ergebnis für das Balancing zusammen.
//
//	go run ./cmd/simulate -games 200 -players kaufmann,feldherr
//	go run ./cmd/simulate -games 1 -seed 42 -json > verlauf.json   (Einzelpartie mit Jahresverlauf)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kaiserreloaded/server/internal/engine"
)

// Ergebnis einer Partie.
type result struct {
	Winner   int            `json:"winner"` // Sitz des Kaisers, -1 = keiner
	EndYear  int            `json:"endYear"`
	Battles  int            `json:"battles"`
	Titles   []int          `json:"titles"`   // höchster Titel je Sitz
	Lands    []int          `json:"lands"`    // Land am Ende je Sitz
	Bankrupt []int          `json:"bankrupt"` // Bankrotte je Sitz
	Years    []yearSnapshot `json:"years,omitempty"`
}

type yearSnapshot struct {
	Year     int   `json:"year"`
	Land     []int `json:"land"`
	Treasury []int `json:"treasury"`
	Title    []int `json:"title"`
	Soldiers []int `json:"soldiers"`
	Populace []int `json:"population"`
}

func main() {
	games := flag.Int("games", 100, "Anzahl Partien")
	seed := flag.Uint64("seed", 1, "Startwert der ersten Partie (weitere: +1)")
	players := flag.String("players", "kaufmann,feldherr", "Spielweisen der Computer-Herrscher, Komma-Liste")
	maxYears := flag.Int("years", 100, "Abbruch nach so vielen Jahren")
	data := flag.String("data", "../data", "Ordner mit balance.json und world.json")
	classic := flag.Bool("classic", false, "Original-Regeln verwenden")
	flag.StringVar(&difficulty, "difficulty", "schwer", "Spielstärke der Computer-Herrscher: leicht, mittel, schwer")
	asJSON := flag.Bool("json", false, "Ergebnisse als JSON ausgeben (mit Jahresverlauf bei -games 1)")
	flag.Parse()

	b, err := engine.LoadBalance(filepath.Join(*data, "balance.json"))
	must(err)
	b.Classic = *classic
	w, err := engine.LoadWorld(filepath.Join(*data, "world.json"))
	must(err)
	strategies := strings.Split(*players, ",")

	var results []result
	for i := 0; i < *games; i++ {
		results = append(results, play(b, w, strategies, *seed+uint64(i), *maxYears, *games == 1))
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", " ")
		must(enc.Encode(results))
		return
	}
	summarize(results, strategies)
}

// difficulty ist die Spielstärke aller Computer-Herrscher (Flag -difficulty).
var difficulty string

// play spielt eine Partie nur über BotAction/BotFallback – wie die KI im Server.
func play(b *engine.Balance, w *engine.World, strategies []string, seed uint64, maxYears int, track bool) result {
	s := engine.DefaultSettings(b)
	s.Difficulty = difficulty
	s.BattleMode, s.PhaseSeconds = "quick", 0
	g := engine.NewGame(s, b, seed)
	g.World = w
	for _, st := range strategies {
		_, err := g.AddBot(st)
		must(err)
	}
	r := result{Titles: make([]int, len(g.Seats)), Lands: make([]int, len(g.Seats)), Bankrupt: make([]int, len(g.Seats))}
	count := func(ev []engine.Event) {
		for _, e := range ev {
			switch e.Type {
			case "battle":
				r.Battles++
			case "checks":
				if c, ok := e.Data.(engine.CheckReport); ok && c.Bankrupt {
					r.Bankrupt[e.Player]++
				}
			case "newYear":
				if track {
					snap := yearSnapshot{Year: e.Year}
					for _, st := range g.Seats {
						p := st.Player
						snap.Land = append(snap.Land, p.Land)
						snap.Treasury = append(snap.Treasury, p.Treasury)
						snap.Title = append(snap.Title, p.Title)
						snap.Soldiers = append(snap.Soldiers, p.Soldiers)
						snap.Populace = append(snap.Populace, p.Population)
					}
					r.Years = append(r.Years, snap)
				}
			}
		}
	}
	count(g.Start(0))
	for step := int64(1); !g.Over && g.Year < b.Start.StartYear+maxYears && step < 200000; step++ {
		for seat := range g.Seats {
			a, ok := g.BotAction(seat, g.Bots[seat])
			if !ok {
				continue
			}
			ev, err := g.Apply(seat, a, step)
			if err != nil {
				ev, _ = g.Apply(seat, g.BotFallback(seat), step)
			}
			count(ev)
		}
		for i, st := range g.Seats {
			r.Titles[i] = max(r.Titles[i], st.Player.Title)
		}
	}
	r.Winner, r.EndYear = g.Winner, g.Year
	for i, st := range g.Seats {
		r.Lands[i] = st.Player.Land
	}
	return r
}

// summarize druckt eine kurze Auswertung für das Balancing.
func summarize(rs []result, strategies []string) {
	n := len(rs)
	wins := make([]int, len(strategies))
	noWinner, battles := 0, 0
	var years []int
	titleSum := make([]int, len(strategies))
	landSum := make([]int, len(strategies))
	bankrupt := make([]int, len(strategies))
	for _, r := range rs {
		if r.Winner >= 0 {
			wins[r.Winner]++
		} else {
			noWinner++
		}
		battles += r.Battles
		years = append(years, r.EndYear)
		for i := range strategies {
			titleSum[i] += r.Titles[i]
			landSum[i] += r.Lands[i]
			bankrupt[i] += r.Bankrupt[i]
		}
	}
	sort.Ints(years)
	fmt.Printf("%d Partien, Spielweisen: %s\n", n, strings.Join(strategies, " gegen "))
	for i, st := range strategies {
		fmt.Printf("  %-9s Siege %3d (%4.1f %%)  Ø höchster Titel %.1f  Ø Land am Ende %6d  Bankrotte %d\n",
			st, wins[i], 100*float64(wins[i])/float64(n), float64(titleSum[i])/float64(n), landSum[i]/n, bankrupt[i])
	}
	fmt.Printf("  ohne Kaiser %d (%4.1f %%)\n", noWinner, 100*float64(noWinner)/float64(n))
	fmt.Printf("  Spielende: Median %d, frühestens %d, spätestens %d\n", years[n/2], years[0], years[n-1])
	fmt.Printf("  Schlachten: Ø %.1f je Partie\n", float64(battles)/float64(n))
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
