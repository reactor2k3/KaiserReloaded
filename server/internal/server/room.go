package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"kaiserreloaded/server/internal/engine"
	"kaiserreloaded/server/internal/store"
)

// Room ist ein Spiel mit seinen Verbindungen. Alle Zugriffe laufen über inbox
// in der Room-Goroutine, deshalb braucht der Spielzustand keine Locks.
type Room struct {
	id       string
	game     *engine.Game
	tokens   []string // Token je Sitz
	conns    map[*Conn]int
	started  bool
	inbox    chan func()
	last     []engine.PublicPlayer // zuletzt gesendete öffentliche Sicht, für Deltas
	now      func() int64
	store    *store.Store            // nil = ohne Speicherung (Tests)
	pausedAt int64                   // Beginn der Ruhe ohne verbundene Spieler (Unix), 0 = läuft
	banned   map[string]bool         // IPs, die der Gastgeber hinausgeworfen hat (nur im Speicher)
	denied   map[string]int          // abgelehnte Anfragen je IP (ab maxDenials gesperrt)
	requests []*joinRequest          // offene Beitrittsanfragen (beitritt.go)
	list     atomic.Pointer[listing] // Eintrag der öffentlichen Spieleliste, vom Hub gelesen

	// Für die Grenze „laufende Spiele“: vom Hub aus atomar lesbar, gesetzt in der Room-Goroutine
	touched   atomic.Int64 // letzte Änderung (Unix)
	isOver    atomic.Bool
	isStarted atomic.Bool
}

// counts meldet, ob das Spiel als laufend zählt (offene Lobby bis 6 h, Spiel bis 14 Tage nach
// dem letzten Zug; beendete Spiele nie).
func (r *Room) counts(now time.Time) bool {
	if r.isOver.Load() {
		return false
	}
	age := now.Sub(time.Unix(r.touched.Load(), 0))
	if r.isStarted.Load() {
		return age < gameActiveFor
	}
	return age < lobbyActiveFor
}

// mark aktualisiert die Kennzahlen (aus der Room-Goroutine).
func (r *Room) mark() {
	r.touched.Store(r.now())
	r.isOver.Store(r.game.Over)
	r.isStarted.Store(r.started)
	r.updateListing()
}

func newRoom(id string, s engine.Settings, b *engine.Balance, w *engine.World, st *store.Store) *Room {
	r := &Room{
		id:    id,
		game:  engine.NewGame(s, b, rand.Uint64()),
		conns: map[*Conn]int{},
		inbox: make(chan func(), 64),
		now:   func() int64 { return time.Now().Unix() },
		store: st,
	}
	r.game.World = w
	r.mark()
	go r.run()
	return r
}

// restoreRoom lädt ein gespeichertes Spiel. Laufende Phasen bekommen ihre Zugzeit neu,
// damit niemand für die Ausfallzeit des Servers bestraft wird.
func restoreRoom(rec store.Record, st *store.Store, w *engine.World) (*Room, error) {
	g, err := engine.RestoreGame(rec.Snapshot)
	if err != nil {
		return nil, err
	}
	r := &Room{
		id: rec.ID, game: g, tokens: rec.Tokens, started: rec.Started,
		conns: map[*Conn]int{}, inbox: make(chan func(), 64),
		now:   func() int64 { return time.Now().Unix() },
		store: st,
	}
	g.World = w
	now := r.now()
	for _, s := range g.Seats {
		if s.Phase != engine.PhaseDone {
			s.PhaseStart = now
		}
	}
	r.last = g.Public()
	r.touched.Store(rec.Updated)
	r.isOver.Store(g.Over)
	r.isStarted.Store(rec.Started)
	r.updateListing()
	go r.run()
	return r, nil
}

// enterHall trägt die beendete Partie in die Ruhmeshalle ein.
func (r *Room) enterHall() {
	if r.store == nil {
		return
	}
	e := store.HallEntry{ID: r.id, Ended: time.Now().Unix(), Year: r.game.Year}
	for i, s := range r.game.Seats {
		p := s.Player
		_, bot := r.game.Bots[i]
		hp := store.HallPlayer{Name: p.Name, Country: p.Country, Female: p.Female, Title: p.Title, Bot: bot, Winner: i == r.game.Winner}
		if hp.Winner {
			e.Winner = p.Name
		}
		e.Players = append(e.Players, hp)
	}
	if err := r.store.SaveHall(e); err != nil {
		slog.Error("ruhmeshalle", "id", r.id, "err", err)
	}
}

