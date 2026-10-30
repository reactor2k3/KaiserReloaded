package engine

import "math"

// Schlachtfeld und Gefecht, nachgebaut nach dem Vorbild.
// Das Feld ist kleiner als im Original (24 statt 76 Zeilen), die Regeln je Zelle sind gleich.

const (
	FieldRows = 24
	FieldCols = 40
)

// Zellarten des Geländes. Gebäude gehören der Seite, auf der sie liegen.
const (
	CellEmpty     = '.'
	CellTree      = 'T'
	CellRiver     = '~'
	CellMarket    = 'M'
	CellMill      = 'L'
	CellPalace    = 'P'
	CellCathedral = 'K'
	CellCapital   = 'H'
)

// Field ist das Gelände einer Schlacht; Border[r] ist die Spalte der Grenzlinie in Zeile r.
type Field struct {
	Rows   int      `json:"rows"`
	Cols   int      `json:"cols"`
	Border []int    `json:"border"`
	Cells  []string `json:"cells"` // je Zeile ein String aus Zellarten
}

// Kampfeinheit auf dem Feld.
type BattleUnit struct {
	ID   int    `json:"id"`
	Side int    `json:"side"` // 0 Angreifer, 1 Verteidiger
	Kind string `json:"kind"` // cavalry | artillery | infantry | militia
	Row  int    `json:"row"`
	Col  int    `json:"col"`
	Dead bool   `json:"dead"`
}

// Placement ist der Aufstellungswunsch eines Spielers für eine Einheit.
type Placement struct {
	Kind string `json:"kind"`
	Row  int    `json:"row"`
	Col  int    `json:"col"`
}

// BattleStep ist ein Schritt der Wiedergabe für die Animation im Client.
type BattleStep struct {
	Unit   int    `json:"unit"`
	Action string `json:"action"` // move | win | lose | destroy | shot | stop
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Target int    `json:"target,omitempty"` // getroffene Einheit
	What   string `json:"what,omitempty"`   // getroffenes Gebäude
}

// SideLosses zählt die Verluste einer Seite.
type SideLosses struct {
	Cavalry   int     `json:"cavalry"`
	Artillery int     `json:"artillery"`
	Infantry  int     `json:"infantry"`
	Militia   int     `json:"militia"`
	Markets   int     `json:"markets"`
	Mills     int     `json:"mills"`
	Palace    float64 `json:"palace"`    // Schaden in Bauteilen
	Cathedral float64 `json:"cathedral"` // Schaden in Bauteilen
}

// Battle ist ein laufendes bzw. entschiedenes Gefecht.
type Battle struct {
	Field  Field         `json:"field"`
	Units  []*BattleUnit `json:"units"`
	Start  []BattleUnit  `json:"start"` // Aufstellung vor dem Gefecht (für die Wiedergabe)
	Steps  []BattleStep  `json:"steps"`
	Losses [2]SideLosses `json:"losses"`
	Land   int           `json:"land"` // > 0 Angreifer gewinnt Hektar, < 0 verliert
	grid   [][]byte
	occ    [][]*BattleUnit
}

// NewField erzeugt Gelände: Grenzlinie, Bäume, ein Fluss je Seite, Bauten nach Besitz.
func NewField(rng *Rng, att, def *Player) Field {
	f := Field{Rows: FieldRows, Cols: FieldCols, Border: make([]int, FieldRows)}
	g := make([][]byte, FieldRows)
	x := FieldCols / 2
	for r := range g {
		g[r] = []byte(stringsRepeat(CellEmpty, FieldCols))
		// Grenzlinie schlängelt sich um die Mitte (397–403)
		x = max(FieldCols/2-2, min(FieldCols/2+2, x+rng.IntN(3)-1))
		f.Border[r] = x
	}
	place := func(side int, c byte, n int) {
		for i := 0; i < n*10 && n > 0; i++ {
			r := rng.IntN(FieldRows)
			lo, hi := 1, f.Border[r]-2
			if side == 1 {
				lo, hi = f.Border[r]+2, FieldCols-2
			}
			col := lo + rng.IntN(max(1, hi-lo))
			if g[r][col] == CellEmpty {
				g[r][col] = c
				n--
			}
		}
	}
	for side, p := range []*Player{att, def} {
		place(side, CellCapital, 1)
		place(side, CellMarket, min(8, p.Markets))
		place(side, CellMill, min(4, p.Mills))
		if p.Palace > 4 {
			place(side, CellPalace, 1)
		}
		if p.Cathedral > 0 {
			place(side, CellCathedral, 1)
		}
	}
	// Bäume (426–430) und je ein kurzer Flusslauf pro Seite (82–91)
	for i := 0; i < 70; i++ {
		r, c := rng.IntN(FieldRows), rng.IntN(FieldCols)
		if g[r][c] == CellEmpty && c != f.Border[r] {
			g[r][c] = CellTree
		}
	}
	for side := 0; side < 2; side++ {
		r := rng.IntN(FieldRows)
		c := 2 + rng.IntN(6)
		if side == 1 {
			c = FieldCols - 3 - rng.IntN(6)
		}
		for n := 0; n < 14 && r >= 0 && r < FieldRows; n++ {
			if g[r][c] == CellEmpty {
				g[r][c] = CellRiver
			}
			switch rng.IntN(3) {
			case 0:
				r++
			case 1:
				c = max(1, min(FieldCols-2, c+1-2*side))
			default:
				r--
			}
		}
	}
	f.Cells = make([]string, FieldRows)
	for r := range g {
		f.Cells[r] = string(g[r])
	}
	return f
}

