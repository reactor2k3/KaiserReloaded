package engine

// LogLimit begrenzt die gespeicherte Chronik; ältere Einträge fallen heraus.
const LogLimit = 150

// Record hängt Ereignisse an die Chronik an. Nur die letzte Schlacht behält die
// Wiedergabedaten (Feld, Einheiten, Schritte); ältere werden auf das Ergebnis gekürzt,
// damit Spielstand und Wiedereinstieg klein bleiben.
func (g *Game) Record(ev []Event) {
	for _, e := range ev {
		if e.Type == "battle" {
			for i := range g.Log {
				if g.Log[i].Type == "battle" {
					g.Log[i].Data = battleSummary(g.Log[i].Data)
				}
			}
		}
		g.Log = append(g.Log, e)
	}
	if over := len(g.Log) - LogLimit; over > 0 {
		g.Log = append([]Event(nil), g.Log[over:]...)
	}
}

// battleSummary behält von einem Schlacht-Ereignis nur Parteien und Ergebnis.
func battleSummary(data any) any {
	m, ok := data.(map[string]any)
	if !ok {
		return data
	}
	out := map[string]any{}
	for _, k := range []string{"attacker", "defender", "result"} {
		out[k] = m[k]
	}
	return out
}
