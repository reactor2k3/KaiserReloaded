package engine

// Building ist ein Staatskauf.
type Building string

const (
	Market    Building = "market"
	Mill      Building = "mill"
	Palace    Building = "palace"
	Cathedral Building = "cathedral"
)

// BuildingCost sind die festen Preise aus dem Original (575–578).
var BuildingCost = map[Building]int{Market: 1000, Mill: 2000, Palace: 5000, Cathedral: 9000}

// Build errichtet ein Gebäude bzw. einen Bauabschnitt, wenn genug Bauland da ist (592–600).
func Build(p *Player, b *Balance, rng *Rng, kind Building) error {
	plots := p.Land/1000 - 1
	switch kind {
	case Market:
		if p.Markets > plots {
			return ErrBuildingLand
		}
	case Mill:
		if p.Mills > plots {
			return ErrBuildingLand
		}
	case Palace:
		if p.Land < 13000 || p.Palace > 15 {
			return ErrBuildingLand
		}
	case Cathedral:
		if p.Land < 25000 || p.Cathedral > 13 {
			return ErrBuildingLand
		}
	default:
		return ErrInvalidAmount
	}
	if err := pay(p, b, float64(BuildingCost[kind])); err != nil {
		return err
	}
	switch kind {
	case Market:
		p.Markets++
		p.Trade++
	case Mill:
		p.Mills++
		// Original teilt N durch 4 – vermutlich Tippfehler für +1/4.
		if b.Classic {
			p.Trade /= 4
		} else {
			p.Trade += 0.25
		}
	case Palace:
		p.Palace++
		p.Trade += 0.5
	case Cathedral:
		p.Cathedral++
		p.Trade++
		p.Clergy += 1 + rng.IntN(6)
	}
	p.RecalcArmy()
	return nil
}

// Unit ist eine kaufbare Truppengattung (Miliz entsteht nur aus Bauten).
type Unit string

const (
	Cavalry   Unit = "cavalry"
	Artillery Unit = "artillery"
	Infantry  Unit = "infantry"
)

// DrawUnitPrices zieht die Grundpreise des Jahres (572).
func DrawUnitPrices(p *Player, rng *Rng) {
	p.Year.UnitPrices = UnitPrices{
		Cavalry:   3680 + rng.IntN(511),
		Artillery: 2300 + rng.IntN(511),
		Infantry:  1500 + rng.IntN(256),
	}
}

// UnitCost liefert den Preis einer Einheit; Söldner kosten 50 % mehr (652–654).
func UnitCost(p *Player, u Unit, mercenary bool) int {
	pr := p.Year.UnitPrices
	base, extra := 0, 0
	switch u {
	case Cavalry:
		base, extra = pr.Cavalry, 600
	case Artillery:
		base, extra = pr.Artillery, 400
	case Infantry:
		base, extra = pr.Infantry, 200
	}
	cost := base + extra
	if mercenary {
		cost += base / 2
	}
	return cost
}

// Recruit stellt eine Einheit à 20 Mann auf (664–672). Rekruten kommen aus dem Volk
// und brauchen mindestens 7 Einwohner je eigenem Soldaten; Söldner sind unbegrenzt.
func Recruit(p *Player, b *Balance, u Unit, mercenary bool) error {
	if u != Cavalry && u != Artillery && u != Infantry {
		return ErrInvalidAmount
	}
	if !mercenary && float64(p.Population)/float64(p.Soldiers-p.Mercenaries+1) <= 7 {
		return ErrTooManySoldier
	}
	if err := pay(p, b, float64(UnitCost(p, u, mercenary))); err != nil {
		return err
	}
	old := float64(p.Soldiers)
	switch u {
	case Cavalry:
		p.Cavalry++
	case Artillery:
		p.Artillery++
	case Infantry:
		p.Infantry++
	}
	if mercenary {
		p.Mercenaries += 20
	} else {
		p.Population -= 20
	}
	p.RecalcArmy()
	dilute(p, int(old), p.Soldiers) // Rekruten sind weder ausgebildet noch modern ausgerüstet
	// Neue Einheit bringt Moral 1,0 mit; Gesamtmoral nach Kopfzahl gemittelt.
	p.Morale = (p.Morale*old + 20) / (old + 20)
	return nil
}

// Maneuver kostet 4 Taler je Soldat plus 1000 und hebt die Moral um 0,1 (673).
func Maneuver(p *Player, b *Balance) error {
	if err := pay(p, b, float64(4*p.Soldiers+1000)); err != nil {
		return err
	}
	p.Morale += 0.1
	return nil
}
