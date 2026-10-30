package engine

import "math"

// Zufallsereignisse und Risiko-Entscheidungen nach Kaiser II (Eisermann/Schablitzki 1987,
// Public Domain). Ablauf wie im Vorbild:
// je Herrscher und Jahr 50 % Chance auf ein Ereignis, davon ein Viertel Risiko-Entscheidungen
// (Ja/Nein, Ausgang 50:50), sonst ein festes Ereignis. Beträge sind an die Wirtschaft des
// Remakes angepasst (Startkasse 15 000 statt rund 40 000 Taler) und wachsen mit dem Titel.
// Schiffsereignisse (Original T 38–41) treten nur ein, wenn passende Schiffe vorhanden sind.

// EventKind beschreibt, worauf ein festes Ereignis wirkt.
type EventKind string

const (
	EvTalerPlus  EventKind = "talerPlus"
	EvTalerMinus EventKind = "talerMinus"
	EvLandPlus   EventKind = "landPlus"
	EvLandMinus  EventKind = "landMinus"
	EvSoldPlus   EventKind = "soldatenPlus"
	EvSoldMinus  EventKind = "soldatenMinus"
	EvKornMinus  EventKind = "kornMinus"
	EvShipPlus   EventKind = "schiffePlus"    // Kolonialflotte bringt Prisen heim
	EvShipStorm  EventKind = "sturm"          // Schiffe im Heimathafen gehen verloren
	EvFleetLoss  EventKind = "flottenVerlust" // Kolonialflotten zerstreut, Schiffe verloren
)

// EventDef ist ein festes Ereignis mit Titel und Text.
type EventDef struct {
	ID    string
	Kind  EventKind
	Title string
	Text  string
}

// RiskDef ist eine Risiko-Entscheidung: Frage, guter und schlechter Ausgang, Folge bei „Nein“.
type RiskDef struct {
	ID       string
	Title    string
	Question string
	Good     string
	Bad      string
	NoText   string // leer = „Nein“ bleibt folgenlos
}

