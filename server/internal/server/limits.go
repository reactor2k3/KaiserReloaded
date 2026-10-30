package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Grenzen und Schutz vor Missbrauch. Alle Werte sind bewusst großzügig für normale Spieler und
// eng für automatisierte Angriffe (viele Verbindungen, Nachrichtenfluten, Spiele-Anlage am Fließband).

// Limits sind die einstellbaren Obergrenzen des Servers (Flags in cmd/kaiserd).
type Limits struct {
	MaxGames      int  // gleichzeitig laufende Spiele (inkl. offener Lobbys)
	MaxConnsPerIP int  // gleichzeitige WebSocket-Verbindungen je IP
	MaxConns      int  // gleichzeitige WebSocket-Verbindungen insgesamt
	TrustProxy    bool // IP aus X-Forwarded-For übernehmen (nur hinter eigenem Reverse-Proxy!)
}

// DefaultLimits: 10 Spiele, 8 Verbindungen je IP (mehrere Tabs/Spiele), 400 insgesamt.
func DefaultLimits() Limits {
	return Limits{MaxGames: 10, MaxConnsPerIP: 8, MaxConns: 400}
}

// Wann ein Spiel als „laufend“ zählt: offene Lobby höchstens 6 Stunden, gestartetes Spiel bis
// 14 Tage nach dem letzten Zug. Verwaiste Spiele blockieren so keinen Platz auf Dauer.
const (
	lobbyActiveFor = 6 * time.Hour
	gameActiveFor  = 14 * 24 * time.Hour
)

// bucket ist ein Token-Eimer: rate Ereignisse je Sekunde, höchstens burst auf einmal.
type bucket struct {
	tokens float64
	last   time.Time
}

// RateLimiter begrenzt Ereignisse je Schlüssel (z. B. IP-Adresse) mit Token-Eimern.
type RateLimiter struct {
	mu    sync.Mutex
	rate  float64
	burst float64
	b     map[string]*bucket
	now   func() time.Time
}

// NewRateLimiter: rate Ereignisse je Sekunde, burst als Vorrat.
func NewRateLimiter(rate, burst float64) *RateLimiter {
	return &RateLimiter{rate: rate, burst: burst, b: map[string]*bucket{}, now: time.Now}
}

// Allow verbraucht einen Token für key; false = Grenze erreicht.
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	bk := l.b[key]
	if bk == nil {
		// Aufräumen, damit die Tabelle bei vielen wechselnden IPs nicht wächst
		if len(l.b) > 10000 {
			for k, v := range l.b {
				if now.Sub(v.last) > time.Hour {
					delete(l.b, k)
				}
			}
		}
		bk = &bucket{tokens: l.burst, last: now}
		l.b[key] = bk
	}
	bk.tokens = min(l.burst, bk.tokens+now.Sub(bk.last).Seconds()*l.rate)
	bk.last = now
	if bk.tokens < 1 {
		return false
	}
	bk.tokens--
	return true
}

// connCounter zählt offene Verbindungen je IP und insgesamt.
type connCounter struct {
	mu    sync.Mutex
	perIP map[string]int
	total int
}

// acquire belegt einen Platz, wenn beide Grenzen es zulassen.
func (c *connCounter) acquire(ip string, perIP, total int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.perIP == nil {
		c.perIP = map[string]int{}
	}
	if c.perIP[ip] >= perIP || c.total >= total {
		return false
	}
	c.perIP[ip]++
	c.total++
	return true
}

func (c *connCounter) release(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.perIP[ip]--; c.perIP[ip] <= 0 {
		delete(c.perIP, ip)
	}
	c.total--
}

// clientIP: Absender der Anfrage. X-Forwarded-For nur, wenn ausdrücklich ein eigener Proxy davor
// steht – sonst könnte jeder Angreifer seine IP frei erfinden und die Grenzen umgehen.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Der letzte Eintrag stammt vom eigenen Proxy und ist nicht fälschbar
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// LimitHTTP begrenzt gewöhnliche HTTP-Anfragen je IP (Seiten, Bilder, API). Der WebSocket-Pfad
// hat eigene Grenzen und läuft hier nur durch.
func LimitHTTP(next http.Handler, l *RateLimiter, trustProxy bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ws" && !l.Allow(clientIP(r, trustProxy)) {
			w.Header().Set("Retry-After", "10")
			http.Error(w, "Zu viele Anfragen – bitte kurz warten.", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
