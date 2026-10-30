package engine

import "math"

// CheckReport fasst die Prüfungen nach der Steuerphase zusammen.
type CheckReport struct {
	Bankrupt    bool `json:"bankrupt"`
	Suspended   bool `json:"suspended"`
	LostMarkets int  `json:"lostMarkets"`
	LostMills   int  `json:"lostMills"`
	LostPalace  int  `json:"lostPalace"`
	LostCathed  int  `json:"lostCathedral"`
}

// AfterTaxChecks prüft Bankrott, Amtsenthebung und Landmangel (571–573, 695–717, 766).
func AfterTaxChecks(p *Player, b *Balance, rng *Rng, year int) CheckReport {
	var r CheckReport
	if IsBankrupt(p, rng) {
		Bankruptcy(p, b, year)
		r.Bankrupt = true
		return r
	}
	if p.Land < p.Population {
		p.SuspendedUntil = year + 2 // nächstes Jahr wird ausgesetzt
		r.Suspended = true
		return r
	}
	plots := p.Land / 1000
	if p.Markets > plots {
		r.LostMarkets, p.Markets = p.Markets-plots, plots
	}
	if p.Mills > plots {
		r.LostMills, p.Mills = p.Mills-plots, plots
	}
	// Original: Palast/Kathedrale komplett verloren; Remake: nur Teile über der Grenze.
	if plots < 12 && p.Palace > 0 {
		keep := 0
		if !b.Classic {
			keep = min(p.Palace, max(0, plots-12+p.Palace))
		}
		r.LostPalace, p.Palace = p.Palace-keep, keep
	}
	if plots < 24 && p.Cathedral > 0 {
		keep := 0
		if !b.Classic {
			keep = min(p.Cathedral, max(0, plots-24+p.Cathedral))
		}
		r.LostCathed, p.Cathedral = p.Cathedral-keep, keep
	}
	p.RecalcArmy()
	return r
}

// IsBankrupt: Kasse unter −10 000 × Titel × 0,8..1,3 (695).
func IsBankrupt(p *Player, rng *Rng) bool {
	limit := -10000 * float64(p.Title) * float64(8+rng.IntN(6)) / 10
	return limit > float64(p.Treasury)
}

// Bankruptcy pfändet Besitz: Gebäude verkauft, Söldner und Moral halbiert,
// Land über 3000 ha verkauft, Restgeld in Land, Lebenserwartung −2 (728–739).
func Bankruptcy(p *Player, b *Balance, year int) {
	p.Treasury += p.Cathedral*2500 + p.Palace*1500 + p.Mills*1000 + p.Markets*500
	p.Cathedral, p.Palace, p.Mills, p.Markets = 0, 0, 0, 0
	p.Mercenaries /= 2
	p.Morale /= 2
	price := math.Max(0.1, p.Year.LandPrice)
	if p.Land > 3000 {
		p.Treasury = int(float64(p.Treasury) + float64(p.Land-3000)*price*0.8)
		p.Land = 3000
	}
	if p.Treasury > 0 {
		p.Land += int(float64(p.Treasury) / price)
		p.Treasury = 0
	}
	p.DeathYear -= 2
	p.RecalcArmy()
}

// TitleScore ist die Wertung für den Titelaufstieg (601–606).
func TitleScore(p *Player) int {
	parts := []float64{
		float64(p.Markets), float64(p.Palace), float64(p.Cathedral), float64(p.Mills),
		float64(p.Treasury) / 5000, float64(p.Land) / 6000, p.Burghers / 50,
		float64(p.Nobility) / 5, float64(p.Soldiers) / 50, float64(p.Clergy) / 10,
		float64(p.Population) / 2000, p.Trade / 5,
	}
	sum := 0
	for _, x := range parts {
		sum += int(math.Min(17, x))
	}
	return min(9, sum/9)
}

// YearEndReport beschreibt das Jahresende eines Herrschers.
type YearEndReport struct {
	Promoted    bool `json:"promoted"`
	NewTitle    int  `json:"newTitle"`
	Emperor     bool `json:"emperor"`
	TimePenalty int  `json:"timePenalty"`
	Died        bool `json:"died"`
	Interest    int  `json:"interest"`
}

// canPromote prüft die Zusatzbedingungen der hohen Titel (740–743).
func canPromote(p *Player) bool {
	switch {
	case p.Treasury < 0, p.Loan > 0: // Kaiser II: mit offenem Bankkredit kein Aufstieg
		return false
	case p.Title == 7 && p.Palace < 16:
		return false
	case p.Title == 8 && (p.Cathedral < 14 || p.Mills < 15 || p.Markets < 25 || p.Treasury < 100000):
		return false
	}
	return true
}

// EndYear: Titel, Zeitstrafe, Tod, Zins und Moralverfall (607–613, 718).
// overtime ist die Zeit über dem Limit in Sekunden (Remake: je Phase summiert).
// deathChance ist die Sterbechance je Jahr ab der Lebenserwartung (Sterblichkeit, 0 = unsterblich).
func EndYear(p *Player, b *Balance, rng *Rng, year int, turnSeconds, overtime int, deathChance float64) YearEndReport {
	var r YearEndReport
	if p.Title < TitleScore(p) && canPromote(p) {
		p.Title++
		r.Promoted, r.NewTitle = true, p.Title
		r.Emperor = p.Title >= 9
	}

	// Zeitstrafe: Original ab 80 s Zugdauer 1–3 Jahre, Remake 1 Jahr je angefangene Minute über Limit.
	if b.Classic {
		if turnSeconds >= 80 {
			r.TimePenalty = rng.IntN(3) + 1
		}
	} else if overtime > 0 {
		r.TimePenalty = (overtime + 59) / 60
	}
	if r.TimePenalty > 0 {
		p.DeathYear = max(year+1, p.DeathYear-r.TimePenalty)
	}

	// Tod ab Erreichen der Lebenserwartung: jedes Jahr mit deathChance (Original 50 %).
	if deathChance > 0 && year >= p.DeathYear {
		if rng.Float() >= deathChance {
			p.DeathYear++
		} else {
			p.DeathYear = 0
			r.Died = true
			return r
		}
	}

	// Zins: Original 10 % auf alles, Remake getrennt für Guthaben und Schulden.
	rate := 0.10
	if !b.Classic {
		rate = b.InterestOnSavings
		if p.Treasury < 0 {
			rate = b.InterestOnCredit
		}
	}
	before := p.Treasury
	p.Treasury = int(math.Floor(float64(p.Treasury) * (1 + rate)))
	r.Interest = p.Treasury - before
	p.Morale *= 0.9
	return r
}