// Feste Ereignisse (Original T 14–44), Texte frei nacherzählt.
var Events = []EventDef{
	{"erbschaft", EvTalerPlus, "Eine Erbschaft", "Ein wohlhabender Großgrundbesitzer ist verstorben. Aus Dankbarkeit vermacht er Ihnen sein ganzes Vermögen."},
	{"auswanderer", EvTalerPlus, "Gruß aus der Neuen Welt", "Ein reicher Händler, dessen Großeltern aus Ihrem Land stammen, unterstützt die Heimat seiner Vorfahren mit einer großzügigen Spende."},
	{"spionage", EvTalerPlus, "Geheimnisse zu Gold", "Ihre Spione haben wertvolle militärische Geheimnisse beschafft, die Sie an die Großmächte Europas verkaufen konnten."},
	{"geheimpolizei", EvTalerPlus, "Schlag gegen die Opposition", "Die Geheimpolizei hat führende Oppositionelle festgenommen. Ihr Besitz fällt an den Staat."},
	{"unterschlagung", EvTalerPlus, "Verschwundene Gelder gefunden", "Durch Zufall wurden unterschlagene Einnahmen vergangener Jahre entdeckt und sogleich beschlagnahmt."},
	{"belohnung", EvTalerPlus, "Eine stattliche Belohnung", "Ihre Gendarmen haben einen von England gesuchten Revolutionär gefasst. Für seine Auslieferung erhalten Sie eine stattliche Belohnung."},
	{"portugal", EvTalerPlus, "Aufträge aus Portugal", "Portugal, reich geworden durch das Gold seiner Kolonien, lässt Schiffe auf Ihren Werften bauen. Ein gutes Geschäft."},
	{"altschulden", EvTalerPlus, "Alte Schulden beglichen", "Ein Nachbarstaat zahlt endlich Schulden zurück, die er noch zur Regierungszeit Ihres Vaters aufgenommen hatte."},
	{"bestechung", EvTalerPlus, "Bestechungsgelder beschlagnahmt", "Große Summen an Bestechungsgeldern wurden entdeckt und eingezogen. Die betroffenen Minister wurden sofort entlassen."},
	{"kaufleute", EvTalerPlus, "Kaufleute wandern zu", "Viele in anderen Ländern verfolgte Kaufleute haben sich bei Ihnen niedergelassen. Die Wirtschaft erlebt einen unerwarteten Aufschwung."},
	{"epidemie", EvTalerMinus, "Eine schwere Epidemie", "Eine Seuche hat auch Ihre Familie erfasst. Ärzte aus ganz Europa reisten an, und ihre Rechnungen sind entsprechend hoch."},
	{"erdbeben", EvTalerMinus, "Ein Erdbeben", "Mitten in der Nacht erschütterte ein seltenes Erdbeben Ihr Land. Der Wiederaufbau ist teuer."},
	{"finanzminister", EvTalerMinus, "Der Finanzminister ist fort", "Ihr Finanzminister hat Staatsgelder veruntreut und sich ins Ausland abgesetzt. Weder Geld noch Minister sind zurückzuholen."},
	{"kaiserabgabe", EvTalerMinus, "Abgabe für den Kaiser", "Da Sie dem Kaiser keine Truppen stellen, verlangt er eine gesonderte Abgabe für sein Heer."},
	{"kathedrale", EvTalerMinus, "Die Kathedrale bröckelt", "Die Mittel des Klerus reichen für die Restaurierung der Kathedrale Ihrer Hauptstadt bei Weitem nicht. Den Großteil zahlen Sie."},
	{"sabotage", EvTalerMinus, "Sabotage im Heer", "Saboteure haben Ihr Heer geschwächt. Reparaturen und die Jagd auf die Täter verschlingen beträchtliche Summen."},
	{"oper", EvTalerMinus, "Eine Oper für die Hauptstadt", "Zur Förderung der Künste war der Bau einer Oper unausweichlich. Ein Prachtbau, aber ein teurer."},
	{"tuerken", EvTalerMinus, "Die Türken im Südosten", "Zur Abwehr der türkischen Truppen im Südosten Europas müssen Sie einen Beitrag leisten."},
	{"winter", EvTalerMinus, "Ein harter Winter", "Ein langer, bitterer Winter bedrohte die Landbevölkerung. Um das Schlimmste zu verhindern, wurden die Hilfen vervielfacht."},
	{"vermaechtnis", EvLandPlus, "Ein Vermächtnis", "Ein entfernter Verwandter ist verstorben und vermacht Ihnen seine gesamten Ländereien."},
	{"papst", EvLandPlus, "Der Papst greift ein", "Der Papst hat den Herrscher eines Nachbarstaates exkommuniziert. Er musste abdanken, ein Teil seiner Besitzungen fällt Ihnen zu."},
	{"frankreich", EvLandMinus, "Frankreich besetzt Land", "Frankreich hat Teile Ihres Landes besetzt. Allein können Sie gegen die Großmacht nichts ausrichten."},
	{"separatisten", EvLandMinus, "Aufstand der Separatisten", "Einflussreiche Adlige wollen eine Region abtrennen und haben einen Aufstand angezettelt. Ihnen sind die Hände gebunden."},
	{"freiwillige", EvSoldPlus, "Freiwillige melden sich", "Eine erfreuliche Botschaft: Junge Männer aus allen Landesteilen melden sich freiwillig zum Dienst in Ihrem Heer."},
	{"fahnenflucht", EvSoldMinus, "Fahnenflucht", "Unzufrieden mit Sold und Verpflegung haben sich viele Soldaten in der Nacht davongemacht."},
	{"kornbrand", EvKornMinus, "Feuer im Kornspeicher", "Ein Großfeuer hat die Kornspeicher Ihrer Hauptstadt erfasst. Ein großer Teil der Vorräte ist verbrannt."},
	{"prisen", EvShipPlus, "Prisen auf hoher See", "Ihre Kolonialflotte hat fremde Kauffahrer aufgebracht und als Prisen in Ihre Häfen geführt."},
	{"freibeuter", EvShipPlus, "Freibeuter laufen über", "Eine Schar Freibeuter bietet Ihnen ihre Schiffe an und segelt fortan unter Ihrer Flagge."},
	{"sturm", EvShipStorm, "Sturm im Nordatlantik", "Ihre Flotte geriet im Nordatlantik in einen schweren Sturm. Einige Schiffe sind gesunken."},
	{"piraten", EvFleetLoss, "Piraten überfallen die Kolonien", "Piraten haben Ihre Kolonialflotten überfallen. Schiffe gingen verloren, die übrigen flüchteten in die Heimat."},
}