// persist schreibt den aktuellen Stand in die Datenbank (aus der Room-Goroutine).
func (r *Room) persist() {
	r.mark()
	if r.store == nil {
		return
	}
	snap, err := r.game.Snapshot()
	if err == nil {
		err = r.store.Save(store.Record{ID: r.id, Started: r.started, Tokens: r.tokens, Snapshot: snap})
	}
	if err != nil {
		slog.Error("spiel speichern", "id", r.id, "err", err)
	}
}

// run verarbeitet Aufträge nacheinander und prüft jede Sekunde die Zeitlimits.
func (r *Room) run() {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	// Computer-Herrscher handeln im sichtbaren Takt: ein Schritt je Sitz und Takt.
	botTick := time.NewTicker(BotStep)
	defer botTick.Stop()
	for {
		select {
		case f := <-r.inbox:
			f()
		case <-tick.C:
			if r.started && !r.paused() {
				if ev := r.game.Timeouts(r.now()); len(ev) > 0 {
					r.afterChange(ev, -1)
				}
			}
		case <-botTick.C:
			if !r.paused() {
				r.botStep()
			}
		}
	}
}

// paused: Ohne verbundenen Spieler ruht die Partie (keine Zugzeit, keine Computerzüge), damit
// nichts unbeobachtet weiterläuft. Asynchrone Partien ruhen nie: Ihre Zugzeiten laufen in Echtzeit. Beim Wiederverbinden werden die Phasenstarts um die
// Ruhezeit verschoben, damit niemand wegen der Pause Zugzeit verliert.
func (r *Room) paused() bool {
	if len(r.conns) > 0 || !r.started || r.game.Over || r.game.Settings.Async() {
		if r.pausedAt != 0 {
			d := r.now() - r.pausedAt
			for _, s := range r.game.Seats {
				s.PhaseStart += d
			}
			r.pausedAt = 0
		}
		return false
	}
	if r.pausedAt == 0 {
		r.pausedAt = r.now()
	}
	return true
}

// BotStep ist der Takt der Computer-Herrscher; langsam genug, um jeden Zug zu verfolgen.
var BotStep = 900 * time.Millisecond

// botStep lässt jeden Computer-Sitz eine Aktion ausführen; abgelehnte Aktionen beenden die Phase.
func (r *Room) botStep() {
	if !r.started || r.game.Over || len(r.game.Bots) == 0 {
		return
	}
	for seat := range r.game.Seats {
		strategy, isBot := r.game.Bots[seat]
		if !isBot {
			continue
		}
		a, ok := r.game.BotAction(seat, strategy)
		if !ok {
			continue
		}
		ev, err := r.game.Apply(seat, a, r.now())
		if err != nil {
			ev, err = r.game.Apply(seat, r.game.BotFallback(seat), r.now())
		}
		if err == nil {
			r.afterChange(ev, seat)
		}
	}
}

// addBot setzt einen Computer-Herrscher an den Tisch (nur Gastgeber, vor Spielbeginn).
func (r *Room) addBot(c *Conn, strategy string) {
	if c.seat != 0 || r.started {
		c.sendErr("not_host", "Nur der Gastgeber kann vor dem Start Computer-Herrscher hinzufügen.")
		return
	}
	if _, err := r.game.AddBot(strategy); err != nil {
		c.sendRuleErr(err)
		return
	}
	r.tokens = append(r.tokens, newToken()) // nie verschickt: Bot-Sitze sind nicht übernehmbar
	r.persist()
	r.broadcastLobby()
}

// kick entfernt einen Herrscher aus der Lobby (nur Gastgeber, vor Spielbeginn, nicht sich
// selbst). Menschliche Spieler werden zusätzlich per IP für dieses Spiel gesperrt, damit sie
// nicht sofort mit neuem Namen wiederkommen. Spätere Sitze rücken auf und erhalten ihre neue
// Sitznummer per welcome; ihr Token bleibt gleich.
func (r *Room) kick(c *Conn, seat int) {
	if c.seat != 0 || r.started {
		c.sendErr("not_host", "Nur der Gastgeber kann vor dem Start Herrscher entfernen.")
		return
	}
	if seat <= 0 || seat >= len(r.game.Seats) {
		c.sendErr("bad_seat", "Diesen Platz gibt es nicht.")
		return
	}
	human := !r.game.CPU[seat]
	r.game.RemoveSeat(seat)
	r.tokens = append(r.tokens[:seat], r.tokens[seat+1:]...)
	for conn, s := range r.conns {
		switch {
		case s == seat:
			if human {
				r.ban(conn.ip)
			}
			delete(r.conns, conn)
			conn.room.Store(nil)
			conn.seat = -1
			conn.sendErr("kicked", "Der Gastgeber hat Sie aus diesem Spiel entfernt.")
		case s > seat:
			r.conns[conn] = s - 1
			conn.seat = s - 1
			conn.send(welcomeMsg{"welcome", r.id, r.tokens[s-1], s - 1, false})
		}
	}
	r.persist()
	r.broadcastLobby()
}

