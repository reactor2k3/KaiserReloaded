package engine

import "math"

// HarvestReport beschreibt das Ergebnis der Ernte.
type HarvestReport struct {
	Weather    int     `json:"weather"`
	Yield      int     `json:"yield"`
	RotPercent int     `json:"rotPercent"`
	GrainNeed  int     `json:"grainNeed"`
	LandPrice  float64 `json:"landPrice"`
	GrainPrice int     `json:"grainPrice"`
	GrainAfter int     `json:"grainAfter"`
}

// Harvest startet das Jahr eines Herrschers: Wetter, Ertrag, Preise, Fäulnis (466–472, 620–624).
func Harvest(p *Player, b *Balance, rng *Rng) HarvestReport {
	p.RecalcArmy()
	p.Year = YearState{}
	w := rng.IntN(5) + 1

	// Bewirtschaftbar: begrenzt durch Land, Arbeitskräfte (Mühlen binden 100 je Stück) und Saatgut.
	area := min(p.Land, max(0, 5*(p.Population-100*p.Mills)), 2*p.Grain)
	yield := int(float64(area) * (float64(w) - 0.5))
	p.Grain += yield

	need := p.GrainNeed()
	scarcity := 2.0
	if yield >= 1 {
		scarcity = math.Min(2, math.Max(0.8, float64(need)/float64(yield)))
	}
	base := (3*float64(w) + 12 + rng.Float()*12) / 10
	landPrice := math.Floor(10*base*scarcity) / 10

	// Fäulnis: Original 1–50 %, Remake engere Spanne, Mühlen dienen als Lager.
	var rot int
	if b.Classic {
		rot = rng.IntN(50) + 1
	} else {
		hi := max(b.RotMinPercent, b.RotMaxPercent-p.Mills*b.RotReductionPerMill)
		rot = b.RotMinPercent + rng.IntN(hi-b.RotMinPercent+1)
	}
	p.Grain = int(float64(p.Grain) * (1 - float64(rot)/100))

	// Original sprang bei Preis < 1 zur Ernte zurück (Doppelernte-Bug); hier Mindestpreis 1.
	grainPrice := max(1, int((20-3*float64(w)+rng.Float()*10)*8*scarcity))

	p.Year.Weather, p.Year.GrainNeed = w, need
	p.Year.LandPrice, p.Year.GrainPrice = landPrice, grainPrice
	return HarvestReport{w, yield, rot, need, landPrice, grainPrice, p.Grain}
}

// creditLimit ist der tiefste erlaubte Kassenstand nach einem Kauf (nur Remake).
func creditLimit(p *Player, b *Balance) int { return -b.CreditPerTitle * p.Title }

// pay zieht Kosten ab; im Remake nur innerhalb des Kreditrahmens, im Original ungeprüft.
func pay(p *Player, b *Balance, cost float64) error {
	after := int(math.Floor(float64(p.Treasury) - cost))
	if !b.Classic && after < creditLimit(p, b) {
		return ErrNoMoney
	}
	p.Treasury = after
	return nil
}

// BuyGrain kauft Korn, höchstens 3× Jahresbedarf (494–497).
func BuyGrain(p *Player, b *Balance, qty int) error {
	if qty <= 0 {
		return ErrInvalidAmount
	}
	if p.Year.GrainBought+qty > 3*p.Year.GrainNeed {
		return ErrGrainExhausted
	}
	if err := pay(p, b, float64(qty)*float64(p.Year.GrainPrice)/1000); err != nil {
		return err
	}
	p.Year.GrainBought += qty
	p.Grain += qty
	return nil
}

// SellGrain verkauft Korn mit rund 10 % Abschlag (498–500).
func SellGrain(p *Player, qty int) error {
	if qty <= 0 {
		return ErrInvalidAmount
	}
	if qty > p.Grain {
		return ErrNotEnoughGrain
	}
	p.Year.GrainBought -= qty
	p.Grain -= qty
	p.Treasury = int(float64(p.Treasury) + float64(qty)*float64(p.Year.GrainPrice)/1111)
	return nil
}

// BuyLand kauft Hektar zum Landpreis (501–502).
func BuyLand(p *Player, b *Balance, qty int) error {
	if qty <= 0 {
		return ErrInvalidAmount
	}
	if err := pay(p, b, float64(qty)*p.Year.LandPrice); err != nil {
		return err
	}
	p.Land += qty
	return nil
}

// SellLand verkauft Hektar zu 90 % des Landpreises, mindestens 1 ha bleibt (503–505).
func SellLand(p *Player, qty int) error {
	if qty <= 0 {
		return ErrInvalidAmount
	}
	if qty > p.Land-1 {
		return ErrNotEnoughLand
	}
	p.Land -= qty
	p.Treasury = int(float64(p.Treasury) + float64(qty)*p.Year.LandPrice*0.9)
	return nil
}