// Risiko-Entscheidungen (Original T 47–80), Texte frei nacherzählt.
var Risks = []RiskDef{
	{"england", "Eine Audienz in London", "Eine beinahe einmalige Gelegenheit: Wollen Sie den König von England aufsuchen, um ein Abkommen mit der Großmacht zu schließen?",
		"Der König war über Ihren Besuch hocherfreut und unterstützt Sie, um das Gleichgewicht in Europa zu wahren.",
		"Wegen Ihrer Verbindungen zu Frankreich lehnt der König jeden Kontakt ab. Die Kosten der aufwendigen Reise tragen Sie.", ""},
	{"revolution", "Revolution beim Nachbarn", "In einem Nachbarstaat tobt die Revolution. Wollen Sie der Restauration mit einem Kredit zu Hilfe kommen?",
		"Mit Ihrer Hilfe wurde die Revolution niedergeschlagen. Der alte und neue Herrscher zahlt zurück und zeigt sich erkenntlich.",
		"Trotz Ihrer Hilfe siegte die Revolution, Ihr Kredit ist verloren. Immerhin konnten Sie einen Teil rechtzeitig abziehen.", ""},
	{"bodenschaetze", "Bodenschätze", "Wollen Sie Mittel bereitstellen, um nach Bodenschätzen in Ihrem Land suchen und sie fördern zu lassen?",
		"Ein Erfolg! Erträge und der Verkauf von Minenrechten übertreffen die Kosten bei Weitem.",
		"Die Untersuchungen blieben ergebnislos, an Abbau ist nicht zu denken. Das Geld ist verloren.", ""},
	{"erbe", "Ein ungewisses Erbe", "Ein entfernter Verwandter vermacht Ihnen seinen ganzen Besitz. Wie es um seine Finanzen steht, weiß niemand. Wollen Sie das Erbe antreten?",
		"Die Bücher zeigen ein unerwartet gutes Bild: Sie erben große Ländereien und ein stattliches Vermögen!",
		"Wie befürchtet war Ihr Verwandter hoch verschuldet. Als Erbe müssen Sie für seine Schulden aufkommen.", ""},
	{"heirat", "Eine Hochzeit", "Sie sind in Ihren besten Jahren, die Zeit ist reif für eine Ehe. Wollen Sie mit dem Spross einer Patrizierfamilie den Bund der Ehe schließen?",
		"Ihre Hochzeit wurde im ganzen Land gefeiert. Sie leben in Glück und Zufriedenheit, nicht zuletzt dank der hohen Mitgift.",
		"Die Ehe ist glücklich, doch die Feier mit Gästen aus ganz Europa kostete weit mehr, als die Mitgift einbrachte.", ""},
	{"universitaet", "Eine Universität in Not", "Eine Universität Ihres Landes ist in finanzielle Not geraten. Wollen Sie sie übernehmen und unter staatlicher Leitung weiterführen?",
		"Ein Glücksgriff: Mit Erfindungen aus der Universität erzielen Sie Gewinne, die alle Kosten mehr als aufwiegen.",
		"Die Universität erweist sich als Fass ohne Boden. Schulden und Unterhalt übersteigen jeden Nutzen.", ""},
	{"spanien", "Spanien bittet um Kredit", "Spaniens Finanzen sind durch Fehlspekulationen zerrüttet, man bittet Sie um Hilfe. Wollen Sie ein Kreditgeschäft mit Spanien wagen?",
		"Spanien hat sich erholt und zahlt Ihr Geld samt stattlicher Zinsen zurück.",
		"Spanien musste trotz Ihrer Hilfe die Zahlungsunfähigkeit erklären. Ihr Geld ist verloren.", ""},
	{"konkordat", "Ein Konkordat mit Rom", "Wollen Sie, um die Rückendeckung des Vatikans zu gewinnen, ein Konkordat schließen und die katholische Kirche zur Staatsreligion erklären?",
		"Trotz kleiner Proteste wurde das Konkordat geschlossen. Rom zeigt sich mit einer großzügigen Summe erkenntlich.",
		"Der Vertrag löste einen Aufstand der Protestanten aus. Sie mussten ihn aufheben und für die Schäden aufkommen.", ""},
	{"aerzte", "Sie sind erkrankt", "Sie sind ernsthaft erkrankt, wie es ausgeht, ist ungewiss. Wollen Sie die besten Ärzte Europas kommen lassen?",
		"Die Investition hat sich gelohnt! Die Gelehrten behandelten Sie mit erstaunlichem Erfolg, Sie fühlen sich um Jahre verjüngt.",
		"Die teuren Ärzte konnten wenig ausrichten. Sie genesen, doch die Krankheit hat Spuren hinterlassen.",
		"Ohne Ärzte genasen Sie zwar recht schnell, doch einige Folgen der Krankheit werden Sie nie ganz los."},
	{"hospital", "Ein Hospital", "Ihre Ratgeber drängen auf den Bau eines Hospitals zum Wohl aller Untertanen. Sind Sie einverstanden?",
		"Die Kosten blieben im Rahmen, und Ihre Untertanen danken es Ihnen: Viele kommen ins Land, um hier zu leben.",
		"Der Bau verschlang mehr als geplant, und der erhoffte Zuspruch blieb aus.",
		"Oppositionelle Adlige nutzen Ihre Weigerung, um das Volk gegen Sie aufzubringen. Etliche Untertanen verlassen das Land."},
}

