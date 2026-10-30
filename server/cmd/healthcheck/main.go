// Command healthcheck prüft im Container, ob kaiserd antwortet (distroless hat kein curl).
package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	c := http.Client{Timeout: 2 * time.Second}
	res, err := c.Get("http://127.0.0.1:8080/healthz")
	if err != nil || res.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
