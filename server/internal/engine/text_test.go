package engine

import "testing"

// TestCleanText: normale Namen erlaubt, unsichtbare Zeichen, Zalgo und Links abgelehnt.
func TestCleanText(t *testing.T) {
	cases := []struct {
		in   string
		want TextProblem
	}{
		{"Friedrich der Große", TextOK},
		{"  Graf   von  Württemberg ", TextOK},
		{"St. Gallen", TextOK},
		{"Kaiser.Max", TextOK},
		{"Nguyễn", TextOK},
		{"", TextLength},
		{"abcdefghijklmnopqrstu", TextLength},
		{"Max‮redli", TextChars}, // Bidi-Override dreht die Anzeige
		{"Ma​x", TextChars},      // Zero-Width-Space
		{"Ma\x00x", TextChars},   // Steuerzeichen
		{"Ź̂̃alg", TextChars},   // drei gestapelte Akzente
		{"www.spam", TextLink},
		{"https://x", TextLink},
		{"kaufen.com", TextLink},
		{"spam (dot) de", TextLink},
		{"Ma\xffx", TextChars}, // kein gültiges UTF-8
	}
	for _, c := range cases {
		if _, got := CleanText(c.in, 20, false); got != c.want {
			t.Errorf("%q: %d, erwartet %d", c.in, got, c.want)
		}
	}
	if s, p := CleanText("", 40, true); p != TextOK || s != "" {
		t.Error("leerer Wahlspruch muss erlaubt sein")
	}
}