// RiskByID sucht eine Risiko-Entscheidung.
func RiskByID(id string) *RiskDef {
	for i := range Risks {
		if Risks[i].ID == id {
			return &Risks[i]
		}
	}
	return nil
}

// EventReport ist das Ergebnis eines Ereignisses für Chronik und Modal.
type EventReport struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Taler    int    `json:"taler,omitempty"`
	Land     int    `json:"land,omitempty"`
	Soldiers int    `json:"soldiers,omitempty"` // Mann (Einheiten × 20)
	Grain    int    `json:"grain,omitempty"`
	People   int    `json:"people,omitempty"`
	Years    int    `json:"years,omitempty"` // Änderung der Lebenserwartung
	Ships    int    `json:"ships,omitempty"` // gewonnene oder verlorene Schiffe
	Risk     bool   `json:"risk,omitempty"`  // Ausgang einer Risiko-Entscheidung
}

// scale: Beträge wachsen mit dem Titel (Baron +25 %, …, Kaiser +200 %).
func scale(p *Player, base float64) int {
	return int(math.Round(base * (1 + float64(p.Title-1)*0.25)))
}

// RollEvent würfelt das Jahresereignis eines Herrschers: nil (kein Ereignis), ein festes
// Ereignis (sofort angewandt) oder eine offene Risiko-Entscheidung (risk != "").
func (g *Game) RollEvent(seat int) (rep *EventReport, risk string) {
	if g.Balance.Classic || g.Rng.Float() >= 0.5 {
		return nil, ""
	}
	if g.Rng.Float() < 0.25 {
		return nil, Risks[g.Rng.IntN(len(Risks))].ID
	}
	// Schiffsereignisse nur mit passenden Schiffen (Original würfelt dann neu)
	ev := Events[g.Rng.IntN(len(Events))]
	for !g.eventFits(seat, ev) {
		ev = Events[g.Rng.IntN(len(Events))]
	}
	return g.applyEvent(seat, ev), ""
}

// applyEvent setzt die Wirkung eines festen Ereignisses um.
func (g *Game) applyEvent(seat int, ev EventDef) *EventReport {
	p := g.Seats[seat].Player
	r := &EventReport{ID: ev.ID, Title: ev.Title, Text: ev.Text}
	f := g.Rng.Float()
	switch ev.Kind {
	case EvTalerPlus:
		r.Taler = scale(p, 2500+5000*f)
	case EvTalerMinus:
		r.Taler = -scale(p, 2000+4000*f)
	case EvLandPlus:
		r.Land = int(float64(p.Land) * (0.05 + 0.10*f))
		if free := g.FreeLand(p.Country); free >= 0 {
			r.Land = min(r.Land, free) // geteilte Länder: nur freie Fläche
		}
	case EvLandMinus:
		r.Land = -int(float64(p.Land) * (0.04 + 0.08*f))
	case EvSoldPlus:
		r.Soldiers = 20 * (2 + g.Rng.IntN(5))
	case EvSoldMinus:
		r.Soldiers = -20 * max(1, p.Infantry/10+g.Rng.IntN(2))
	case EvKornMinus:
		r.Grain = -int(float64(p.Grain) * (0.3 + 0.3*f))
	case EvShipPlus: // 2 + Zufall(Kolonialschiffe / 5), Anteil Kriegsschiffe wie in der Flotte
		r.Ships = 2 + int(float64(fleetSize(p))/5*f)
		war := int(float64(r.Ships) * float64(p.Warships) / float64(max(1, p.Traders+p.Warships)))
		p.Warships += war
		p.Traders += r.Ships - war
	case EvShipStorm: // Verluste im Heimathafen
		ht, hw := shipsAtHome(p)
		r.Ships = -min(ht+hw, 2+int(float64(ht+hw)/5*f))
		lostT := min(ht, int(math.Round(float64(-r.Ships)*float64(ht)/float64(max(1, ht+hw)))))
		p.Traders -= lostT
		p.Warships -= -r.Ships - lostT
	case EvFleetLoss: // alle Kolonialflotten heim, 2 + Zufall(Kolonialschiffe / 7) verloren
		col := fleetSize(p)
		r.Ships = -min(col, 2+int(float64(col)/7*f))
		lostT := min(p.Traders, int(math.Round(float64(-r.Ships)*float64(p.Traders)/float64(max(1, p.Traders+p.Warships)))))
		p.Traders -= lostT
		p.Warships -= -r.Ships - lostT
		for i := range p.Colonies {
			p.Colonies[i].Traders, p.Colonies[i].Warships = 0, 0
		}
	}
	fixFleet(p)
	g.applyReport(p, r)
	return r
}

