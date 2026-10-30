package server

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"kaiserreloaded/server/internal/engine"
	"kaiserreloaded/server/internal/media"
	"kaiserreloaded/server/internal/store"
)

// testClient ist eine WebSocket-Verbindung mit Hilfsfunktionen zum Warten auf Nachrichten.
type testClient struct {
	t  *testing.T
	ws *websocket.Conn
}

func (c *testClient) send(v any) {
	c.t.Helper()
	raw, _ := json.Marshal(v)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageText, raw); err != nil {
		c.t.Fatal(err)
	}
}

// waitFor liest Nachrichten, bis eine vom Typ t kommt (andere werden übersprungen).
func (c *testClient) waitFor(t string) map[string]any {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		_, raw, err := c.ws.Read(ctx)
		if err != nil {
			c.t.Fatalf("warte auf %q: %v", t, err)
		}
		var m map[string]any
		json.Unmarshal(raw, &m)
		if m["t"] == t {
			return m
		}
	}
}

func setup(t *testing.T) (*httptest.Server, *Hub) {
	t.Helper()
	b := engine.DefaultBalance()
	w, err := engine.LoadWorld(filepath.Join("..", "..", "..", "data", "world.json"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	hub, err := NewHub(b, w, nil, st)
	if err != nil {
		t.Fatal(err)
	}
	ms, err := media.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ws", hub)
	mux.Handle("/upload", hub.UploadHandler(ms))
	mux.Handle("/media/", ms)
	mux.Handle("/api/games", hub.ListHandler())
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, hub
}

func dial(t *testing.T, srv *httptest.Server) *testClient {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ws.CloseNow() })
	return &testClient{t, ws}
}

func TestLobbyJoinStartAndPlay(t *testing.T) {
	srv, _ := setup(t)
	host := dial(t, srv)
	host.send(map[string]any{"t": "create", "name": "Micha", "country": 5})
	w := host.waitFor("welcome")
	gameID, token := w["gameId"].(string), w["token"].(string)
	if !w["host"].(bool) {
		t.Fatal("Ersteller ist nicht Gastgeber")
	}

	guest := dial(t, srv)
	guest.send(map[string]any{"t": "peek", "gameId": gameID})
	if p := guest.waitFor("preview"); len(p["players"].([]any)) != 1 {
		t.Fatalf("Vorschau falsch: %v", p)
	}
	guest.send(map[string]any{"t": "join", "gameId": gameID, "name": "Anna", "country": 10})
	if e := guest.waitFor("error"); e["code"] != "invalid_country" {
		t.Fatalf("ungültiges Land erlaubt: %v", e)
	}
	guest.send(map[string]any{"t": "join", "gameId": gameID, "name": "  ", "country": 3})
	if e := guest.waitFor("error"); e["code"] != "bad_name" {
		t.Fatalf("leerer Name erlaubt: %v", e)
	}
	admit(t, host, guest, gameID, "Anna", 3)

	guest.send(map[string]any{"t": "start"})
	if e := guest.waitFor("error"); e["code"] != "not_host" {
		t.Fatalf("Gast darf starten: %v", e)
	}
	host.send(map[string]any{"t": "start"})
	self := host.waitFor("self")["self"].(map[string]any)
	if self["phase"] != "market" {
		t.Fatalf("nach Start nicht in der Marktphase: %v", self["phase"])
	}
	// Verdeckte Steuerfaktoren dürfen den Server nicht verlassen
	if tf := self["player"].(map[string]any)["year"].(map[string]any)["taxFactors"].(map[string]any); tf["Tolerance"].(float64) != 0 {
		t.Error("Steuerfaktoren werden an den Client geschickt")
	}

	host.send(map[string]any{"t": "act", "action": map[string]any{"type": "endTurn"}})
	if e := host.waitFor("error"); e["code"] != "wrong_phase" {
		t.Fatalf("Zug beenden in Marktphase erlaubt: %v", e)
	}
	host.send(map[string]any{"t": "act", "action": map[string]any{"type": "next"}})
	if s := host.waitFor("self")["self"].(map[string]any); s["phase"] != "feed" {
		t.Fatalf("Phase nicht weitergeschaltet: %v", s["phase"])
	}

	// Wiedereinstieg mit Token liefert Vollstand und Chronik
	again := dial(t, srv)
	again.send(map[string]any{"t": "rejoin", "gameId": gameID, "token": token})
	again.waitFor("welcome")
	if p := again.waitFor("public"); p["full"] != true {
		t.Error("Wiedereinstieg ohne vollständige Übersicht")
	}
	if h := again.waitFor("history"); len(h["events"].([]any)) == 0 {
		t.Error("Chronik beim Wiedereinstieg leer")
	}
	bad := dial(t, srv)
	bad.send(map[string]any{"t": "rejoin", "gameId": gameID, "token": strings.Repeat("0", 32)})
	if e := bad.waitFor("error"); e["code"] != "bad_token" {
		t.Errorf("falsches Token angenommen: %v", e)
	}
}

