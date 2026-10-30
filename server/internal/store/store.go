// Package store speichert Spielstände in SQLite, damit ein Neustart des Servers
// keine laufenden Spiele verliert. Gespeichert wird je Spiel ein vollständiger
// Schnappschuss (Engine-JSON inkl. Zufallszustand) plus die Sitz-Tokens.
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // reiner Go-Treiber, kein cgo nötig
)

// Store kapselt die Datenbank.
type Store struct{ db *sql.DB }

// Record ist ein gespeichertes Spiel.
type Record struct {
	ID       string
	Started  bool
	Tokens   []string
	Snapshot []byte
	Updated  int64 // Unix-Zeit der letzten Speicherung (nur beim Laden gesetzt)
}

// Open öffnet bzw. legt die Datenbank an.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("sqlite öffnen: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite schreibt ohnehin seriell
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS games (
		id       TEXT PRIMARY KEY,
		started  INTEGER NOT NULL,
		tokens   TEXT    NOT NULL,
		snapshot BLOB    NOT NULL,
		updated  INTEGER NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("tabelle anlegen: %w", err)
	}
	// Ruhmeshalle: abgeschlossene Partien, je Spiel genau ein Eintrag
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS hall (
		id      TEXT PRIMARY KEY,
		ended   INTEGER NOT NULL,
		year    INTEGER NOT NULL,
		winner  TEXT    NOT NULL,
		players TEXT    NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("ruhmeshalle anlegen: %w", err)
	}
	return &Store{db: db}, nil
}

// HallPlayer ist ein Teilnehmer einer abgeschlossenen Partie.
type HallPlayer struct {
	Name    string `json:"name"`
	Country int    `json:"country"`
	Female  bool   `json:"female"`
	Title   int    `json:"title"`
	Bot     bool   `json:"bot"`
	Winner  bool   `json:"winner"`
}

// HallEntry ist eine abgeschlossene Partie.
type HallEntry struct {
	ID      string       `json:"id"`
	Ended   int64        `json:"ended"`
	Year    int          `json:"year"`
	Winner  string       `json:"winner"` // Name des Kaisers, leer = niemand gekrönt
	Players []HallPlayer `json:"players"`
}

// SaveHall trägt eine beendete Partie ein (mehrfaches Speichern überschreibt).
func (s *Store) SaveHall(e HallEntry) error {
	players, err := json.Marshal(e.Players)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO hall (id, ended, year, winner, players) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET ended = excluded.ended, year = excluded.year,
		winner = excluded.winner, players = excluded.players`,
		e.ID, e.Ended, e.Year, e.Winner, string(players))
	return err
}

// Hall liefert die jüngsten Einträge der Ruhmeshalle.
func (s *Store) Hall(limit int) ([]HallEntry, error) {
	rows, err := s.db.Query(`SELECT id, ended, year, winner, players FROM hall ORDER BY ended DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HallEntry
	for rows.Next() {
		var e HallEntry
		var players string
		if err := rows.Scan(&e.ID, &e.Ended, &e.Year, &e.Winner, &players); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(players), &e.Players); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Close schließt die Datenbank.
func (s *Store) Close() error { return s.db.Close() }

// Save legt ein Spiel an oder überschreibt es (vorbereitete Anweisung, gebundene Parameter).
func (s *Store) Save(r Record) error {
	tokens, err := json.Marshal(r.Tokens)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO games (id, started, tokens, snapshot, updated) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET started = excluded.started, tokens = excluded.tokens,
		snapshot = excluded.snapshot, updated = excluded.updated`,
		r.ID, r.Started, string(tokens), r.Snapshot, time.Now().Unix())
	return err
}

// LoadAll liefert alle gespeicherten Spiele.
func (s *Store) LoadAll() ([]Record, error) {
	rows, err := s.db.Query(`SELECT id, started, tokens, snapshot, updated FROM games`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var r Record
		var tokens string
		if err := rows.Scan(&r.ID, &r.Started, &tokens, &r.Snapshot, &r.Updated); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tokens), &r.Tokens); err != nil {
			return nil, fmt.Errorf("tokens von %s: %w", r.ID, err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
