package server

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync"
)

// Web-Hülle für den Betrieb ohne Reverse-Proxy (ein einzelnes Docker-Image): Sicherheits-Header,
// gzip für Textdateien und Cache-Regeln für den gebauten Client. WebSocket (/ws) und Uploads
// (/media/, eigene Header) bleiben unberührt.

// Dateitypen, die Go nicht eingebaut kennt; im distroless-Image fehlt /etc/mime.types, und mit
// „nosniff“ würden Browser falsch typisierte Dateien (Favicon, Manifest) verwerfen.
func init() {
	mime.AddExtensionType(".ico", "image/x-icon")
	mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

// gzipTypes: Endungen, die sich lohnen zu komprimieren (Bilder wie PNG sind schon komprimiert).
var gzipTypes = []string{".js", ".css", ".html", ".json", ".svg", ".webmanifest", ".map", ".txt", ".md"}

var gzPool = sync.Pool{New: func() any { w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed); return w }}

// gzipWriter leitet den Körper durch gzip; Content-Length entfällt, weil sie sich ändert.
type gzipWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (g *gzipWriter) WriteHeader(code int) {
	g.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipWriter) Write(b []byte) (int, error) { return g.gz.Write(b) }

// Web umhüllt den Haupt-Handler mit Headern, Kompression und Cache-Regeln.
func Web(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		// Content-Security-Policy: Skripte nur vom eigenen Server, Schriften von Google Fonts,
		// Inline-Styles erlaubt (Svelte setzt style-Attribute), keine fremden Verbindungen.
		if !strings.HasPrefix(r.URL.Path, "/media/") {
			h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; "+
				"img-src 'self' data: blob:; connect-src 'self'; media-src 'self' blob:; object-src 'none'; "+
				"base-uri 'self'; form-action 'self'; frame-ancestors 'self'")
		}
		path := r.URL.Path
		if path == "/ws" || strings.HasPrefix(path, "/media/") || r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		// Vite legt gebaute Dateien mit Inhalts-Hash unter /assets/ ab: dürfen dauerhaft gecacht werden.
		// Alles andere (index.html, Wappen, Lizenzen) bei jedem Laden neu prüfen.
		if strings.HasPrefix(path, "/assets/") {
			h.Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			h.Set("Cache-Control", "no-cache")
		}
		if !compressible(path) || !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz := gzPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() { gz.Close(); gzPool.Put(gz) }()
		h.Set("Content-Encoding", "gzip")
		h.Add("Vary", "Accept-Encoding")
		next.ServeHTTP(&gzipWriter{ResponseWriter: w, gz: gz}, r)
	})
}

// compressible: Textdateien und die Startseite („/“ liefert index.html).
func compressible(path string) bool {
	if strings.HasSuffix(path, "/") {
		return true
	}
	for _, ext := range gzipTypes {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
