package engine

import (
	"encoding/json"
	"fmt"
	"os"
)

// StartValues sind die Startwerte eines neuen Herrschers.
type StartValues struct {
	Land         int `json:"land"`
	Treasury     int `json:"treasury"`
	Population   int `json:"population"`
	Grain        int `json:"grain"`
	Customs      int `json:"customs"`
	VAT          int `json:"vat"`
	IncomeTax    int `json:"incomeTax"`
	Justice      int `json:"justice"`
	DeathYearMin int `json:"deathYearMin"`
	DeathYearMax int `json:"deathYearMax"`
	StartYear    int `json:"startYear"`
	Title        int `json:"title"` // Starttitel, Standard 1; höher nur für Tests und Varianten
}

// Balance bündelt alle einstellbaren Zahlen. Classic schaltet die Original-Regeln
// des Vorbilds ein; die übrigen Felder gelten dann nicht.
type Balance struct {
	Classic             bool        `json:"classic"`
	Start               StartValues `json:"start"`
	RotMinPercent       int         `json:"rotMinPercent"`
	RotMaxPercent       int         `json:"rotMaxPercent"`
	RotReductionPerMill int         `json:"rotReductionPerMill"`
	CreditPerTitle      int         `json:"creditPerTitle"`
	InterestOnCredit    float64     `json:"interestOnCredit"`
	InterestOnSavings   float64     `json:"interestOnSavings"`
	PhaseSeconds        int         `json:"phaseSeconds"`
	MinPopulation       int         `json:"minPopulation"`
	CountryArea         int         `json:"countryArea"` // Hektar je Land, geteilt von bis zu drei Herrschern (0 = ohne Grenze)
}

// DefaultBalance entspricht game/data/balance.json und dient Tests als Grundlage.
func DefaultBalance() *Balance {
	return &Balance{
		Start: StartValues{
			Land: 10000, Treasury: 15000, Population: 2000, Grain: 10000,
			Customs: 25, VAT: 10, IncomeTax: 5, Justice: 2,
			DeathYearMin: 1760, DeathYearMax: 1780, StartYear: 1700, Title: 1,
		},
		RotMinPercent: 5, RotMaxPercent: 25, RotReductionPerMill: 1,
		CreditPerTitle: 5000, InterestOnCredit: 0.10, InterestOnSavings: 0.03,
		PhaseSeconds: 90, MinPopulation: 382, CountryArea: 50000,
	}
}

// ClassicBalance liefert die Regeln des C64-Originals.
func ClassicBalance() *Balance {
	b := DefaultBalance()
	b.Classic = true
	return b
}

// LoadBalance liest eine Balance-Datei; fehlende Felder behalten die Standardwerte.
func LoadBalance(path string) (*Balance, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("balance lesen: %w", err)
	}
	b := DefaultBalance()
	if err := json.Unmarshal(raw, b); err != nil {
		return nil, fmt.Errorf("balance parsen: %w", err)
	}
	if b.RotMinPercent < 0 || b.RotMaxPercent < b.RotMinPercent || b.RotMaxPercent > 100 {
		return nil, fmt.Errorf("balance: ungültige Fäulnis-Spanne %d..%d", b.RotMinPercent, b.RotMaxPercent)
	}
	if b.Start.DeathYearMax < b.Start.DeathYearMin {
		return nil, fmt.Errorf("balance: deathYearMax < deathYearMin")
	}
	return b, nil
}