// autopilot lässt den Computer für den eigenen Herrscher spielen (leere Spielweise = aus).
func (r *Room) autopilot(c *Conn, strategy string) {
	if strategy != "" {
		if _, ok := engine.BotStrategies[strategy]; !ok {
			c.sendErr("bad_strategy", "Unbekannte Spielweise.")
			return
		}
	}
	r.game.SetBot(c.seat, strategy)
	r.afterChange(nil, c.seat)
}

// do reiht einen Auftrag in die Room-Goroutine ein.
func (r *Room) do(f func()) { r.inbox <- f }

// join setzt eine Verbindung an einen neuen Sitz (nur vor Spielbeginn).
func (r *Room) join(c *Conn, name string, country int, female bool) {
	if r.started {
		c.sendErr("already_started", "Das Spiel läuft bereits.")
		return
	}
	if len(r.game.Seats) >= engine.MaxSeats {
		c.sendErr("room_full", "Der Reichstag ist voll.")
		return
	}
	if r.banned[c.ip] {
		c.sendErr("banned", "Der Gastgeber hat Sie aus diesem Spiel entfernt.")
		return
	}
	seat, err := r.game.AddPlayer(name, country, female)
	if err != nil {
		c.sendRuleErr(err)
		return
	}
	tok := newToken()
	r.tokens = append(r.tokens, tok)
	r.attach(c, seat, tok)
	r.persist()
	r.broadcastLobby()
}

// rejoin verbindet einen Sitz nach Verbindungsabbruch wieder (Token-Vergleich zeitkonstant).
func (r *Room) rejoin(c *Conn, token string) {
	for seat, t := range r.tokens {
		if subtle.ConstantTimeCompare([]byte(t), []byte(token)) == 1 {
			r.attach(c, seat, t)
			r.broadcastLobby()
			if r.started {
				r.sendGameState(c, seat)
			}
			return
		}
	}
	// Noch nicht zugelassen: zurück in den Wartezustand der Anfrage
	if r.resumeRequest(c, token) {
		return
	}
	c.sendErr("bad_token", "Dieser Platz ist nicht (mehr) gültig.")
}

// sendGameState schickt einer Verbindung den Vollstand eines laufenden Spiels.
func (r *Room) sendGameState(c *Conn, seat int) {
	c.send(r.publicMsg(true))
	c.send(selfMsg{"self", r.game.Self(seat)})
	// Bisherige Chronik nachliefern, damit nach dem Neuladen nichts fehlt
	c.send(eventsMsg{"history", r.game.Log})
}

// status liefert den Stand für „Meine Spiele“; Details nur bei passendem Token.
func (r *Room) status(token string) gameStatus {
	st := gameStatus{GameID: r.id, Found: true, Started: r.started, Over: r.game.Over}
	seat := -1
	for i, t := range r.tokens {
		if subtle.ConstantTimeCompare([]byte(t), []byte(token)) == 1 {
			seat = i
		}
	}
	if seat < 0 {
		st.Pending = r.pendingFor(token)
		return st
	}
	g := r.game
	s := g.Seats[seat]
	st.Valid, st.Year, st.Host = true, g.Year, seat == 0
	st.Name, st.Country, st.Female, st.Title = s.Player.Name, s.Player.Country, s.Player.Female, s.Player.Title
	st.Alive = s.Player.Alive()
	st.MyTurn = r.started && !g.Over && st.Alive && s.Phase != engine.PhaseDone
	if st.MyTurn {
		st.Deadline = g.Deadline(seat)
	}
	if g.Winner >= 0 {
		st.Winner = g.Seats[g.Winner].Player.Name
	}
	for _, x := range g.Seats {
		st.Players = append(st.Players, x.Player.Name)
	}
	return st
}

