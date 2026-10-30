package server

import (
	"net/http/httptest"
	"testing"
	"time"
)

// TestRateLimiter: Vorrat wird verbraucht, füllt sich mit der Zeit wieder auf.
func TestRateLimiter(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewRateLimiter(1, 3)
	l.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if !l.Allow("a") {
			t.Fatalf("Anfrage %d abgelehnt", i)
		}
	}
	if l.Allow("a") {
		t.Error("Vorrat erschöpft, trotzdem erlaubt")
	}
	if !l.Allow("b") {
		t.Error("andere IP darf nicht betroffen sein")
	}
	now = now.Add(2 * time.Second)
	if !l.Allow("a") || !l.Allow("a") || l.Allow("a") {
		t.Error("Auffüllen falsch")
	}
}

// TestConnCounter: Grenze je IP und insgesamt.
func TestConnCounter(t *testing.T) {
	var c connCounter
	if !c.acquire("x", 2, 3) || !c.acquire("x", 2, 3) || c.acquire("x", 2, 3) {
		t.Error("Grenze je IP")
	}
	if !c.acquire("y", 2, 3) || c.acquire("z", 2, 3) {
		t.Error("Gesamtgrenze")
	}
	c.release("x")
	if !c.acquire("z", 2, 3) {
		t.Error("Freigabe wirkt nicht")
	}
}

// TestClientIP: X-Forwarded-For nur mit ausdrücklichem Vertrauen.
func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.5:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	if clientIP(r, false) != "10.0.0.5" {
		t.Error("Header darf ohne Vertrauen nicht zählen")
	}
	if clientIP(r, true) != "5.6.7.8" {
		t.Error("letzter Eintrag des eigenen Proxys erwartet")
	}
}
