package engine

import "math/rand/v2"

// Rng ist die deterministische Zufallsquelle eines Spiels. Gleicher Seed und
// gleiche Aktionen ergeben denselben Spielverlauf (Replays, Fehlersuche).
type Rng struct {
	src *rand.PCG
	r   *rand.Rand
}

// NewRng erzeugt eine Zufallsquelle aus einem 64-Bit-Seed.
func NewRng(seed uint64) *Rng {
	src := rand.NewPCG(seed, seed^0x9E3779B97F4A7C15)
	return &Rng{src: src, r: rand.New(src)}
}

// Float liefert 0 ≤ x < 1, entspricht RND(0) im Original.
func (g *Rng) Float() float64 { return g.r.Float64() }

// IntN liefert 0 ≤ x < n.
func (g *Rng) IntN(n int) int { return g.r.IntN(n) }

// MarshalBinary sichert den Zustand, damit ein Spiel nach Neustart weiterläuft.
func (g *Rng) MarshalBinary() ([]byte, error) { return g.src.MarshalBinary() }

// UnmarshalBinary stellt einen gesicherten Zustand wieder her.
func (g *Rng) UnmarshalBinary(data []byte) error {
	if g.src == nil {
		g.src = &rand.PCG{}
		g.r = rand.New(g.src)
	}
	return g.src.UnmarshalBinary(data)
}