func (r *Room) attach(c *Conn, seat int, token string) {
	c.room.Store(r)
	c.seat = seat
	r.conns[c] = seat
	c.send(welcomeMsg{"welcome", r.id, token, seat, seat == 0})
	if seat == 0 && len(r.requests) > 0 {
		r.sendRequests()
	}
}

// leave trennt eine Verbindung; der Sitz bleibt erhalten.
func (r *Room) leave(c *Conn) {
	delete(r.conns, c)
	r.broadcastLobby()
}

// start beginnt das Spiel; nur der Gastgeber (Sitz 0).
func (r *Room) start(c *Conn) {
	if c.seat != 0 || r.started {
		c.sendErr("not_host", "Nur der Gastgeber kann das Spiel starten.")
		return
	}
	r.started = true
	ev := r.game.Start(r.now())
	r.broadcastLobby()
	r.broadcast(r.publicMsg(true))
	r.afterChange(ev, -1)
}

// act führt einen Spielzug aus und verteilt die Änderungen.
func (r *Room) act(c *Conn, a engine.Action) {
	if !r.started {
		c.sendErr("not_started", "Das Spiel hat noch nicht begonnen.")
		return
	}
	ev, err := r.game.Apply(c.seat, a, r.now())
	if err != nil {
		c.sendRuleErr(err)
		return
	}
	r.afterChange(ev, c.seat)
}

// afterChange schickt Ereignisse an alle, die eigene Sicht an Betroffene
// und die öffentliche Übersicht nur mit geänderten Einträgen.
func (r *Room) afterChange(ev []engine.Event, actor int) {
	affected := map[int]bool{}
	if actor >= 0 {
		affected[actor] = true
	}
	for _, e := range ev {
		if e.Player < 0 { // Jahreswechsel u. ä. betreffen alle
			for i := range r.game.Seats {
				affected[i] = true
			}
		} else {
			affected[e.Player] = true
		}
	}
	if len(ev) > 0 {
		r.game.Record(ev) // Chronik für Wiedereinstieg, wird mit dem Spielstand gespeichert
		for _, e := range ev {
			if e.Type == "gameOver" {
				r.enterHall()
			}
		}
		r.broadcast(eventsMsg{"events", ev})
	}
	for c, seat := range r.conns {
		if affected[seat] {
			c.send(selfMsg{"self", r.game.Self(seat)})
		}
	}
	r.persist()
	if m := r.publicMsg(false); len(m.Players) > 0 || m.Over {
		r.broadcast(m)
	}
}

// publicMsg baut die öffentliche Sicht; ohne full nur geänderte Spieler.
func (r *Room) publicMsg(full bool) publicMsg {
	cur := r.game.Public()
	m := publicMsg{T: "public", Year: r.game.Year, Over: r.game.Over, Winner: r.game.Winner, Full: full, Area: r.game.MapArea(), Pacts: r.game.Pacts}
	for i, p := range cur {
		if full || i >= len(r.last) || r.last[i] != p {
			m.Players = append(m.Players, p)
		}
	}
	r.last = cur
	return m
}

func (r *Room) broadcastLobby() { r.broadcast(r.lobby("lobby")) }

// lobby baut die Lobby-Übersicht; t = "lobby" für Mitspieler, "preview" vor dem Beitritt.
func (r *Room) lobby(t string) lobbyMsg {
	m := lobbyMsg{T: t, GameID: r.id, Started: r.started, Settings: r.game.Settings, Joinable: r.joinable()}
	online := map[int]bool{}
	for _, s := range r.conns {
		online[s] = true
	}
	for i, s := range r.game.Seats {
		m.Players = append(m.Players, lobbyPlayer{s.Player.Name, s.Player.Country, s.Player.Female, online[i], r.game.Bots[i]})
	}
	return m
}

func (r *Room) broadcast(v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		slog.Error("marshal", "err", err)
		return
	}
	for c := range r.conns {
		c.sendRaw(raw)
	}
}

// sendRuleErr zeigt Regelverstöße im Klartext, alles andere nur allgemein.
func (c *Conn) sendRuleErr(err error) {
	var re *engine.RuleError
	if errors.As(err, &re) {
		c.sendErr(re.Code, re.Message)
		return
	}
	slog.Warn("aktion fehlgeschlagen", "err", err)
	c.sendErr("internal", "Das hat nicht geklappt.")
}
