// Package engine enthält die reinen Spielregeln von KaiserReloaded.
// Kein I/O, keine Netzwerk-Logik: Funktionen bekommen Zustand und Zufallsquelle
// und verändern nur diesen Zustand.
package engine

// Player ist der vollständige Zustand eines Herrschers.
type Player struct {
	Name    string `json:"name"`
	Country int    `json:"country"` // 1..9, siehe world.json
	Female  bool   `json:"female"`

	Land       int `json:"land"`       // A, Hektar
	Treasury   int `json:"treasury"`   // B, Taler
	Population int `json:"population"` // C
	Grain      int `json:"grain"`      // P, Maß
	Title      int `json:"title"`      // T, 1..9

	Customs   int `json:"customs"`   // ZO, %
	VAT       int `json:"vat"`       // R, %
	IncomeTax int `json:"incomeTax"` // L, %
	Justice   int `json:"justice"`   // D, 1 sehr fair .. 4 gierig

	DeathYear      int `json:"deathYear"`        // K, 0 = verstorben
	SuspendedUntil int `json:"suspendedUntil"`   // B8, Amtsenthebung bis einschließlich Jahr-1
	Joined         int `json:"joined,omitempty"` // Jahr des Beitritts ins laufende Spiel (0 = von Anfang an)

	Markets   int `json:"markets"`   // E
	Mills     int `json:"mills"`     // F
	Palace    int `json:"palace"`    // G, max. 16
	Cathedral int `json:"cathedral"` // H, max. 14

	Cavalry     int     `json:"cavalry"`     // U, Einheiten à 20 Mann
	Artillery   int     `json:"artillery"`   // S
	Infantry    int     `json:"infantry"`    // J
	Militia     int     `json:"militia"`     // Q, wird aus Bauten abgeleitet
	Soldiers    int     `json:"soldiers"`    // I
	Mercenaries int     `json:"mercenaries"` // W
	Morale      float64 `json:"morale"`      // O
	Training    int     `json:"training"`    // Ausbildung 0–10 (Kaiser II, heer.go)
	Equipment   int     `json:"equipment"`   // Material 0–10 (Kaiser II, heer.go)

	// Innenpolitik nach Kaiser II (stimmung.go): Stimmung 0–10 und Gendarmen
	Mood      int  `json:"mood"`
	MoodInit  bool `json:"moodInit"` // false bei Spielständen von vor der Stimmung -> Start 5
	Gendarmes int  `json:"gendarmes"`

	// Finanzen nach Kaiser II (finanzen.go)
	Loan       int `json:"loan"`       // Bankkredit
	ReichBonds int `json:"reichBonds"` // Reichsanleihen
	// Flotte und Kolonien nach Kaiser II (kolonien.go)
	Traders  int       `json:"traders"`  // Handelsschiffe insgesamt
	Warships int       `json:"warships"` // Kriegsschiffe insgesamt
	Colonies [3]Colony `json:"colonies"` // Afrika, Amerika, Asien

	LastTax int `json:"lastTax"` // Staatseinnahmen des Vorjahres: Steuern + Märkte + Mühlen (Anleihenpreis)

	Nobility int     `json:"nobility"` // V, Arbeitsname „Adel“
	Clergy   int     `json:"clergy"`   // M, „Klerus“
	Burghers float64 `json:"burghers"` // WS, „Bürger“
	Trade    float64 `json:"trade"`    // N, „Handel“

	// Profil: frei wählbar, ohne Einfluss auf die Regeln
	Motto  string   `json:"motto"`
	Avatar string   `json:"avatar"` // Dateiname unter /media/, leer = Initiale
	Arms   ArmsSpec `json:"arms"`

	Year YearState `json:"year"`
}