func stringsRepeat(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

// sideCells liefert die freien Zellen einer Seite für die Aufstellung (mit Abstand zur Grenze).
func (f *Field) validPlacement(side, r, c int) bool {
	if r < 0 || r >= f.Rows || c < 0 || c >= f.Cols || f.Cells[r][c] != CellEmpty {
		return false
	}
	if side == 0 {
		return c <= f.Border[r]-2
	}
	return c >= f.Border[r]+2
}

// AutoPlace stellt Einheiten automatisch auf: Kavallerie und Infanterie nach vorn,
// Artillerie dahinter, Miliz bei den eigenen Bauten.
func AutoPlace(rng *Rng, f *Field, side int, counts map[string]int, taken map[[2]int]bool) []Placement {
	var out []Placement
	depth := map[string][2]int{"cavalry": {2, 5}, "infantry": {2, 6}, "artillery": {6, 10}, "militia": {8, 14}}
	for _, kind := range []string{"cavalry", "infantry", "artillery", "militia"} {
		for n := counts[kind]; n > 0; {
			placed := false
			for try := 0; try < 400 && !placed; try++ {
				r := rng.IntN(f.Rows)
				d := depth[kind][0] + rng.IntN(depth[kind][1]-depth[kind][0]+1)
				c := f.Border[r] - d
				if side == 1 {
					c = f.Border[r] + d
				}
				if f.validPlacement(side, r, c) && !taken[[2]int{r, c}] {
					taken[[2]int{r, c}] = true
					out = append(out, Placement{kind, r, c})
					placed = true
				}
			}
			if !placed {
				break // Feld voll: restliche Einheiten bleiben in Reserve
			}
			n--
		}
	}
	return out
}

// Gewicht einer verteidigenden Einheit beim Zusammenstoß (172–174, 219–221).
var defenceWeight = map[string]float64{"cavalry": 5, "artillery": 0.6, "infantry": 1, "militia": 1.5}

// Fight führt das Gefecht aus: erst Angriff (Zeile für Zeile), dann Gegenangriff.
// morale: Moral der beiden Seiten; landValue: Hektar je Schritt im Feindesland (A/800).
func (b *Battle) Fight(rng *Rng, morale [2]float64, landValue [2]int) {
	b.grid = make([][]byte, b.Field.Rows)
	b.occ = make([][]*BattleUnit, b.Field.Rows)
	for r := range b.grid {
		b.grid[r] = []byte(b.Field.Cells[r])
		b.occ[r] = make([]*BattleUnit, b.Field.Cols)
	}
	for _, u := range b.Units {
		b.occ[u.Row][u.Col] = u
	}
	for side := 0; side < 2; side++ {
		rows := make([]int, b.Field.Rows)
		for i := range rows {
			rows[i] = i
			if side == 1 { // Gegenangriff von unten nach oben (301)
				rows[i] = b.Field.Rows - 1 - i
			}
		}
		for _, r := range rows {
			z := 0.0 // Schwung gefallener Einheiten geht an die nächste der Zeile
			for _, u := range b.frontOrder(side, r) {
				if u.Dead {
					continue
				}
				switch u.Kind {
				case "cavalry", "infantry":
					z = b.charge(rng, u, morale, landValue, z)
				case "artillery":
					b.bombard(rng, u, &morale)
				}
			}
		}
	}
}

// frontOrder: Einheiten einer Zeile, vorderste zuerst.
func (b *Battle) frontOrder(side, r int) []*BattleUnit {
	var out []*BattleUnit
	if side == 0 {
		for c := b.Field.Cols - 1; c >= 0; c-- {
			if u := b.occ[r][c]; u != nil && u.Side == 0 {
				out = append(out, u)
			}
		}
	} else {
		for c := 0; c < b.Field.Cols; c++ {
			if u := b.occ[r][c]; u != nil && u.Side == 1 {
				out = append(out, u)
			}
		}
	}
	return out
}

// charge: Kavallerie (Reichweite 19, Stärke 2) oder Infanterie (13, 1) stürmt vor (158–185).
func (b *Battle) charge(rng *Rng, u *BattleUnit, morale [2]float64, landValue [2]int, z float64) float64 {
	dir, rangeLeft, str := 1, 13, 1.0
	if u.Kind == "cavalry" {
		rangeLeft, str = 19, 2
	}
	if u.Side == 1 {
		dir = -1
	}
	str *= morale[u.Side]
	enemy := 1 - u.Side
	for ; rangeLeft > 0 && str > 0; rangeLeft-- {
		nc := u.Col + dir
		if nc < 0 || nc >= b.Field.Cols {
			break
		}
		cell := b.grid[u.Row][nc]
		if t := b.occ[u.Row][nc]; t != nil {
			if t.Side == u.Side {
				break // eigene Einheit im Weg
			}
			g := morale[enemy] * defenceWeight[t.Kind]
			if rng.Float()*(g+str+z) > g {
				b.kill(t)
				b.Steps = append(b.Steps, BattleStep{Unit: u.ID, Action: "win", Row: u.Row, Col: nc, Target: t.ID})
				str -= g
				z = 0
			} else {
				b.kill(u)
				b.Steps = append(b.Steps, BattleStep{Unit: u.ID, Action: "lose", Row: u.Row, Col: u.Col, Target: t.ID})
				return z + str
			}
			continue
		}
		switch cell {
		case CellTree:
			str *= 0.95
		case CellRiver:
			str *= 0.6
		case CellMarket, CellMill:
			if b.ownerOf(u.Row, nc) == enemy {
				b.destroy(enemy, u.Row, nc, cell)
				b.Steps = append(b.Steps, BattleStep{Unit: u.ID, Action: "destroy", Row: u.Row, Col: nc, What: string(cell)})
				pen := 0.3
				if cell == CellMill {
					pen += 0.25
				}
				str = str + z - pen
				z = 0
			}
		case CellPalace, CellCathedral:
			if b.ownerOf(u.Row, nc) == enemy {
				b.damage(enemy, cell, str)
			}
		}
		// Schritt ausführen
		b.occ[u.Row][u.Col] = nil
		u.Col = nc
		b.occ[u.Row][u.Col] = u
		b.Steps = append(b.Steps, BattleStep{Unit: u.ID, Action: "move", Row: u.Row, Col: u.Col})
		if b.ownerOf(u.Row, u.Col) == enemy {
			if u.Side == 0 {
				b.Land += landValue[0]
			} else {
				b.Land -= landValue[1]
			}
		}
	}
	return z
}

// bombard: Artillerie schießt 16–21 Felder weit, je Schuss Stärke −0,3 (186–202).
func (b *Battle) bombard(rng *Rng, u *BattleUnit, morale *[2]float64) {
	str := math.Min(1.5, morale[u.Side])
	enemy := 1 - u.Side
	dir := 1
	if u.Side == 1 {
		dir = -1
	}
	for ; str > 0; str -= 0.3 {
		c := u.Col + dir*(16+rng.IntN(6))
		r := max(0, min(b.Field.Rows-1, u.Row+rng.IntN(3)-1))
		if c < 0 || c >= b.Field.Cols {
			continue
		}
		step := BattleStep{Unit: u.ID, Action: "shot", Row: r, Col: c}
		if t := b.occ[r][c]; t != nil && t.Side == enemy {
			b.kill(t)
			step.Target = t.ID
			morale[enemy] = math.Max(0.1, morale[enemy]-0.02)
		} else if cell := b.grid[r][c]; b.ownerOf(r, c) == enemy {
			switch cell {
			case CellMarket, CellMill:
				b.destroy(enemy, r, c, cell)
				step.What = string(cell)
			case CellPalace, CellCathedral:
				b.damage(enemy, cell, 0.5)
				step.What = string(cell)
			}
		}
		b.Steps = append(b.Steps, step)
	}
}

func (b *Battle) ownerOf(r, c int) int {
	if c < b.Field.Border[r] {
		return 0
	}
	return 1
}

func (b *Battle) kill(u *BattleUnit) {
	u.Dead = true
	b.occ[u.Row][u.Col] = nil
	l := &b.Losses[u.Side]
	switch u.Kind {
	case "cavalry":
		l.Cavalry++
	case "artillery":
		l.Artillery++
	case "infantry":
		l.Infantry++
	case "militia":
		l.Militia++
	}
}

func (b *Battle) destroy(side, r, c int, cell byte) {
	b.grid[r][c] = CellEmpty
	if cell == CellMarket {
		b.Losses[side].Markets++
	} else {
		b.Losses[side].Mills++
	}
}

// damage: Palast/Kathedrale nehmen Schaden 2·Stärke/Typ (78).
func (b *Battle) damage(side int, cell byte, str float64) {
	if cell == CellPalace {
		b.Losses[side].Palace += 2 * str / 2
	} else {
		b.Losses[side].Cathedral += 2 * str / 3
	}
}