// FeedReport fasst die Bevölkerungsbewegung eines Jahres zusammen.
type FeedReport struct {
	Births     int `json:"births"`
	Deaths     int `json:"deaths"`
	Immigrants int `json:"immigrants"`
	Emigrants  int `json:"emigrants"`
}

// FeedLimits liefert die erlaubte Ausgabe-Spanne (20 % bis 80 % der Reserve).
func FeedLimits(p *Player) (lo, hi int) {
	// Bei sehr kleiner Reserve (z. B. nach einem Kornspeicherbrand) darf das Minimum das Maximum
	// nicht übersteigen, sonst ließe sich gar nichts verteilen.
	hi = int(float64(p.Grain) * 0.8)
	return min(int(math.Ceil(float64(p.Grain)/5)), hi), hi
}

// Feed verteilt Korn an das Volk und berechnet Geburten, Tod und Wanderung (506–535).
func Feed(p *Player, b *Balance, rng *Rng, amount int) (FeedReport, error) {
	var rep FeedReport
	if p.Year.Fed {
		return rep, ErrAlreadyFed
	}
	lo, hi := FeedLimits(p)
	if amount > hi {
		return rep, ErrTooMuch
	}
	if amount < lo {
		return rep, ErrTooLittle
	}
	p.Grain -= amount
	p.Year.Fed = true
	need := float64(max(1, p.Year.GrainNeed))
	p.Year.FedRatio = float64(amount) / need
	z := float64(amount)
	pop := func() float64 { return float64(p.Population) }

	if z >= need-1 {
		rep.Births = int((rng.Float()*7 + 1) * pop() / 100)
		p.Population += rep.Births
		rep.Deaths = p.die(b, int((rng.Float()*3+1)*pop()/100))
		if rng.Float()*20+1 > float64(p.VAT) {
			p.Nobility += rng.IntN(2)
		}
		if rng.Float()*20+1 > float64(p.IncomeTax) {
			p.Clergy += rng.IntN(3)
		}
		// Überfluss lockt Einwanderer, gebremst durch hohe Steuern.
		if need*(1.1+rng.Float()*0.4) <= z {
			d := (z - need) / need * 10 * (pop() / 1000) * (rng.Float()*65 + 2)
			d = math.Min(d, math.Floor(pop()/10))
			imm := int((rng.Float()*d + 2) * (1.1 - float64(p.VAT)/100) *
				(1.1 - float64(p.Customs)/100) * (1.1 - float64(p.IncomeTax)/100) / 2)
			rep.Immigrants = imm
			p.Population += imm
			p.Burghers += math.Min(50, rng.Float()*float64(imm)/5+1)
			p.Nobility++
			p.Clergy += 2
		}
		if p.Justice > 2 || 90-p.Customs-p.VAT-p.IncomeTax < 0 {
			rep.Emigrants = p.emigrate(b, rng)
		}
		return rep, nil
	}

	// Hunger: Sterberate steigt mit dem Mangel.
	short := (need-z)/need*100 - 9
	short = math.Max(0, math.Min(65, short))
	rep.Births = int((rng.Float()*3 + 1) * pop() / 100)
	p.Population += rep.Births
	deaths := int((rng.Float()*(short+8) + 1) * pop() / 100)
	if z/need < 0.5 {
		deaths = int((1 - z/need) * pop())
	}
	rep.Deaths = p.die(b, deaths)
	rep.Emigrants = p.emigrate(b, rng)
	return rep, nil
}

// die zieht Sterbefälle ab, ohne unter die Mindestbevölkerung zu fallen.
func (p *Player) die(b *Balance, n int) int {
	if p.Population-n < b.MinPopulation {
		n = max(0, p.Population-b.MinPopulation)
	}
	p.Population -= n
	return n
}

// emigrate: Auswanderung wegen Justiz (630). Original trifft auch Justiz 1 („sehr fair“).
func (p *Player) emigrate(b *Balance, rng *Rng) int {
	if !b.Classic && p.Justice == 1 {
		return 0
	}
	d := float64(p.Justice - 2)
	n := int((2 + rng.Float()) * (float64(p.Population) / 100 * d * d))
	if n < 2 {
		return 0
	}
	p.Population -= n
	return n
}

// IncomeReport: Erträge der Märkte und Mühlen, Sold der Armee (536–542).
type IncomeReport struct {
	Markets int `json:"markets"`
	Mills   int `json:"mills"`
	Pay     int `json:"pay"`
}

// CollectIncome bucht Markt- und Mühlenerträge und zieht den Sold ab.
func CollectIncome(p *Player, rng *Rng) IncomeReport {
	r := IncomeReport{
		Markets: p.Markets * int(127+rng.Float()*127),
		Mills:   p.Mills * int(250+rng.Float()*250),
		Pay:     p.Soldiers*3 + p.Mercenaries*12 + p.Gendarmes*GendarmPay,
	}
	p.Treasury += r.Markets + r.Mills - r.Pay
	return r
}
