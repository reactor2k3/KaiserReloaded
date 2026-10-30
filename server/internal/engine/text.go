package engine

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Prüfung freier Texte (Spielernamen, Wahlsprüche), die andere Spieler zu sehen bekommen.
// HTML-Escaping übernimmt der Client (Svelte setzt Text nie als HTML); hier geht es um
// Missbrauch, den Escaping nicht abfängt: unsichtbare oder richtungsdrehende Zeichen,
// „Zalgo“-Text aus gestapelten Akzenten und Werbe-Links.

// TextProblem sagt, warum ein Text abgelehnt wurde.
type TextProblem int

const (
	TextOK     TextProblem = iota
	TextLength             // leer oder zu lang
	TextChars              // Steuer-, Format- oder unbelegte Zeichen, gestapelte Akzente
	TextLink               // Adresse oder Link
)

// maxMarks: so viele Akzente darf ein Buchstabe tragen (Vietnamesisch braucht zwei).
const maxMarks = 2

// linkPattern erkennt Adressen: Schema, „www.“, Name mit gängiger Endung direkt am Punkt
// („kaiser.de“) oder verschleiert („kaiser (dot) de“). Bewusst nur typische Endungen ohne
// Leerzeichen am Punkt, damit Namen wie „St. Gallen“ oder „Kaiser.Max“ erlaubt bleiben.
var linkPattern = regexp.MustCompile(`(?i)(://|www\.|[a-z0-9-](\.|\s*[(\[]dot[)\]]\s*)(com|net|org|de|at|ch|eu|ru|io|xyz|info|top|biz|gg|ly|tk|cc|app|link|shop|site|online|club)\b)`)

// CleanText normalisiert Leerraum und prüft einen Text auf 1..max Zeichen (mit allowEmpty
// auch leer). Abgelehnt werden unsichtbare Zeichen (Steuer-, Format-, Privat- und unbelegte
// Zeichen; z. B. Zero-Width und Bidi-Overrides), mehr als maxMarks Akzente je Buchstabe
// und Links.
func CleanText(s string, max int, allowEmpty bool) (string, TextProblem) {
	if !utf8.ValidString(s) {
		return "", TextChars
	}
	s = strings.Join(strings.Fields(s), " ")
	if (s == "" && !allowEmpty) || utf8.RuneCountInString(s) > max {
		return "", TextLength
	}
	marks := 0
	for _, r := range s {
		switch {
		// Nur sichtbare Klassen zulassen: Buchstaben, Akzente, Ziffern, Satz- und Sonderzeichen,
		// normales Leerzeichen. Damit fallen Cc, Cf (Bidi, Zero-Width), Co, Cs und Unbelegtes weg.
		case !unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S, unicode.Zs):
			return "", TextChars
		case unicode.In(r, unicode.Mn, unicode.Me):
			if marks++; marks > maxMarks {
				return "", TextChars
			}
		default:
			marks = 0
		}
	}
	if linkPattern.MatchString(s) {
		return "", TextLink
	}
	return s, TextOK
}

// TextMessage ist die Fehlermeldung zu einem Problem; what nennt das Feld („Der Name“).
func TextMessage(p TextProblem, what string, max int) string {
	switch p {
	case TextChars:
		return what + " enthält unzulässige oder unsichtbare Zeichen."
	case TextLink:
		return what + " darf keine Internetadressen oder Links enthalten."
	}
	return what + " ist leer oder länger als " + strconv.Itoa(max) + " Zeichen."
}