func TestUploadAuthAndTypes(t *testing.T) {
	srv, _ := setup(t)
	c := dial(t, srv)
	c.send(map[string]any{"t": "create", "name": "Micha", "country": 5})
	w := c.waitFor("welcome")

	post := func(token, kind, name string, data []byte) (int, map[string]string) {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		mw.WriteField("gameId", w["gameId"].(string))
		mw.WriteField("token", token)
		mw.WriteField("kind", kind)
		fw, _ := mw.CreateFormFile("file", name)
		fw.Write(data)
		mw.Close()
		res, err := http.Post(srv.URL+"/upload", mw.FormDataContentType(), &body)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]string
		json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}

	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, image.NewRGBA(image.Rect(0, 0, 64, 40)))
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><rect width="1" height="1"/></svg>`)

	if code, _ := post(strings.Repeat("a", 32), "avatar", "a.png", pngBuf.Bytes()); code != http.StatusForbidden {
		t.Errorf("Upload ohne gültiges Token: %d", code)
	}
	if code, _ := post(w["token"].(string), "avatar", "a.svg", svg); code != http.StatusUnsupportedMediaType {
		t.Errorf("SVG als Spielerbild: %d", code)
	}
	code, out := post(w["token"].(string), "arms", "w.svg", svg)
	if code != http.StatusOK {
		t.Fatalf("SVG-Wappen abgelehnt: %d %v", code, out)
	}
	res, err := http.Get(srv.URL + "/media/" + out["media"])
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var served bytes.Buffer
	served.ReadFrom(res.Body)
	if strings.Contains(served.String(), "script") || res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("Wappen nicht gesäubert oder ohne Schutz-Header: %s", served.String())
	}
	if res2, _ := http.Get(srv.URL + "/media/../kaiser.db"); res2 != nil && res2.StatusCode == http.StatusOK {
		t.Error("Pfad außerhalb des Medienordners erreichbar")
	}
}

// „Meine Spiele“: Status nur mit passendem Token, unbekannte Spiele als nicht gefunden.
func TestGamesStatus(t *testing.T) {
	srv, _ := setup(t)
	host := dial(t, srv)
	host.send(map[string]any{"t": "create", "name": "Micha", "country": 5})
	w := host.waitFor("welcome")
	gameID, token := w["gameId"].(string), w["token"].(string)
	host.send(map[string]any{"t": "addBot", "strategy": "kaufmann"})
	host.send(map[string]any{"t": "start"})
	host.waitFor("self")

	other := dial(t, srv)
	other.send(map[string]any{"t": "games", "games": []map[string]string{
		{"gameId": gameID, "token": token},
		{"gameId": gameID, "token": strings.Repeat("0", 32)},
		{"gameId": "ZZZZZZ", "token": token},
	}})
	gs := other.waitFor("games")["games"].([]any)
	if len(gs) != 3 {
		t.Fatalf("drei Einträge erwartet: %v", gs)
	}
	mine := gs[0].(map[string]any)
	if mine["valid"] != true || mine["started"] != true || mine["name"] != "Micha" || mine["myTurn"] != true || mine["host"] != true ||
		len(mine["players"].([]any)) != 2 {
		t.Errorf("eigenes Spiel falsch: %v", mine)
	}
	if wrong := gs[1].(map[string]any); wrong["found"] != true || wrong["valid"] != false || wrong["name"] != nil {
		t.Errorf("falsches Token verrät Daten: %v", wrong)
	}
	if gone := gs[2].(map[string]any); gone["found"] != false {
		t.Errorf("unbekanntes Spiel gefunden: %v", gone)
	}
}

// TestGameLimit: Ist die Höchstzahl laufender Spiele erreicht, wird kein weiteres angelegt.
func TestGameLimit(t *testing.T) {
	srv, hub := setup(t)
	hub.SetLimits(Limits{MaxGames: 1, MaxConnsPerIP: 8, MaxConns: 100})
	a := dial(t, srv)
	a.send(map[string]any{"t": "create", "name": "Erster", "country": 5})
	a.waitFor("welcome")
	b := dial(t, srv)
	b.send(map[string]any{"t": "create", "name": "Zweiter", "country": 4})
	e := b.waitFor("error")
	if e["code"] != "server_full" || !strings.Contains(e["message"].(string), "1 Spiele laufen") {
		t.Errorf("erwartet server_full, bekommen %v", e)
	}
	if hub.ActiveGames() != 1 {
		t.Errorf("laufende Spiele: %d", hub.ActiveGames())
	}
}

// TestKick: Gastgeber entfernt einen Spieler; spätere Sitze rücken auf, der Entfernte ist gesperrt.
func TestKick(t *testing.T) {
	srv, _ := setup(t)
	host := dial(t, srv)
	host.send(map[string]any{"t": "create", "name": "Wirt", "country": 5})
	gameID := host.waitFor("welcome")["gameId"].(string)
	guest := dial(t, srv)
	admit(t, host, guest, gameID, "Stoerer", 1)
	third := dial(t, srv)
	tok := admit(t, host, third, gameID, "Dritter", 2)

	guest.send(map[string]any{"t": "kick", "seat": 0})
	if e := guest.waitFor("error"); e["code"] != "not_host" {
		t.Errorf("Gast darf nicht entfernen: %v", e)
	}
	host.send(map[string]any{"t": "kick", "seat": 1})
	if e := guest.waitFor("error"); e["code"] != "kicked" {
		t.Errorf("Entfernter nicht benachrichtigt: %v", e)
	}
	if w := third.waitFor("welcome"); w["seat"] != 1.0 || w["token"] != tok {
		t.Errorf("Dritter nicht aufgerückt: %v", w)
	}
	// Ältere Lobby-Stände überspringen, bis „Dritter“ auf Platz 2 steht
	for {
		ps := host.waitFor("lobby")["players"].([]any)
		if len(ps) == 2 && ps[1].(map[string]any)["name"] == "Dritter" {
			break
		}
	}
	again := dial(t, srv)
	again.send(map[string]any{"t": "join", "gameId": gameID, "name": "Neuer Name", "country": 3})
	// Gesperrt: schon die Anfrage wird abgelehnt
	if e := again.waitFor("error"); e["code"] != "banned" {
		t.Errorf("Entfernter kommt wieder herein: %v", e)
	}
}

// admit: Gast fragt an, Gastgeber lässt zu; liefert das Token des neuen Sitzes.
func admit(t *testing.T, host, guest *testClient, gameID, name string, country int) any {
	t.Helper()
	guest.send(map[string]any{"t": "join", "gameId": gameID, "name": name, "country": country})
	tok := guest.waitFor("pending")["token"]
	for {
		for _, q := range host.waitFor("requests")["requests"].([]any) {
			if q := q.(map[string]any); q["name"] == name {
				host.send(map[string]any{"t": "decide", "request": q["id"], "accept": true})
				if w := guest.waitFor("welcome"); w["token"] != tok {
					t.Fatalf("Sitz-Token weicht vom Anfrage-Token ab: %v", w)
				}
				return tok
			}
		}
	}
}

// TestJoinRequest: Ablehnen, Zurückkehren nach Neuladen, Beitritt ins laufende Spiel und
// Ende der Beitrittsmöglichkeit nach der ersten Beförderung.
func TestJoinRequest(t *testing.T) {
	srv, hub := setup(t)
	host := dial(t, srv)
	host.send(map[string]any{"t": "create", "name": "Wirt", "country": 5})
	gameID := host.waitFor("welcome")["gameId"].(string)

	// Ablehnen
	g1 := dial(t, srv)
	g1.send(map[string]any{"t": "join", "gameId": gameID, "name": "Abgelehnt", "country": 1})
	g1.waitFor("pending")
	id := host.waitFor("requests")["requests"].([]any)[0].(map[string]any)["id"]
	host.send(map[string]any{"t": "decide", "request": id, "accept": false})
	if e := g1.waitFor("error"); e["code"] != "denied" {
		t.Fatalf("Ablehnung nicht gemeldet: %v", e)
	}

	// Start, dann Anfrage im laufenden Spiel; der Anfragende lädt neu, bevor zugelassen wird
	host.send(map[string]any{"t": "addBot", "strategy": "kaufmann"})
	host.send(map[string]any{"t": "start"})
	host.waitFor("self")
	g2 := dial(t, srv)
	g2.send(map[string]any{"t": "join", "gameId": gameID, "name": "Spaet", "country": 2})
	tok := g2.waitFor("pending")["token"]
	g2.ws.CloseNow()
	g3 := dial(t, srv)
	g3.send(map[string]any{"t": "rejoin", "gameId": gameID, "token": tok})
	g3.waitFor("pending")
	var reqID any
	for reqID == nil {
		for _, q := range host.waitFor("requests")["requests"].([]any) {
			if q := q.(map[string]any); q["name"] == "Spaet" {
				reqID = q["id"]
			}
		}
	}
	host.send(map[string]any{"t": "decide", "request": reqID, "accept": true})
	if w := g3.waitFor("welcome"); w["seat"] != 2.0 {
		t.Fatalf("falscher Sitz: %v", w)
	}
	if self := g3.waitFor("self")["self"].(map[string]any); self["phase"] != "done" {
		t.Errorf("Neuer muss bis zum nächsten Jahr warten: %v", self["phase"])
	}

	// Liste: Spiel steht drin und ist beitretbar; nach einer Beförderung nicht mehr
	room := hub.room(gameID)
	listed := func() map[string]any {
		res, err := http.Get(srv.URL + "/api/games")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string][]map[string]any
		json.NewDecoder(res.Body).Decode(&out)
		return out["games"][0]
	}
	if l := listed(); l["gameId"] != gameID || l["joinable"] != true || l["players"] != 3.0 {
		t.Errorf("Listeneintrag falsch: %v", l)
	}
	done := make(chan struct{})
	room.do(func() { room.game.Seats[1].Player.Title++; room.persist(); close(done) })
	<-done
	if l := listed(); l["joinable"] != false {
		t.Errorf("nach Beförderung noch beitretbar: %v", l)
	}
	g4 := dial(t, srv)
	g4.send(map[string]any{"t": "join", "gameId": gameID, "name": "Zu spaet", "country": 3})
	if e := g4.waitFor("error"); e["code"] != "not_joinable" {
		t.Errorf("Beitritt nach Beförderung möglich: %v", e)
	}
}
