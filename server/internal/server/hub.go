package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"kaiserreloaded/server/internal/engine"
	"kaiserreloaded/server/internal/store"
)

// Hub verwaltet alle Räume und nimmt WebSocket-Verbindungen an.
type Hub struct {
	balance        *engine.Balance
	world          *engine.World
	store          *store.Store
	originPatterns []string
	mu             sync.Mutex
	rooms          map[string]*Room

	// Schutz vor Missbrauch (limits.go)
	limits      Limits
	conns       connCounter
	msgLimit    *RateLimiter // Nachrichten je IP: 20/s, Vorrat 60
	gameLimit   *RateLimiter // neue Spiele je IP: eines je 5 Minuten, Vorrat 3
	uploadLimit *RateLimiter // Uploads je IP: einer je 10 s, Vorrat 5
}

// SetLimits setzt die Obergrenzen (vor dem Start des HTTP-Servers aufrufen).
func (h *Hub) SetLimits(l Limits) { h.limits = l }

// ActiveGames zählt die laufenden Spiele (siehe Room.counts).
func (h *Hub) ActiveGames() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	n := 0
	for _, r := range h.rooms {
		if r.counts(now) {
			n++
		}
	}
	return n
}

// errFull: Meldung, wenn alle Plätze für Spiele belegt sind.
func (h *Hub) errFull() string {
	return fmt.Sprintf("Im Moment sind keine weiteren Spiele möglich, weil gerade %d Spiele laufen. "+
		"Bitte später noch einmal versuchen oder einem laufenden Spiel beitreten.", h.limits.MaxGames)
}

// NewHub erzeugt den Hub und lädt gespeicherte Spiele. originPatterns erlaubt fremde
// Origins (z. B. Vite-Dev-Server); st darf nil sein (dann ohne Speicherung).
func NewHub(b *engine.Balance, w *engine.World, originPatterns []string, st *store.Store) (*Hub, error) {
	h := &Hub{balance: b, world: w, originPatterns: originPatterns, store: st, rooms: map[string]*Room{},
		limits: DefaultLimits(), msgLimit: NewRateLimiter(20, 60), gameLimit: NewRateLimiter(1.0/300, 3),
		uploadLimit: NewRateLimiter(0.1, 5)}
	if st == nil {
		return h, nil
	}
	recs, err := st.LoadAll()
	if err != nil {
		return nil, err
	}
	for _, rec := range recs {
		r, err := restoreRoom(rec, st, w)
		if err != nil {
			slog.Warn("spiel nicht wiederhergestellt", "id", rec.ID, "err", err)
			continue
		}
		h.rooms[rec.ID] = r
	}
	slog.Info("spiele geladen", "anzahl", len(h.rooms))
	return h, nil
}

// Conn ist eine Browser-Verbindung mit eigenem Sendepuffer.
type Conn struct {
	ip      string // Absender-IP für die Grenzen je IP
	ws      *websocket.Conn
	out     chan []byte
	room    atomic.Pointer[Room] // wird in der Room-Goroutine gesetzt, im Lese-Loop gelesen
	pending atomic.Pointer[Room] // Spiel mit offener Beitrittsanfrage dieser Verbindung
	seat    int                  // nur in der Room-Goroutine benutzen
}

func (c *Conn) send(v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		slog.Error("marshal", "err", err)
		return
	}
	c.sendRaw(raw)
}

// sendRaw verwirft Nachrichten, wenn ein Client nicht hinterherkommt; er holt sich beim Rejoin den Vollstand.
func (c *Conn) sendRaw(raw []byte) {
	select {
	case c.out <- raw:
	default:
		slog.Warn("sendepuffer voll, verbindung wird geschlossen")
		c.ws.CloseNow()
	}
}

func (c *Conn) sendErr(code, msg string) { c.send(errorMsg{"error", code, msg}) }

// ServeHTTP nimmt eine WebSocket-Verbindung an und liest Nachrichten bis zum Abbruch.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Verbindungen je IP und insgesamt begrenzen, bevor der WebSocket überhaupt angenommen wird
	ip := clientIP(r, h.limits.TrustProxy)
	if !h.conns.acquire(ip, h.limits.MaxConnsPerIP, h.limits.MaxConns) {
		http.Error(w, "Zu viele Verbindungen.", http.StatusServiceUnavailable)
		return
	}
	defer h.conns.release(ip)
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: h.originPatterns})
	if err != nil {
		return
	}
	ws.SetReadLimit(4096) // Spielzüge sind klein; schützt vor übergroßen Nachrichten
	c := &Conn{ip: ip, ws: ws, out: make(chan []byte, 64), seat: -1}
	flood := 0 // abgelehnte Nachrichten in Folge; wer weiterflutet, wird getrennt
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go c.writeLoop(ctx)

	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			break
		}
		if !h.msgLimit.Allow(ip) {
			if flood++; flood == 1 {
				c.sendErr("too_fast", "Zu viele Aktionen auf einmal – bitte kurz warten.")
			}
			if flood > 200 {
				slog.Warn("nachrichtenflut, verbindung getrennt", "ip", ip)
				break
			}
			continue
		}
		flood = 0
		var m ClientMsg
		if json.Unmarshal(data, &m) != nil {
			c.sendErr("bad_message", "Unverständliche Nachricht.")
			continue
		}
		h.handle(c, m)
	}
	if room := c.room.Load(); room != nil {
		room.do(func() { room.leave(c) })
	}
	if room := c.pending.Load(); room != nil {
		room.do(func() { room.dropConn(c) })
	}
	ws.CloseNow()
}

