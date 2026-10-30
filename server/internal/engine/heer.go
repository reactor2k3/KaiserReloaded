package engine

// Ausbildung und Material des Heeres nach dem Vorbild Kaiser II (Armee-Menü: „Ausbildung steigern“,
// „Modernisieren“; Schlagkraft aus Soldaten, Moral, Ausbildung, Material:
// Soldaten × (80 + Moral + 2 × Ausbildung + 3 × Material)).
//
// Remake: Ausbildung und Material je 0–10. Sie verstärken jede eigene Einheit im Gefecht:
//   Schlagkraft-Faktor = 1 + 3 % × Ausbildung + 5 % × Material  (höchstens 1,8)
// Kosten je Stufe: Ausbildung (Stufe+1) × 400 + 2 Taler je Soldat, Material (Stufe+1) × 600 + 4 je Soldat.
// Neue Rekruten verdünnen beides (Mittel nach Kopfzahl), Ausbildung verfällt jährlich mit 30 % um 1.

const MaxDrill = 10

// Strength ist der Schlagkraft-Faktor des Heeres im Gefecht.
func Strength(p *Player) float64 {
	return 1 + 0.03*float64(p.Training) + 0.05*float64(p.Equipment)
}

// TrainCost und EquipCost: Preis der nächsten Stufe.
func TrainCost(p *Player) int { return (p.Training+1)*400 + 2*p.Soldiers }
func EquipCost(p *Player) int { return (p.Equipment+1)*600 + 4*p.Soldiers }

var errDrillMax = &RuleError{"drill_max", "Höher geht es nicht."}

// Train steigert die Ausbildung um eine Stufe (Bauphase).
func Train(p *Player, b *Balance) error {
	if p.Training >= MaxDrill {
		return errDrillMax
	}
	if err := pay(p, b, float64(TrainCost(p))); err != nil {
		return err
	}
	p.Training++
	return nil
}

// Equip modernisiert das Material um eine Stufe (Bauphase).
func Equip(p *Player, b *Balance) error {
	if p.Equipment >= MaxDrill {
		return errDrillMax
	}
	if err := pay(p, b, float64(EquipCost(p))); err != nil {
		return err
	}
	p.Equipment++
	return nil
}

// dilute mittelt Ausbildung und Material nach Kopfzahl, wenn neue Soldaten dazukommen.
func dilute(p *Player, oldSoldiers, newSoldiers int) {
	if newSoldiers <= 0 {
		return
	}
	p.Training = (p.Training*oldSoldiers + newSoldiers/2) / newSoldiers
	p.Equipment = (p.Equipment*oldSoldiers + newSoldiers/2) / newSoldiers
}
