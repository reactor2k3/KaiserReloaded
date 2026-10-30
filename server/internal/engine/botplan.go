package engine

import "math"

// Planung der Computer-Herrscher: statt fester Schwellen rechnen die Bots mit der Titelwertung
// (TitleScore), einer Rücklage und den Bedingungen der hohen Titel (canPromote).
//
//   - Rücklage: Geld, das für Korn, Sold und Unvorhergesehenes in der Kasse bleibt.
//   - Kein Kauf, der den sonst sicheren Aufstieg dieses Jahres kostet (Kasse zählt mit Kasse/5000).
//   - Zielfläche: genug Bauland für Märkte und Mühlen, ab Graf 13 000 ha (Palast), ab Fürst
//     26 000 ha (Kathedrale und 25 Märkte für die Kaiserkrone).
//   - Korn: Wer mehr als 110–150 % des Bedarfs verteilt, lockt Einwanderer, Adel und Klerus an
//     (Feed in economy.go). Die Bots kaufen dafür zu, solange der Preis vertretbar ist.
//   - Krieg nur, wenn die eigene Stärke den Gegner samt seinen Verbündeten deutlich übertrifft;
//     Ziel ist der Gegner mit dem meisten Land unter diesen.

// botReserve: Mindestkasse nach Ausgaben in der Bauphase.
func botReserve(p *Player) int {
	pay := p.Soldiers*3 + p.Mercenaries*12 + p.Gendarmes*GendarmPay + p.Traders*TraderUpkeep + p.Warships*WarshipUpkeep
	return 2000*p.Title + pay
}

// spare: frei verfügbares Geld über der Rücklage.
func spare(p *Player) int { return p.Treasury - botReserve(p) }

// scorePoints entspricht TitleScore ohne die Teilung durch 9 (Summe der gedeckelten Punkte).
func scorePoints(p *Player) int {
	parts := []float64{
		float64(p.Markets), float64(p.Palace), float64(p.Cathedral), float64(p.Mills),
		float64(p.Treasury) / 5000, float64(p.Land) / 6000, p.Burghers / 50,
		float64(p.Nobility) / 5, float64(p.Soldiers) / 50, float64(p.Clergy) / 10,
		float64(p.Population) / 2000, p.Trade / 5,
	}
	sum := 0
	for _, x := range parts {
		sum += int(math.Min(17, math.Max(0, x)))
	}
	return sum
}

// costsPromotion: Würde eine Ausgabe den Aufstieg dieses Jahres verhindern? apply ändert eine
// Kopie des Herrschers so, wie es die Aktion täte.
func costsPromotion(p *Player, apply func(q *Player)) bool {
	if p.Title >= 9 || !canPromote(p) || scorePoints(p)/9 <= p.Title {
		return false
	}
	q := *p
	apply(&q)
	return scorePoints(&q)/9 <= p.Title || !canPromote(&q)
}

// targetLand: Fläche, auf die ein Bot hinarbeitet.
func targetLand(p *Player) int {
	t := max(p.Population*12/10, (p.Markets+p.Mills/2+3)*1000)
	switch {
	case p.Title >= 6:
		t = max(t, 26000)
	case p.Title >= 4:
		t = max(t, 14000)
	}
	return t
}

// botSkill: Spielstärke der Computer-Herrscher (Lobby-Einstellung Difficulty).
//   - FeedMax: großzügigste Kornverteilung (Einwanderung erst ab 110–150 % des Bedarfs).
//   - Idle: Anteil der Jahre, in denen der Bot in der Bauphase nur das Nötigste tut.
//   - Plan: rechnet mit der Titelwertung und baut Märkte und Mühlen auf Kredit.
//   - MaxInvest: höchstens so viele Bau-, Werbe- und Drillaufträge je Jahr (0 = unbegrenzt).
type botSkill struct {
	FeedMax   float64
	Idle      float64
	Plan      bool
	MaxInvest int
}

