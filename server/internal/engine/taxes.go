package engine

// TaxReport sind die Staatseinnahmen bei den aktuellen Sätzen.
type TaxReport struct {
	Customs   int `json:"customs"`
	VAT       int `json:"vat"`
	IncomeTax int `json:"incomeTax"`
	Justice   int `json:"justice"`
	Total     int `json:"total"`
}

// DrawTaxFactors zieht die verdeckten Faktoren des Jahres (543–544). Einmal pro Jahr,
// damit Satzänderungen im Dialog eine stabile Vorschau ergeben.
func DrawTaxFactors(p *Player, rng *Rng) {
	b := float64(p.Population) / 100
	p.Year.TaxFactors = TaxFactors{
		F3:        b / (rng.Float()*3000 + 2500),
		F4:        b / (rng.Float()*4000 + 2000),
		F5:        b / (rng.Float()*2000 + 3000),
		F6:        b / (rng.Float()*3000 + 2500) * 100,
		Tolerance: int(70 + rng.Float()*82),
	}
}

// Taxes berechnet die Einnahmen bei den aktuellen Sätzen, ohne sie zu buchen (633–637).
func Taxes(p *Player) TaxReport {
	f := p.Year.TaxFactors
	v, ws, n, t, d := float64(p.Nobility), p.Burghers, p.Trade, float64(p.Title), float64(p.Justice)
	room := float64(f.Tolerance - p.Customs - p.VAT - p.IncomeTax)

	customs := float64(int((v*180+t*75+ws*20)*(room/100) + n*100))
	customs = float64(int(customs * float64(p.Customs) * f.F3))
	vat := float64(int((v*50 + ws*75 + n*10) * room * (5 - d) / 200))
	vat = float64(int(vat * float64(p.VAT) * f.F4))
	income := float64(int(v*250 + n*20 + (10*d*v)*(room/100)))
	income = float64(int(income * float64(p.IncomeTax) * f.F5))
	justice := int((d*300 - 500) * t * f.F6)
	if p.Title == 8 { // König: Sonderrechte bei Zoll und Einkommensteuer
		customs *= 3
		income *= 2
	}
	// Volksstimmung (Kaiser II): zufriedenes Volk zahlt mehr, unruhiges weniger
	mf := MoodFactor(p)
	customs, vat, income = customs*mf, vat*mf, income*mf
	r := TaxReport{Customs: int(customs), VAT: int(vat), IncomeTax: int(income), Justice: justice}
	r.Total = r.Customs + r.VAT + r.IncomeTax + r.Justice
	return r
}

// SetTaxes ändert Sätze und Justiz mit Prüfung der Grenzen (557–570).
func SetTaxes(p *Player, customs, vat, incomeTax, justice int) error {
	for _, v := range []int{customs, vat, incomeTax} {
		if v < 0 || v > 99 {
			return ErrInvalidTax
		}
	}
	if justice < 1 || justice > 4 {
		return ErrInvalidJustice
	}
	p.Customs, p.VAT, p.IncomeTax, p.Justice = customs, vat, incomeTax, justice
	return nil
}

// CollectTaxes bucht die Einnahmen des Jahres (571).
func CollectTaxes(p *Player) TaxReport {
	r := Taxes(p)
	p.Treasury += r.Total
	return r
}
