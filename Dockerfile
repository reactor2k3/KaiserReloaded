# KaiserReloaded – ein komplettes Image mit Spielserver und gebautem Client (keine weiteren Dienste).
#   docker build -t kaiserreloaded .
#   docker run -d -p 8080:8080 -v kaiser-data:/var/lib/kaiser kaiserreloaded
# Mehrstufig: Client mit Node bauen, Server mit Go bauen, dann nur die Ergebnisse in ein
# minimales Laufzeit-Image (distroless, ohne Shell, läuft als Nicht-root).

# 1. Client (TypeScript/Svelte) bauen
FROM node:24-alpine AS client
WORKDIR /src/client
COPY client/package.json client/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY client/ ./
RUN npm run build

# 2. Server (Go) bauen – reiner Go-SQLite-Treiber, daher ohne cgo statisch
FROM golang:1.27-alpine AS server
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
# timetzdata: Zeitzonendaten ins Programm, damit TZ (z. B. Europe/Berlin) im distroless-Image wirkt
RUN CGO_ENABLED=0 go build -trimpath -tags timetzdata -ldflags="-s -w" -o /out/kaiserd ./cmd/kaiserd \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/healthcheck ./cmd/healthcheck
# Datenordner vorbereiten, damit das Volume beim ersten Start dem Nicht-root-Nutzer gehört
RUN mkdir -p /out/var/lib/kaiser/media

# 3. Laufzeit-Image
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=server /out/kaiserd /out/healthcheck /app/
COPY --from=client /src/client/dist /app/public
COPY data/ /app/data/
COPY --from=server --chown=65532:65532 /out/var/lib/kaiser /var/lib/kaiser
VOLUME ["/var/lib/kaiser"]
EXPOSE 8080
USER nonroot
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD ["/app/healthcheck"]
# Client und WebSocket kommen vom selben Server und Port, daher keine fremden Origins (-origins leer).
# kaiserd setzt Sicherheits-Header, gzip und Cache-Regeln selbst (internal/server/web.go).
ENTRYPOINT ["/app/kaiserd", "-addr", "0.0.0.0:8080", "-data", "/app/data", "-static", "/app/public", \
            "-db", "/var/lib/kaiser/kaiser.db", "-media", "/var/lib/kaiser/media", "-origins", ""]
