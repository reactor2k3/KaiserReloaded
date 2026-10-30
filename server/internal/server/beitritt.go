package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"kaiserreloaded/server/internal/engine"
)

// Öffentliche Spieleliste und Beitritt auf Anfrage: Wer einem Spiel beitreten will, stellt eine
// Anfrage; der Gastgeber (Sitz 0) lässt sie zu oder lehnt ab. Beitreten geht vor dem Start und
// im laufenden Spiel, bis der erste Herrscher befördert wurde (engine.Game.Promoted).

// Grenzen gegen Anfrage-Spam: offene Anfragen je Spiel, Ablehnungen bis zur Sperre der IP.
const (
	maxRequests   = 8
	maxPerIP      = 3 // mehrere je IP, weil Mitspieler am selben Anschluss sitzen können
	maxDenials    = 3
	requestIDSize = 8
)

// joinRequest ist eine offene Beitrittsanfrage. Token wird beim Zulassen zum Sitz-Token, damit
// der Anfragende nach einem Neuladen ohne neue Anfrage hereinkommt. Anfragen liegen nur im
// Speicher; nach einem Server-Neustart muss neu angefragt werden.
type joinRequest struct {
	ID      string
	Token   string
	Name    string
	Country int
	Female  bool
	IP      string
	conn    *Conn // nil, solange der Anfragende nicht verbunden ist
}

// requestView ist die Sicht des Gastgebers auf eine Anfrage (ohne Token und IP).
type requestView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Country int    `json:"country"`
	Female  bool   `json:"female"`
}

type requestsMsg struct {
	T        string        `json:"t"`
	GameID   string        `json:"gameId"`
	Requests []requestView `json:"requests"`
}

// pendingMsg bestätigt dem Anfragenden die Anfrage; Token wird im Browser gemerkt.
type pendingMsg struct {
	T      string `json:"t"`
	GameID string `json:"gameId"`
	Token  string `json:"token"`
	Host   string `json:"host"`
}

// listing ist ein Eintrag der öffentlichen Spieleliste.
type listing struct {
	GameID       string `json:"gameId"`
	Host         string `json:"host"`
	HostCountry  int    `json:"hostCountry"`
	Players      int    `json:"players"`
	MaxPlayers   int    `json:"maxPlayers"`
	Started      bool   `json:"started"`
	Year         int    `json:"year"`
	Mode         string `json:"mode"`
	PhaseSeconds int    `json:"phaseSeconds"`
	Classic      bool   `json:"classic"`
	Joinable     bool   `json:"joinable"`
	updated      int64
}

// joinable: Platz frei, Spiel nicht vorbei und noch niemand befördert.
func (r *Room) joinable() bool {
	return !r.game.Over && len(r.game.Seats) < engine.MaxSeats && !(r.started && r.game.Promoted())
}

// updateListing legt den Listeneintrag für den Hub ab (aus der Room-Goroutine).
func (r *Room) updateListing() {
	g := r.game
	l := &listing{GameID: r.id, Players: len(g.Seats), MaxPlayers: engine.MaxSeats, Started: r.started,
		Year: g.Year, Mode: g.Settings.Mode, PhaseSeconds: g.Settings.PhaseSeconds, Classic: g.Settings.Classic,
		Joinable: r.joinable(), updated: r.touched.Load()}
	if l.Mode == "" {
		l.Mode = "live"
	}
	if len(g.Seats) > 0 {
		l.Host, l.HostCountry = g.Seats[0].Player.Name, g.Seats[0].Player.Country
	}
	r.list.Store(l)
}

// request nimmt eine Beitrittsanfrage an und meldet sie dem Gastgeber.
func (r *Room) request(c *Conn, name string, country int, female bool) {
	switch {
	case country < 1 || country > 9:
		c.sendErr("invalid_country", "Ungültiges Land.")
		return
	case r.banned[c.ip]:
		c.sendErr("banned", "Der Gastgeber hat Sie aus diesem Spiel entfernt.")
		return
	case !r.joinable():
		c.sendErr("not_joinable", "Diesem Spiel kann man nicht mehr beitreten.")
		return
	case r.game.CountryCount(country) >= r.game.PerCountry():
		c.sendErr("country_taken", "Dieses Land ist schon vergeben.")
		return
	case len(r.requests) >= maxRequests:
		c.sendErr("too_many_requests", "Für dieses Spiel warten schon viele Anfragen. Bitte später erneut versuchen.")
		return
	}
	sameIP := 0
	for _, q := range r.requests {
		if q.IP == c.ip {
			sameIP++
		}
		if q.conn == c || sameIP >= maxPerIP {
			c.sendErr("already_requested", "Sie haben für dieses Spiel schon angefragt. Bitte auf den Gastgeber warten.")
			return
		}
	}
	q := &joinRequest{ID: newToken()[:requestIDSize], Token: newToken(), Name: name, Country: country, Female: female, IP: c.ip, conn: c}
	r.requests = append(r.requests, q)
	c.pending.Store(r)
	c.send(pendingMsg{"pending", r.id, q.Token, r.game.Seats[0].Player.Name})
	r.sendRequests()
}