var botSkills = map[string]botSkill{
	"leicht": {FeedMax: 1.1, Idle: 0.5, Plan: false, MaxInvest: 2},
	"mittel": {FeedMax: 1.25, Idle: 0.3, Plan: true, MaxInvest: 2},
	"schwer": {FeedMax: 1.5, Idle: 0, Plan: true},
}

// skill liefert die Spielstärke eines Sitzes; der Autopilot menschlicher Spieler spielt immer voll.
func (g *Game) skill(seat int) botSkill {
	if sk, ok := botSkills[g.Settings.Difficulty]; ok && g.isCPU(seat) {
		return sk
	}
	return botSkills["schwer"]
}

// idleYear: In diesem Jahr trödelt der Bot (fest je Sitz und Jahr, damit es nicht bei jeder
// einzelnen Aktion neu gewürfelt wird).
func (g *Game) idleYear(seat int) bool {
	return float64((seat*7919+g.Year*104729)%100) < g.skill(seat).Idle*100
}

// feedTarget: gewünschte Verteilung. Großzügig (Einwanderung), wenn die Kasse nicht klemmt.
func feedTarget(p *Player, st BotStrategy, sk botSkill) int {
	need := float64(p.Year.GrainNeed)
	ratio := st.FeedRatio
	if p.Treasury > 0 {
		ratio = max(ratio, sk.FeedMax)
	}
	return int(need * ratio)
}

// botMarket: Risiko, Korn, Land, Kredit und Anleihen in der Marktphase.
func (g *Game) botMarket(seat int, st BotStrategy) Action {
	p := g.Seats[seat].Player
	need := p.Year.GrainNeed
	if p.Year.Risk != "" {
		// Krankheit und Hospital immer annehmen, sonst nur mit gutem Polster
		// (aber nicht, wenn die Kasse dafür ins tiefe Minus müsste – Bankrottgefahr)
		yes := ((p.Year.Risk == "aerzte" || p.Year.Risk == "hospital") && p.Treasury > 3000*p.Title) || spare(p) > 15000
		return Action{Type: "risk", Stance: map[bool]string{true: "ja", false: "nein"}[yes]}
	}
	left := 3*need - p.Year.GrainBought
	// Pflicht: Vorrat, um den vollen Bedarf verteilen zu können (höchstens 80 % der Reserve),
	// notfalls auf Kredit – sonst Hunger und Unruhe. Mehr nur, wenn der Preis es zulässt (Kür).
	if want := int(float64(need)*1.05/0.8) - p.Grain; want > 0 && left > 0 && p.Treasury > creditLimit(p, g.Balance)/2 {
		return Action{Type: "buyGrain", Qty: min(want, left)}
	}
	// Kür: Vorrat für die Einwanderung (Verteilung 150 %, höchstens 80 % der Reserve), wenn bezahlbar
	if want := int(float64(feedTarget(p, st, g.skill(seat)))/0.8) + need/10 - p.Grain; want > 0 && left > 0 {
		if cost := want * p.Year.GrainPrice / 1000; cost < spare(p)/3 {
			return Action{Type: "buyGrain", Qty: min(want, left)}
		}
	}
	// Überschuss über dem Dreifachen verkaufen (verdirbt sonst)
	if extra := p.Grain - 3*need; extra > need/2 {
		return Action{Type: "sellGrain", Qty: extra}
	}
	free := g.FreeLand(p.Country)
	buyable := func(q int) int {
		if free >= 0 {
			q = min(q, free)
		}
		return q
	}
	// Pflicht: mehr Hektar als Einwohner (sonst Amtsenthebung)
	if gap := p.Population*12/10 - p.Land; gap > 0 && p.Treasury > creditLimit(p, g.Balance)/2 {
		if q := buyable(gap + 500); q > 0 {
			return Action{Type: "buyLand", Qty: q}
		}
	}
	// Ziel-Fläche in Schritten, soweit das freie Geld reicht und der Aufstieg nicht leidet
	// (ohne Planung erst mit voller Kasse)
	if gap := targetLand(p) - p.Land; gap > 0 && p.Year.LandPrice > 0 && (g.skill(seat).Plan || p.Treasury > 30000) {
		q := buyable(min(gap, 3000, int(float64(spare(p)/2)/p.Year.LandPrice)))
		cost := int(float64(q) * p.Year.LandPrice)
		if q >= 200 && !costsPromotion(p, func(c *Player) { c.Treasury -= cost; c.Land += q }) {
			return Action{Type: "buyLand", Qty: q}
		}
	}
	if !g.Balance.Classic {
		if p.Loan > 0 && p.Treasury > 8000 {
			return Action{Type: "repayLoan", Qty: min(p.Loan, p.Treasury-6000)}
		}
		// Reichsanleihen nur mit echtem Überschuss und wenn die Kasse für den Titel nicht zählt
		if !st.Military && p.ReichBonds < 200 && p.Treasury > 100000 {
			if q := min(200-p.ReichBonds, (p.Treasury-90000)/ReichBondPrice); q > 0 {
				return Action{Type: "buyReichBonds", Qty: q}
			}
		}
	}
	return Action{Type: "next"}
}

