package engine

import (
	"regexp"
	"strings"
)

// ArmsSpec beschreibt das Wappen eines Herrschers: Landeswappen oder hochgeladene Datei.
// Den früheren Baukasten gibt es nicht mehr; alte Spielstände mit Kind "builder" zeigen das
// Landeswappen (Client), neue Baukasten-Wappen lehnt Validate ab.
type ArmsSpec struct {
	Kind    string `json:"kind"`    // country | upload
	Variant int    `json:"variant"` // Landeswappen: gewählte Variante (0 = Vorgabe); immer senden (Client führt zusammen)
	Media   string `json:"media"`   // Upload: Dateiname unter /media/
}

// MaxArmsVariants begrenzt die Auswahl je Land (Dateien <land>-<n>.svg im Client).
const MaxArmsVariants = 8

var mediaName = regexp.MustCompile(`^[0-9a-f]{32}\.(png|svg)$`)

var ErrBadProfile = &RuleError{"bad_profile", "Ungültige Profilangaben."}

// MaxMotto: Länge des Wahlspruchs.
const MaxMotto = 40

// CleanMotto prüft den Wahlspruch (darf leer sein) mit dem gemeinsamen Textfilter (text.go).
func CleanMotto(s string) (string, TextProblem) { return CleanText(s, MaxMotto, true) }

// ValidMedia prüft einen Dateinamen aus dem Upload (Zufalls-ID + Endung).
func ValidMedia(name string) bool { return mediaName.MatchString(name) }

// Validate prüft ein Wappen gegen die erlaubten Werte.
func (a ArmsSpec) Validate() bool {
	switch a.Kind {
	case "country":
		return a.Variant >= 0 && a.Variant < MaxArmsVariants
	case "upload":
		return ValidMedia(a.Media)
	}
	return false
}

// setProfile ändert Wahlspruch und Wappen (in jeder Phase erlaubt).
func setProfile(p *Player, motto string, arms *ArmsSpec) error {
	m, problem := CleanMotto(motto)
	if problem != TextOK {
		return &RuleError{"bad_motto", TextMessage(problem, "Der Wahlspruch", MaxMotto)}
	}
	if arms != nil {
		if !arms.Validate() {
			return ErrBadProfile
		}
		p.Arms = *arms
	}
	p.Motto = m
	return nil
}

// SetAvatar setzt ein hochgeladenes Spielerbild (nur Rasterbild, kein SVG).
func (g *Game) SetAvatar(seat int, media string) error {
	if seat < 0 || seat >= len(g.Seats) || !ValidMedia(media) || strings.HasSuffix(media, ".svg") {
		return ErrBadProfile
	}
	g.Seats[seat].Player.Avatar = media
	return nil
}

// SetUploadedArms setzt ein hochgeladenes Wappen.
func (g *Game) SetUploadedArms(seat int, media string) error {
	if seat < 0 || seat >= len(g.Seats) || !ValidMedia(media) {
		return ErrBadProfile
	}
	g.Seats[seat].Player.Arms = ArmsSpec{Kind: "upload", Media: media}
	return nil
}
