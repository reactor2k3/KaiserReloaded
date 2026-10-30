package engine

import (
	"encoding/json"
	"testing"
)

// Wappen: nur Landeswappen oder Upload; der frühere Baukasten wird abgelehnt, alte Spielstände
// mit Baukasten-Feldern laden aber weiter (unbekannte Felder werden ignoriert).
func TestArmsValidate(t *testing.T) {
	cases := []struct {
		arms ArmsSpec
		ok   bool
	}{
		{ArmsSpec{Kind: "country"}, true},
		{ArmsSpec{Kind: "country", Variant: 3}, true},
		{ArmsSpec{Kind: "country", Variant: MaxArmsVariants}, false},
		{ArmsSpec{Kind: "country", Variant: -1}, false},
		{ArmsSpec{Kind: "upload", Media: "0123456789abcdef0123456789abcdef.png"}, true},
		{ArmsSpec{Kind: "upload", Media: "../etc/passwd"}, false},
		{ArmsSpec{Kind: "builder"}, false},
	}
	for i, c := range cases {
		if got := c.arms.Validate(); got != c.ok {
			t.Errorf("Fall %d: Validate = %v, erwartet %v", i, got, c.ok)
		}
	}
	var old ArmsSpec
	if err := json.Unmarshal([]byte(`{"kind":"builder","division":"bend","tinctures":["gold","green"],"charge":"star"}`), &old); err != nil {
		t.Fatal(err)
	}
	if old.Kind != "builder" {
		t.Errorf("alter Spielstand: %+v", old)
	}
}