// YearState hält die Werte, die nur für das laufende Jahr gelten.
type YearState struct {
	Weather     int        `json:"weather"`     // 1 Dürre .. 5 Rekordernte
	GrainNeed   int        `json:"grainNeed"`   // NO
	LandPrice   float64    `json:"landPrice"`   // Taler je Hektar
	GrainPrice  int        `json:"grainPrice"`  // Taler je 1000 Maß
	GrainBought int        `json:"grainBought"` // C6, für das Kauflimit
	Fed         bool       `json:"fed"`
	TaxFactors  TaxFactors `json:"taxFactors"`
	UnitPrices  UnitPrices `json:"unitPrices"`
	Risk        string     `json:"risk"`        // offene Risiko-Entscheidung (events.go); immer senden, der Client führt Updates zusammen
	FedRatio    float64    `json:"fedRatio"`    // verteiltes Korn / Bedarf (für die Stimmung)
	Opposition  bool       `json:"opposition"`  // Opposition in diesem Jahr schon verhaftet
	Dividends   int        `json:"dividends"`   // erhaltene Anleihen-Ausschüttungen
	Earnings    int        `json:"earnings"`    // Markt- und Mühlenerträge (Grundlage der Anleihen)
	ShipsBuilt  int        `json:"shipsBuilt"`  // in diesem Jahr gebaute Schiffe
	BotInvest   int        `json:"botInvest"`   // Investitionen eines Computer-Herrschers in diesem Jahr (Schwierigkeit)
	NavalAttack bool       `json:"navalAttack"` // in diesem Jahr schon eine Flotte entsandt
}

// TaxFactors sind die verdeckten Zufallsfaktoren der Steuerphase.
type TaxFactors struct {
	F3, F4, F5, F6 float64
	Tolerance      int
}

// UnitPrices sind die pro Jahr neu gezogenen Grundpreise der Truppen.
type UnitPrices struct {
	Cavalry, Artillery, Infantry int
}

// Alive meldet, ob der Herrscher noch lebt.
func (p *Player) Alive() bool { return p.DeathYear > 0 }

// NewPlayer legt einen Herrscher mit den Startwerten aus der Balance an.
func NewPlayer(name string, country int, female bool, b *Balance, rng *Rng) *Player {
	s := b.Start
	p := &Player{
		Name: name, Country: country, Female: female,
		Land: s.Land, Treasury: s.Treasury, Population: s.Population, Grain: s.Grain,
		Title: max(1, min(8, s.Title)), Customs: s.Customs, VAT: s.VAT, IncomeTax: s.IncomeTax, Justice: s.Justice,
		SuspendedUntil: s.StartYear,
		Infantry:       1, Militia: 1, Soldiers: 20, Morale: 1,
		Nobility: 4, Clergy: 5, Burghers: 25, Trade: 1,
		Arms: ArmsSpec{Kind: "country"},
		Mood: 5, MoodInit: true,
	}
	// Original: festes Todesjahr 1760; Remake: zufällig, damit niemand exakt planen kann.
	if b.Classic {
		p.DeathYear = 1760
	} else {
		p.DeathYear = s.DeathYearMin + rng.IntN(s.DeathYearMax-s.DeathYearMin+1)
	}
	p.RecalcArmy()
	return p
}

// RecalcArmy leitet Miliz und Soldatenzahl aus Bauten und Einheiten ab (760–765).
func (p *Player) RecalcArmy() {
	p.Militia = min(p.Markets/5*2, p.Mills/3*2) + (p.Palace+p.Cathedral)/2
	p.Soldiers = (p.Cavalry + p.Artillery + p.Infantry + p.Militia) * 20
	if p.Mercenaries > p.Soldiers {
		p.Mercenaries = p.Soldiers - p.Militia*20
	}
	p.Mercenaries = max(p.Mercenaries, 0)
	if p.Soldiers < 20 {
		p.Soldiers, p.Infantry, p.Cavalry, p.Artillery = 20, 1, 0, 0
	}
}

// GrainNeed ist der Kornbedarf des Reichs pro Jahr (620).
func (p *Player) GrainNeed() int {
	need := 100*float64(p.Nobility) + 40*float64(p.Clergy) + 30*p.Burghers +
		5*float64(p.Population) + 10*float64(p.Soldiers)
	if need < 0 {
		need = -need
	}
	return int(need)
}