// hostilePower: stärkste Kampfkraft eines Nicht-Verbündeten, der schon Krieg führen darf
// (ab Baron) – nur das ist eine echte Bedrohung für die Verteidigung.
func (g *Game) hostilePower(seat int) float64 {
	best := 0.0
	for i, s := range g.Seats {
		if i != seat && s.Player.Alive() && !g.Allied(seat, i) && s.Player.Title >= 2 {
			best = max(best, power(s.Player)*Strength(s.Player))
		}
	}
	return best
}

// defenseOf: Kampfkraft eines Herrschers samt der Verbündeten, die ihm automatisch beistehen.
func (g *Game) defenseOf(seat int) float64 {
	p := g.Seats[seat].Player
	v := power(p) * Strength(p)
	for _, a := range g.Allies(seat) {
		if q := g.Seats[a].Player; q.Alive() {
			v += power(q) * Strength(q)
		}
	}
	return v
}

// botBuild: Wirtschaft, Heer, Bündnisse, Kolonien und Krieg in der Bauphase. ok = false heißt
// „Zug beenden“.
func (g *Game) botBuild(seat int, st BotStrategy) (a Action, ok bool) {
	p := g.Seats[seat].Player
	plots := p.Land/1000 - 1
	canRecruit := float64(p.Population)/float64(p.Soldiers-p.Mercenaries+1) > 7
	free := spare(p)
	sk := g.skill(seat)
	// Ohne Planung (leicht) wird nicht auf die Titelwertung geachtet
	keepsPromotion := func(apply func(q *Player)) bool { return !sk.Plan || !costsPromotion(p, apply) }
	afford := func(cost int, apply func(q *Player)) bool {
		return cost <= free && keepsPromotion(func(q *Player) { q.Treasury -= cost; apply(q) })
	}
	// Märkte und Mühlen verzinsen sich mit rund 19 % im Jahr, besser als der Dispo (10 %) kostet:
	// Sie dürfen auch auf Kredit gebaut werden, bis 2000 Taler je Titelstufe ins Minus.
	invest := func(cost int, apply func(q *Player)) bool {
		if !sk.Plan {
			return afford(cost, apply)
		}
		// Die ersten drei Märkte sind der Weg aus der Armut: dafür bis 4000 Taler je Titelstufe ins Minus
		floor := -2000 * p.Title
		if p.Markets < 3 {
			floor = -4000 * p.Title
		}
		return p.Treasury-cost >= floor && keepsPromotion(func(q *Player) { q.Treasury -= cost; apply(q) })
	}

	// 1. Verteidigung: nicht wehrlos gegenüber dem stärksten Nicht-Verbündeten (Kavallerie wehrt am besten ab)
	want := 0.6
	if st.Military {
		want = 1.0
	}
	if canRecruit && power(p)*Strength(p) < want*g.hostilePower(seat) && p.Soldiers < p.Population/10 && p.Markets >= min(plots, 5) {
		if c := UnitCost(p, Cavalry, false); c <= free {
			return Action{Type: "recruit", Unit: "cavalry"}, true
		}
	}

	// 2. Bündnisangebote beantworten (nicht vom stärksten Rivalen)
	if offers := g.OffersTo(seat); len(offers) > 0 {
		return Action{Type: map[bool]string{true: "acceptPact", false: "declinePact"}[offers[0] != g.strongestRivalSeat(seat)], Target: offers[0]}, true
	}

	// Moral unter 35 % lädt zum Militärputsch ein: Manöver geht immer vor (auch in Trödeljahren)
	if p.Soldiers > 100 && p.Morale < 0.35 && p.Treasury-(4*p.Soldiers+1000) > creditLimit(p, g.Balance)/2 {
		return Action{Type: "maneuver"}, true
	}

	// Trödeljahr (leicht/mittel): nur Verteidigung und Antworten, keine Investitionen
	if g.idleYear(seat) || (sk.MaxInvest > 0 && p.Year.BotInvest >= sk.MaxInvest) {
		return Action{}, false
	}
	// Investitionen mitzählen (Grenze je Jahr für leichte und mittlere Gegner)
	defer func() {
		switch a.Type {
		case "build", "recruit", "train", "equip", "buildShip", "colonize":
			p.Year.BotInvest++
		}
	}()

	// 3. Bedingungen der hohen Titel zuerst (Palast für Fürst → König, Kathedrale & Co. für Kaiser)
	if p.Title >= 7 && p.Palace < 16 && p.Land >= 13000 && afford(BuildingCost[Palace], func(q *Player) { q.Palace++ }) {
		return Action{Type: "build", Building: "palace"}, true
	}
	if p.Title >= 8 {
		switch {
		case p.Cathedral < 14 && p.Land >= 25000 && afford(BuildingCost[Cathedral], func(q *Player) { q.Cathedral++ }):
			return Action{Type: "build", Building: "cathedral"}, true
		case p.Markets < 25 && p.Markets <= plots && afford(BuildingCost[Market], func(q *Player) { q.Markets++ }):
			return Action{Type: "build", Building: "market"}, true
		case p.Mills < 15 && p.Mills <= plots && afford(BuildingCost[Mill], func(q *Player) { q.Mills++ }):
			return Action{Type: "build", Building: "mill"}, true
		}
	}

	// 4. Rendite: Märkte (je 127–254 Taler im Jahr, +1 Handel) vor Mühlen (250–500, binden 100 Arbeiter)
	if p.Markets <= plots && p.Markets < 25 && invest(BuildingCost[Market], func(q *Player) { q.Markets++ }) {
		return Action{Type: "build", Building: "market"}, true
	}
	workers := p.Population - 100*(p.Mills+1)
	if p.Mills <= plots && p.Mills < p.Markets && workers*5 >= p.Land/2 && invest(BuildingCost[Mill], func(q *Player) { q.Mills++ }) {
		return Action{Type: "build", Building: "mill"}, true
	}

	// 5. Titelpunkte: Palast und Kathedrale bringen je einen Punkt (Kathedrale zusätzlich Klerus)
	if p.Land >= 13000 && p.Palace < 16 && afford(BuildingCost[Palace], func(q *Player) { q.Palace++ }) {
		return Action{Type: "build", Building: "palace"}, true
	}
	if p.Land >= 25000 && p.Cathedral < 14 && afford(BuildingCost[Cathedral], func(q *Player) { q.Cathedral++ }) {
		return Action{Type: "build", Building: "cathedral"}, true
	}

	// 6. Heer: Feldherren bauen auf (Soldaten zählen je 50 einen Titelpunkt), üben und rüsten
	if st.Military {
		if p.Soldiers >= 60 && p.Morale < 0.75 && (g.Year%3 == 0 || p.Morale < 0.55) &&
			afford(4*p.Soldiers+1000, func(q *Player) { q.Morale += 0.1 }) {
			return Action{Type: "maneuver"}, true
		}
		// Aufrüsten lohnt nur, wenn ein Gegner in Reichweite ist (samt Verbündeten höchstens 1,6-fach
		// so stark); sonst genügt ein kleineres Heer und das Geld fließt in die Wirtschaft
		limit := p.Population / 20
		if g.warTargetInReach(seat, 1.6) {
			limit = p.Population / 12
		}
		if canRecruit && p.Soldiers < limit && p.Markets >= min(plots, 8) {
			kinds := []Unit{Cavalry, Infantry, Artillery}
			u := kinds[(p.Cavalry+p.Infantry+p.Artillery)%3]
			if afford(UnitCost(p, u, false), func(q *Player) { q.Soldiers += 20 }) {
				return Action{Type: "recruit", Unit: string(u)}, true
			}
		}
		if p.Soldiers > 100 && p.Training < 5 && afford(TrainCost(p), func(*Player) {}) && free > 3*TrainCost(p) {
			return Action{Type: "train"}, true
		}
		if p.Soldiers > 100 && p.Equipment < 5 && afford(EquipCost(p), func(*Player) {}) && free > 3*EquipCost(p) {
			return Action{Type: "equip"}, true
		}
	} else if p.Soldiers >= 60 && p.Morale < 0.5 && afford(4*p.Soldiers+1000, func(q *Player) { q.Morale += 0.1 }) {
		return Action{Type: "maneuver"}, true
	}

	// 7. Bündnisse: Kaufleute suchen bis zu zwei Partner gegen Angreifer, Feldherren einen
	// Waffenbruder, damit nicht alle anderen sich gegen sie verbünden
	maxAllies := 2
	if st.Military {
		maxAllies = 1
	}
	if len(g.Allies(seat)) < maxAllies && g.Year%4 == seat%4 {
		for other, o := range g.Seats {
			if other != seat && o.Player.Alive() && !g.Allied(seat, other) && !g.offered(seat, other) {
				return Action{Type: "proposePact", Target: other}, true
			}
		}
	}

	// 8. Kolonien und Flotte
	if act, ok := g.botColonies(seat, st); ok {
		return act, true
	}

	// 9. Krieg: Ziel mit dem meisten Land, das samt Verbündeten deutlich schwächer ist
	if st.Military && p.Title >= 2 && g.Year%3 == 0 && p.Soldiers >= 100 && p.Treasury > 3000 {
		mine := power(p) * Strength(p)
		target, bestLand := -1, 0
		for other, o := range g.Seats {
			if other == seat || !o.Player.Alive() || g.warAllowed(seat, other) != nil {
				continue
			}
			if mine > 1.25*g.defenseOf(other) && o.Player.Land > bestLand {
				target, bestLand = other, o.Player.Land
			}
		}
		if target >= 0 {
			return Action{Type: "declareWar", Target: target}, true
		}
	}
	return Action{}, false
}

// warTargetInReach: Gibt es einen angreifbaren Gegner, dessen Abwehr samt Verbündeten höchstens
// factor-mal so stark ist wie das eigene Heer?
func (g *Game) warTargetInReach(seat int, factor float64) bool {
	p := g.Seats[seat].Player
	mine := power(p) * Strength(p)
	for other, o := range g.Seats {
		if other != seat && o.Player.Alive() && g.warAllowed(seat, other) == nil && g.defenseOf(other) <= factor*mine {
			return true
		}
	}
	return false
}
