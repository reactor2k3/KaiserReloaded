// Command kaiserd ist der Spielserver: WebSocket unter /ws, statischer Client unter /.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"kaiserreloaded/server/internal/engine"
	"kaiserreloaded/server/internal/media"
	"kaiserreloaded/server/internal/server"
	"kaiserreloaded/server/internal/store"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "Adresse des HTTP-Servers")
	data := flag.String("data", "../data", "Ordner mit balance.json und world.json")
	static := flag.String("static", "../client/dist", "gebauter Client (leer = keiner)")
	dbPath := flag.String("db", "kaiser.db", "SQLite-Datei für Spielstände")
	mediaDir := flag.String("media", "media", "Ordner für hochgeladene Spielerbilder und Wappen")
	origins := flag.String("origins", "localhost:5173,127.0.0.1:5173", "erlaubte fremde Origins (Komma-Liste, Dev-Server)")
	// Grenzen gegen Überlastung und Missbrauch (internal/server/limits.go)
	lim := server.DefaultLimits()
	flag.IntVar(&lim.MaxGames, "max-games", lim.MaxGames, "höchstens so viele gleichzeitig laufende Spiele")
	flag.IntVar(&engine.MaxSeats, "max-players", engine.MaxSeats, "höchstens so viele Herrscher je Spiel")
	flag.IntVar(&lim.MaxConnsPerIP, "max-conns-ip", lim.MaxConnsPerIP, "gleichzeitige Verbindungen je IP")
	flag.IntVar(&lim.MaxConns, "max-conns", lim.MaxConns, "gleichzeitige Verbindungen insgesamt")
	flag.BoolVar(&lim.TrustProxy, "trust-proxy", false, "Client-IP aus X-Forwarded-For (nur hinter eigenem Reverse-Proxy)")
	flag.Parse()

	b, err := engine.LoadBalance(filepath.Join(*data, "balance.json"))
	if err != nil {
		slog.Error("balance", "err", err)
		os.Exit(1)
	}
	world, err := engine.LoadWorld(filepath.Join(*data, "world.json"))
	if err != nil {
		slog.Error("world", "err", err)
		os.Exit(1)
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		slog.Error("datenbank", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	hub, err := server.NewHub(b, world, splitList(*origins), st)
	if err != nil {
		slog.Error("spiele laden", "err", err)
		os.Exit(1)
	}
	hub.SetLimits(lim)

	mstore, err := media.NewStore(*mediaDir)
	if err != nil {
		slog.Error("media", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("/ws", hub)
	mux.Handle("/upload", hub.UploadHandler(mstore))
	mux.Handle("/media/", mstore)
	mux.Handle("/api/ruhmeshalle", server.HallHandler(st))
	mux.Handle("/api/status", hub.StatusHandler())
	mux.Handle("/api/games", hub.ListHandler()) // öffentliche Spieleliste
	// Lebenszeichen für Docker-Healthcheck und Reverse-Proxy
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	if *static != "" {
		if fi, err := os.Stat(*static); err == nil && fi.IsDir() {
			mux.Handle("/", http.FileServer(http.Dir(*static)))
		}
	}

	// Ohne Reverse-Proxy (ein Docker-Image) setzt der Server Header, gzip und Cache-Regeln selbst
	// HTTP-Anfragen je IP begrenzen (20/s, Vorrat 200 für das erste Laden mit allen Wappen);
	// Zeitlimits gegen langsame Angriffe, kleine Header
	handler := server.LimitHTTP(server.Web(mux), server.NewRateLimiter(20, 200), lim.TrustProxy)
	srv := &http.Server{Addr: *addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute, MaxHeaderBytes: 16 << 10}

	// Sauber beenden bei SIGTERM (docker stop) und Strg+C: offene Anfragen abschließen, dann
	// schließt main per defer die Datenbank.
	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-stop.Done()
		ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		slog.Info("kaiserd wird beendet")
		srv.Shutdown(ctx)
	}()

	slog.Info("kaiserd läuft", "addr", *addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
