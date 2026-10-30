package engine

// Settings sind die Einstellungen, die der Gastgeber pro Spiel in der Lobby wählt.
// Sie überschreiben die Standardwerte aus balance.json.
type Settings struct {
	// PhaseSeconds ist die Zugzeit je Phase; 0 = ohne Zeitlimit.
	PhaseSeconds int `json:"phaseSeconds"`
	// TimePenalty: Überziehen kostet Lebenszeit (Original-Regel). Aus = Zug endet nur automatisch.
	TimePenalty bool `json:"timePenalty"`
	// BattleMode: "tactical" (eigene Aufstellung) oder "quick" (automatisch).
	BattleMode string `json:"battleMode"`
	// Classic schaltet die Original-Regeln für dieses Spiel ein.
	Classic bool `json:"classic"`
	// Mortality: Sterblichkeit der Herrscher – "unsterblich", "niedrig", "normal" (leer = normal), "hoch".
	Mortality string `json:"mortality,omitempty"`
	// Difficulty: Spielstärke der Computer-Herrscher – "leicht", "mittel", "schwer" (leer = schwer).
	Difficulty string `json:"difficulty,omitempty"`
	// Mode: "live" (alle gleichzeitig, kurze Zugzeiten, Partie ruht ohne Spieler) oder "async"
	// (lange Zugzeiten von Stunden bis Tagen, Partie läuft auch ohne verbundene Spieler weiter).
	Mode string `json:"mode,omitempty"`
}

// Async meldet ein asynchrones Spiel (Zugzeiten in Stunden, kein Pausieren).
func (s Settings) Async() bool { return s.Mode == "async" }

// Sterblichkeit: Verschiebung der Lebenserwartung (Jahre) und Sterbechance je Jahr danach.
// Normal entspricht dem Original (ab Lebenserwartung jedes Jahr 50 %). Unsterblich schaltet nur
// den natürlichen Tod ab; Exil und Attentat bleiben möglich.
var mortalityRules = map[string]struct {
	Offset int
	Chance float64
}{
	"unsterblich": {0, 0},
	"niedrig":     {15, 0.25},
	"normal":      {0, 0.5},
	"hoch":        {-10, 0.75},
}

// MortalityRule liefert Verschiebung und Sterbechance der gewählten Sterblichkeit.
func (s Settings) MortalityRule() (offset int, chance float64) {
	m, ok := mortalityRules[s.Mortality]
	if !ok {
		m = mortalityRules["normal"]
	}
	return m.Offset, m.Chance
}

// Zugzeit-Auswahl in der Lobby (Sekunden je Phase); 0 = unbegrenzt.
var PhaseSecondsOptions = []int{0, 60, 90, 120, 180, 300}

// Zugzeiten im asynchronen Spiel: 1 h, 6 h, 12 h, 24 h, 3 Tage je Phase.
var AsyncSecondsOptions = []int{3600, 21600, 43200, 86400, 259200}

// DefaultSettings leitet die Lobby-Vorgaben aus der Balance ab.
func DefaultSettings(b *Balance) Settings {
	return Settings{PhaseSeconds: b.PhaseSeconds, TimePenalty: true, BattleMode: "tactical", Classic: b.Classic, Mortality: "normal", Difficulty: "mittel", Mode: "live"}
}

// Validate prüft die Lobby-Eingaben.
func (s Settings) Validate() error {
	if s.Mode != "" && s.Mode != "live" && s.Mode != "async" {
		return &RuleError{"invalid_mode", "Ungültiger Spielmodus."}
	}
	options := PhaseSecondsOptions
	if s.Async() {
		// Asynchron: lange Zugzeit Pflicht, keine Lebenszeit-Strafe fürs Überziehen
		options = AsyncSecondsOptions
		if s.TimePenalty {
			return &RuleError{"invalid_penalty", "Im asynchronen Spiel kostet Überziehen keine Lebenszeit."}
		}
	}
	ok := false
	for _, v := range options {
		if s.PhaseSeconds == v {
			ok = true
		}
	}
	if !ok {
		return &RuleError{"invalid_phase_seconds", "Ungültige Zugzeit."}
	}
	if s.BattleMode != "tactical" && s.BattleMode != "quick" {
		return &RuleError{"invalid_battle_mode", "Ungültiger Schlachtmodus."}
	}
	if _, ok := botSkills[s.Difficulty]; s.Difficulty != "" && !ok {
		return &RuleError{"invalid_difficulty", "Ungültiger Schwierigkeitsgrad."}
	}
	if _, ok := mortalityRules[s.Mortality]; s.Mortality != "" && !ok {
		return &RuleError{"invalid_mortality", "Ungültige Sterblichkeit."}
	}
	return nil
}

// Overtime liefert die Sekunden über dem Limit; ohne Limit oder ohne Strafe immer 0.
func (s Settings) Overtime(usedSeconds int) int {
	if s.PhaseSeconds == 0 || !s.TimePenalty {
		return 0
	}
	return max(0, usedSeconds-s.PhaseSeconds)
}
