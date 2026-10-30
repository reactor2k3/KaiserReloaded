// Package server verbindet Browser per WebSocket mit laufenden Spielen.
// Jedes Spiel (Room) läuft in genau einer Goroutine; Verbindungen schicken
// nur Absichten, der Server prüft und rechnet über die Engine.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"kaiserreloaded/server/internal/engine"
)

// ClientMsg ist eine Nachricht vom Browser. Feld t bestimmt die Art.
type ClientMsg struct {
	T        string           `json:"t"`                  // create | join | rejoin | peek | games | start | act | addBot | autopilot | kick | decide | cancelJoin
	Strategy string           `json:"strategy,omitempty"` // Spielweise für addBot/autopilot
	GameID   string           `json:"gameId,omitempty"`
	Token    string           `json:"token,omitempty"`
	Name     string           `json:"name,omitempty"`
	Country  int              `json:"country,omitempty"`
	Female   bool             `json:"female,omitempty"`
	Settings *engine.Settings `json:"settings,omitempty"`
	Action   *engine.Action   `json:"action,omitempty"`
	Games    []gameRef        `json:"games,omitempty"`   // games: eigene Plätze (aus dem Browser gemerkt)
	Seat     int              `json:"seat,omitempty"`    // kick: Sitz, der die Lobby verlassen muss
	Request  string           `json:"request,omitempty"` // decide: ID der Beitrittsanfrage
	Accept   bool             `json:"accept,omitempty"`  // decide: zulassen (sonst ablehnen)
}

// gameRef ist ein gemerkter Platz: Spiel-ID plus geheimes Token des Sitzes.
type gameRef struct {
	GameID string `json:"gameId"`
	Token  string `json:"token"`
}

// maxGameRefs begrenzt die Abfrage „Meine Spiele“ (Schutz vor Massenabfragen).
const maxGameRefs = 20

// gameStatus beschreibt ein gemerktes Spiel für die Startseite. Found=false: Spiel gibt es
// nicht mehr; Valid=false: Token passt zu keinem Sitz. Dann bleiben die übrigen Felder leer.
type gameStatus struct {
	GameID   string   `json:"gameId"`
	Found    bool     `json:"found"`
	Valid    bool     `json:"valid"`
	Pending  bool     `json:"pending"` // Beitrittsanfrage wartet noch auf den Gastgeber
	Started  bool     `json:"started"`
	Over     bool     `json:"over"`
	Year     int      `json:"year,omitempty"`
	Name     string   `json:"name,omitempty"`
	Country  int      `json:"country,omitempty"`
	Female   bool     `json:"female,omitempty"`
	Title    int      `json:"title,omitempty"`
	Alive    bool     `json:"alive"`
	Host     bool     `json:"host"`
	MyTurn   bool     `json:"myTurn"`   // eigene Phase noch nicht abgeschlossen
	Deadline int64    `json:"deadline"` // Ende der Zugzeit (Unix), 0 = unbegrenzt
	Winner   string   `json:"winner,omitempty"`
	Players  []string `json:"players"` // Namen aller Herrscher am Tisch
}

type gamesMsg struct {
	T     string       `json:"t"`
	Games []gameStatus `json:"games"`
}

// Ausgehende Nachrichten. Nach dem ersten Vollstand schickt der Server nur Änderungen:
// „public“ enthält nur geänderte Einträge, „self“ nur an betroffene Spieler.
type welcomeMsg struct {
	T      string `json:"t"`
	GameID string `json:"gameId"`
	Token  string `json:"token"`
	Seat   int    `json:"seat"`
	Host   bool   `json:"host"`
}

type lobbyPlayer struct {
	Name    string `json:"name"`
	Country int    `json:"country"`
	Female  bool   `json:"female"`
	Online  bool   `json:"online"`
	Bot     string `json:"bot"` // Spielweise eines Computer-Herrschers
}

type lobbyMsg struct {
	T        string          `json:"t"`
	GameID   string          `json:"gameId"`
	Started  bool            `json:"started"`
	Joinable bool            `json:"joinable"` // Beitritt (auf Anfrage) noch möglich
	Settings engine.Settings `json:"settings"`
	Players  []lobbyPlayer   `json:"players"`
}

type publicMsg struct {
	T       string                `json:"t"`
	Year    int                   `json:"year"`
	Over    bool                  `json:"over"`
	Winner  int                   `json:"winner"`
	Full    bool                  `json:"full"` // true = komplette Liste, sonst nur Änderungen
	Players []engine.PublicPlayer `json:"players"`
	Area    int                   `json:"countryArea"` // Hektar je Land für die Kartenanteile
	Pacts   []engine.Pact         `json:"pacts"`       // Verteidigungsabkommen (öffentlich)
}

type selfMsg struct {
	T    string          `json:"t"`
	Self engine.SelfView `json:"self"`
}

type eventsMsg struct {
	T      string         `json:"t"`
	Events []engine.Event `json:"events"`
}

type errorMsg struct {
	T       string `json:"t"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// maxName: Länge des Spielernamens.
const maxName = 20

// cleanName prüft Spielernamen (1–20 Zeichen) mit dem gemeinsamen Textfilter (engine/text.go).
func cleanName(s string) (string, engine.TextProblem) { return engine.CleanText(s, maxName, false) }

// Spiel-IDs ohne verwechselbare Zeichen (kein 0/O, 1/I).
const idAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newGameID() string {
	b := make([]byte, 6)
	rand.Read(b)
	for i := range b {
		b[i] = idAlphabet[int(b[i])%len(idAlphabet)]
	}
	return string(b)
}

func validGameID(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune(idAlphabet, c) {
			return false
		}
	}
	return true
}

// newToken erzeugt das geheime Wiedereinstiegs-Token eines Sitzes.
func newToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