// applyReport bucht die Folgen eines Berichts auf den Herrscher.
func (g *Game) applyReport(p *Player, r *EventReport) {
	p.Treasury += r.Taler
	p.Land = max(1000, p.Land+r.Land)
	p.Grain = max(0, p.Grain+r.Grain)
	p.Population = max(g.Balance.MinPopulation, p.Population+r.People)
	if r.Soldiers != 0 {
		units := r.Soldiers / 20
		if units < 0 {
			units = -min(-units, p.Infantry)
			r.Soldiers = units * 20
		}
		p.Infantry += units
		p.RecalcArmy()
	}
	if r.Years != 0 && p.DeathYear > 0 {
		p.DeathYear = max(g.Year+1, p.DeathYear+r.Years)
	}
}

// AnswerRisk löst eine offene Risiko-Entscheidung auf: „Ja“ gewinnt oder verliert mit 50 %,
// „Nein“ bleibt folgenlos (außer bei Krankheit und Hospital).
func (g *Game) AnswerRisk(seat int, yes bool) *EventReport {
	p := g.Seats[seat].Player
	def := RiskByID(p.Year.Risk)
	p.Year.Risk = ""
	if def == nil {
		return nil
	}
	r := &EventReport{ID: def.ID, Title: def.Title, Risk: true}
	if !yes {
		if def.NoText == "" {
			return nil
		}
		r.Text = def.NoText
		switch def.ID {
		case "aerzte":
			r.Years = -(1 + g.Rng.IntN(2))
		case "hospital":
			r.People = -int(float64(p.Population) * (0.02 + 0.03*g.Rng.Float()))
		}
		g.applyReport(p, r)
		return r
	}
	good := g.Rng.Float() < 0.5
	f := g.Rng.Float()
	if good {
		r.Text = def.Good
	} else {
		r.Text = def.Bad
	}
	switch def.ID {
	case "aerzte":
		r.Taler = -scale(p, 1500+1500*f)
		if good {
			r.Years = 4 + g.Rng.IntN(4)
		} else {
			r.Years = -1
		}
	case "hospital":
		r.Taler = -scale(p, 2000+2000*f)
		if good {
			r.People = int(float64(p.Population) * (0.04 + 0.04*g.Rng.Float()))
		}
	case "erbe":
		if good {
			r.Taler = scale(p, 6000+6000*f)
			r.Land = int(float64(p.Land) * (0.06 + 0.06*g.Rng.Float()))
			if free := g.FreeLand(p.Country); free >= 0 {
				r.Land = min(r.Land, free)
			}
		} else {
			r.Taler = -scale(p, 4000+4000*f)
		}
	default:
		// Geldgeschäfte: Gewinn deutlich höher als Verlust, wie im Original (+15 000 / −4 500 …)
		if good {
			r.Taler = scale(p, 5000+7000*f)
		} else {
			r.Taler = -scale(p, 3000+4000*f)
		}
	}
	g.applyReport(p, r)
	return r
}

// eventFits: Schiffsereignisse setzen Kolonialflotten bzw. mindestens zwei Schiffe im Hafen voraus.
func (g *Game) eventFits(seat int, ev EventDef) bool {
	p := g.Seats[seat].Player
	switch ev.Kind {
	case EvShipPlus, EvFleetLoss:
		return fleetSize(p) > 0
	case EvShipStorm:
		ht, hw := shipsAtHome(p)
		return ht+hw >= 2
	}
	return true
}
