package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"kaiserreloaded/server/internal/media"
)

// UploadHandler nimmt Spielerbild oder Wappen per POST (multipart) an.
// Felder: gameId, token, kind (avatar | arms), file. Das Token belegt den Sitz.
func (h *Hub) UploadHandler(store *media.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply := func(code int, v any) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(v)
		}
		fail := func(code int, msg string) { reply(code, map[string]string{"message": msg}) }

		if r.Method != http.MethodPost {
			fail(http.StatusMethodNotAllowed, "Nur POST erlaubt.")
			return
		}
		if !h.uploadLimit.Allow(clientIP(r, h.limits.TrustProxy)) {
			fail(http.StatusTooManyRequests, "Zu viele Uploads – bitte kurz warten.")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, media.MaxUpload+64<<10)
		if err := r.ParseMultipartForm(media.MaxUpload + 64<<10); err != nil {
			fail(http.StatusRequestEntityTooLarge, "Die Datei ist zu groß (höchstens 1 MB).")
			return
		}
		kind := r.FormValue("kind")
		if kind != "avatar" && kind != "arms" {
			fail(http.StatusBadRequest, "Unbekannte Art des Uploads.")
			return
		}
		room := h.room(r.FormValue("gameId"))
		if room == nil {
			fail(http.StatusNotFound, "Spiel nicht gefunden.")
			return
		}
		seat := room.seatFor(r.FormValue("token"))
		if seat < 0 {
			fail(http.StatusForbidden, "Kein gültiger Platz in diesem Spiel.")
			return
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			fail(http.StatusBadRequest, "Keine Datei erhalten.")
			return
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, media.MaxUpload+1))
		if err != nil {
			fail(http.StatusBadRequest, "Datei konnte nicht gelesen werden.")
			return
		}
		name, err := store.Save(data, kind == "arms")
		if err != nil {
			msg := "Das Bild konnte nicht verarbeitet werden."
			if errors.Is(err, media.ErrUnsupported) {
				msg = "Erlaubt sind PNG, JPEG oder WebP" + map[bool]string{true: " sowie SVG für Wappen", false: ""}[kind == "arms"] + "."
			}
			slog.Info("upload abgelehnt", "err", err)
			fail(http.StatusUnsupportedMediaType, msg)
			return
		}
		done := make(chan error, 1)
		room.do(func() {
			var err error
			if kind == "avatar" {
				err = room.game.SetAvatar(seat, name)
			} else {
				err = room.game.SetUploadedArms(seat, name)
			}
			if err == nil {
				room.afterChange(nil, seat)
			}
			done <- err
		})
		if err := <-done; err != nil {
			fail(http.StatusBadRequest, "Das Bild konnte nicht übernommen werden.")
			return
		}
		reply(http.StatusOK, map[string]string{"media": name})
	})
}

// seatFor sucht den Sitz zu einem Token (zeitkonstanter Vergleich), -1 wenn keiner passt.
func (r *Room) seatFor(token string) int {
	res := make(chan int, 1)
	r.do(func() {
		for seat, t := range r.tokens {
			if subtle.ConstantTimeCompare([]byte(t), []byte(token)) == 1 {
				res <- seat
				return
			}
		}
		res <- -1
	})
	return <-res
}