// sendRequests schickt die offenen Anfragen an alle Verbindungen des Gastgebers.
func (r *Room) sendRequests() {
	m := requestsMsg{T: "requests", GameID: r.id, Requests: []requestView{}}
	for _, q := range r.requests {
		m.Requests = append(m.Requests, requestView{q.ID, q.Name, q.Country, q.Female})
	}
	for c, seat := range r.conns {
		if seat == 0 {
			c.send(m)
		}
	}
}

// decide: Gastgeber lässt eine Anfrage zu oder lehnt sie ab. Beim Zulassen wird alles noch
// einmal geprüft, weil sich das Spiel seit der Anfrage geändert haben kann.
func (r *Room) decide(c *Conn, id string, accept bool) {
	if c.seat != 0 {
		c.sendErr("not_host", "Nur der Gastgeber entscheidet über Beitritte.")
		return
	}
	idx := -1
	for i, q := range r.requests {
		if q.ID == id {
			idx = i
		}
	}
	if idx < 0 {
		c.sendErr("no_request", "Diese Anfrage gibt es nicht mehr.")
		return
	}
	q := r.requests[idx]
	r.requests = append(r.requests[:idx], r.requests[idx+1:]...)
	defer r.sendRequests()
	if q.conn != nil {
		q.conn.pending.Store(nil)
	}
	if !accept {
		if r.denied == nil {
			r.denied = map[string]int{}
		}
		if r.denied[q.IP]++; r.denied[q.IP] >= maxDenials {
			r.ban(q.IP)
		}
		if q.conn != nil {
			q.conn.sendErr("denied", "Der Gastgeber hat Ihre Beitrittsanfrage abgelehnt.")
		}
		return
	}
	if !r.joinable() {
		c.sendErr("not_joinable", "Ein Beitritt ist nicht mehr möglich (Spiel voll oder schon jemand befördert).")
		if q.conn != nil {
			q.conn.sendErr("not_joinable", "Diesem Spiel kann man nicht mehr beitreten.")
		}
		return
	}
	var seat int
	var err error
	if r.started {
		seat, err = r.game.JoinRunning(q.Name, q.Country, q.Female)
	} else {
		seat, err = r.game.AddPlayer(q.Name, q.Country, q.Female)
	}
	if err != nil {
		c.sendRuleErr(err)
		if q.conn != nil {
			q.conn.sendRuleErr(err)
		}
		return
	}
	r.tokens = append(r.tokens, q.Token)
	if q.conn != nil {
		r.attach(q.conn, seat, q.Token)
		if r.started {
			r.sendGameState(q.conn, seat)
		}
	}
	r.persist()
	r.broadcastLobby()
	if r.started {
		r.broadcast(r.publicMsg(true))
	}
}

// ban sperrt eine IP für dieses Spiel (Hinauswurf oder wiederholt abgelehnt).
func (r *Room) ban(ip string) {
	if r.banned == nil {
		r.banned = map[string]bool{}
	}
	r.banned[ip] = true
}

// resumeRequest verbindet einen Anfragenden nach Neuladen wieder mit seiner Anfrage.
func (r *Room) resumeRequest(c *Conn, token string) bool {
	for _, q := range r.requests {
		if subtle.ConstantTimeCompare([]byte(q.Token), []byte(token)) == 1 {
			q.conn = c
			c.pending.Store(r)
			c.send(pendingMsg{"pending", r.id, q.Token, r.game.Seats[0].Player.Name})
			return true
		}
	}
	return false
}

// dropConn löst eine getrennte Verbindung von ihrer Anfrage; die Anfrage bleibt bestehen.
func (r *Room) dropConn(c *Conn) {
	for _, q := range r.requests {
		if q.conn == c {
			q.conn = nil
		}
	}
}

// cancelRequest zieht die eigene Anfrage zurück.
func (r *Room) cancelRequest(c *Conn) {
	for i, q := range r.requests {
		if q.conn == c {
			r.requests = append(r.requests[:i], r.requests[i+1:]...)
			c.pending.Store(nil)
			r.sendRequests()
			return
		}
	}
}

// pendingFor meldet, ob ein Token zu einer offenen Anfrage gehört (für „Meine Spiele“).
func (r *Room) pendingFor(token string) bool {
	for _, q := range r.requests {
		if subtle.ConstantTimeCompare([]byte(q.Token), []byte(token)) == 1 {
			return true
		}
	}
	return false
}

// ListHandler liefert die öffentliche Spieleliste (GET /api/games): laufende und offene Spiele,
// beitretbare zuerst, sonst die zuletzt aktiven. Keine Tokens oder IPs.
func (h *Hub) ListHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		now := time.Now()
		h.mu.Lock()
		out := []*listing{}
		for _, r := range h.rooms {
			if l := r.list.Load(); l != nil && r.counts(now) {
				out = append(out, l)
			}
		}
		h.mu.Unlock()
		sort.Slice(out, func(i, j int) bool {
			if out[i].Joinable != out[j].Joinable {
				return out[i].Joinable
			}
			return out[i].updated > out[j].updated
		})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]any{"games": out})
	})
}
