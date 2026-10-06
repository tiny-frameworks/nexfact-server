# ---------------------------------------------------
# Stage 1: Build
# ---------------------------------------------------
FROM golang:1.23-alpine AS builder

# Git & Zertifikate für den Build bereitstellen
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Modul-Dateien kopieren für optimales Caching
COPY go.mod go.sum ./
RUN go mod download

# Quellcode kopieren
COPY . .

# Statisch gebundene Go-Binary ohne CGO kompilieren
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/nexfact-server \
    ./cmd/nexfact-server

# ---------------------------------------------------
# Stage 2: Runtime Environment
# ---------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Non-root Benutzer für maximale Sicherheit anlegen
RUN addgroup -S nexfact && adduser -S nexfact -G nexfact

# Arbeits- und Datenverzeichnisse erstellen
WORKDIR /app
RUN mkdir -p /app/watch /app/processed /app/error /app/env \
    && chown -R nexfact:nexfact /app

# Kompiliertes Binary aus Stage 1 kopieren
COPY --from=builder /app/nexfact-server /app/nexfact-server

# Standard-Konfigurationsdatei kopieren (falls im Repo vorhanden)
COPY --chown=nexfact:nexfact config.yaml /app/config.yaml

# Zu Unprivileged User wechseln
USER nexfact

# P2P WebSocket RPC Port freigeben
EXPOSE 8080

# Lautstärken/Volumes für persisten Speicher deklarieren
VOLUME ["/app/watch", "/app/processed", "/app/error", "/app/env"]

# Container-Startbefehl
ENTRYPOINT ["/app/nexfact-server"]
CMD ["-cfgFile", "/app/config.yaml"]