func (c *Conn) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case raw := <-c.out:
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.ws.Write(wctx, websocket.MessageText, raw)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

// handle prüft Eingaben und leitet sie an den passenden Raum weiter.
func (h *Hub) handle(c *Conn, m ClientMsg) {
	switch m.T {
	case "create", "join":
		if c.room.Load() != nil {
			c.sendErr("already_joined", "Sie sitzen bereits in einem Spiel.")
			return
		}
		if c.pending.Load() != nil {
			c.sendErr("already_requested", "Sie warten noch auf die Zulassung zu einem Spiel.")
			return
		}
		name, problem := cleanName(m.Name)
		if problem != engine.TextOK {
			c.sendErr("bad_name", engine.TextMessage(problem, "Der Name", maxName))
			return
		}
		var room *Room
		if m.T == "create" {
			// Höchstens MaxGames laufende Spiele; Spiele-Anlage je IP gebremst
			if h.ActiveGames() >= h.limits.MaxGames {
				c.sendErr("server_full", h.errFull())
				return
			}
			if !h.gameLimit.Allow(c.ip) {
				c.sendErr("too_many_games", "Sie haben gerade schon mehrere Spiele angelegt. Bitte etwas warten.")
				return
			}
			s := engine.DefaultSettings(h.balance)
			if m.Settings != nil {
				s = *m.Settings
			}
			if err := s.Validate(); err != nil {
				c.sendRuleErr(err)
				return
			}
			room = h.newRoom(s)
			room.do(func() { room.join(c, name, m.Country, m.Female) })
			return
		}
		// Beitritt nur auf Anfrage; der Gastgeber entscheidet (beitritt.go)
		room = h.room(m.GameID)
		if room == nil {
			c.sendErr("no_game", "Spiel nicht gefunden.")
			return
		}
		room.do(func() { room.request(c, name, m.Country, m.Female) })
	case "rejoin":
		room := h.room(m.GameID)
		if room == nil || c.room.Load() != nil {
			c.sendErr("no_game", "Spiel nicht gefunden.")
			return
		}
		room.do(func() { room.rejoin(c, m.Token) })
	case "peek":
		// Vorschau vor dem Beitritt: vergebene Länder und Einstellungen, ohne Platz zu belegen.
		room := h.room(m.GameID)
		if room == nil {
			c.sendErr("no_game", "Spiel nicht gefunden.")
			return
		}
		room.do(func() { c.send(room.lobby("preview")) })
	case "games":
		// Status der im Browser gemerkten Spiele, ohne dort Platz zu nehmen. Läuft nebenher,
		// damit die Lesegoroutine nicht auf mehrere Räume wartet.
		refs := m.Games
		if len(refs) > maxGameRefs {
			refs = refs[:maxGameRefs]
		}
		go func() { c.send(gamesMsg{"games", h.gamesStatus(refs)}) }()
	case "start":
		if room := c.room.Load(); room != nil {
			room.do(func() { room.start(c) })
		}
	case "addBot":
		if room := c.room.Load(); room != nil {
			room.do(func() { room.addBot(c, m.Strategy) })
		}
	case "decide":
		if room := c.room.Load(); room != nil {
			id, ok := m.Request, m.Accept
			room.do(func() { room.decide(c, id, ok) })
		}
	case "cancelJoin":
		if room := c.pending.Load(); room != nil {
			room.do(func() { room.cancelRequest(c) })
		}
	case "kick":
		if room := c.room.Load(); room != nil {
			seat := m.Seat
			room.do(func() { room.kick(c, seat) })
		}
	case "autopilot":
		if room := c.room.Load(); room != nil {
			room.do(func() { room.autopilot(c, m.Strategy) })
		}
	case "act":
		if room := c.room.Load(); room != nil && m.Action != nil {
			a := *m.Action
			room.do(func() { room.act(c, a) })
		}
	default:
		c.sendErr("bad_message", "Unbekannte Nachricht.")
	}
}

func (h *Hub) newRoom(s engine.Settings) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := newGameID()
	for h.rooms[id] != nil {
		id = newGameID()
	}
	r := newRoom(id, s, h.balance, h.world, h.store)
	h.rooms[id] = r
	return r
}

func (h *Hub) room(id string) *Room {
	if !validGameID(id) {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.rooms[id]
}

// gamesStatus fragt jeden Raum in seiner eigenen Goroutine ab (höchstens 2 s je Raum).
func (h *Hub) gamesStatus(refs []gameRef) []gameStatus {
	out := make([]gameStatus, 0, len(refs))
	for _, ref := range refs {
		room := h.room(ref.GameID)
		if room == nil {
			out = append(out, gameStatus{GameID: ref.GameID})
			continue
		}
		ch := make(chan gameStatus, 1)
		token := ref.Token
		room.do(func() { ch <- room.status(token) })
		select {
		case st := <-ch:
			out = append(out, st)
		case <-time.After(2 * time.Second):
			out = append(out, gameStatus{GameID: ref.GameID, Found: true})
		}
	}
	return out
}

// StatusHandler liefert Auslastung und Grenzen für die Lobby (GET /api/status).
func (h *Hub) StatusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{
			"games": h.ActiveGames(), "maxGames": h.limits.MaxGames, "maxPlayers": engine.MaxSeats,
		})
	})
}
