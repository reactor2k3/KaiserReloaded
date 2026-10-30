package server

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWeb: Header immer, gzip nur für Text bei Accept-Encoding, Cache je nach Pfad.
func TestWeb(t *testing.T) {
	body := strings.Repeat("Kaiser ", 200)
	h := Web(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, body) }))

	req := httptest.NewRequest("GET", "/assets/index-abc.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "gzip" || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("Header: %v", rec.Header())
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := io.ReadAll(zr); string(got) != body {
		t.Error("gzip-Inhalt falsch")
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/icon-192.png", nil))
	if rec.Header().Get("Content-Encoding") != "" || rec.Header().Get("Cache-Control") != "no-cache" ||
		rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("PNG: %v", rec.Header())
	}
	if mime.TypeByExtension(".webmanifest") != "application/manifest+json" || mime.TypeByExtension(".ico") != "image/x-icon" {
		t.Error("Dateitypen für Favicon/Manifest fehlen")
	}
}
