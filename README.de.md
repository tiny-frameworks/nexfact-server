<sup>🌍 **Sprache:** 🇬🇧 [Englisch →](README.md)</sub>

---

[![Codeberg Release](https://img.shields.io/codeberg/v/release/tiny-frameworks/nexfact-server?logo=codeberg&logoColor=white)](https://codeberg.org/tiny-frameworks/nexfact-server)
[![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

| | |
|----|----|
|![GSF-NexFact-Logo](nexfact.png)| ***GSF-NexFact-Server***<br>der Service Host und Anwendungs-Adapter für die **GSF-Nexfact** Invoice-Engine.|

<sup>***GSF*** steht für ***Go Small Frameworks*** — minimalistische Tools für robuste Applicationen.</sup>

---

### Repository

Das Projekt wird auf Codeberg entwickelt und gewartet.

**Canonical repository:**

> https://codeberg.org/tiny-frameworks/nexfact-server

GitHub ist ein automatisierter Mirror des Codeberg Repositories

**GitHub mirror:**

> https://github.com/tiny-frameworks/nexfact-server

---

## GSF-NexFact-Server

`GSF-Nexfact-Server` ist der Service Host und Anwendungs-Adapter für die **GSF-Nexfact** Invoice-Engine. Er stellt das Kern-Modul über verschiedene Inbound-Kanäle (P2P JSON-RPC über WebSockets sowie lokales Verzeichnis-Polling) bereit und kümmert sich um Konfiguration, Logging und Lifecycle-Management.

Das Projekt ist Teil der **tiny-frameworks** Initiative.

---

## Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                           nexfact-server                                │
│                                                                         │
│   ┌───────────────────────────┐         ┌────────────────────────────┐  │
│   │   P2P / RPC Adapter       │         │  Directory Watcher Adapter │  │
│   │ (OrchestratorDelegate)    │         │   (Input-Folder Polling)   │  │
│   └─────────────┬─────────────┘         └──────────────┬─────────────┘  │
│                 │                                      │                │
│                 └──────────────────┬───────────────────┘                │
│                                    │                                    │
│                                    ▼                                    │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                         nexfact Core                            │   │
│   │                   (orchestrator.Orchestrator)                   │   │
│   └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘

```

* **P2P WebSocket RPC:** Ermöglicht die synchrone Rechnungsverarbeitung über das `nexutils/p2p/rpc`-Protokoll (z. B. für Pharo Smalltalk via `nexutils-st` oder Go-Peers).
* **Directory Watcher:** Überwacht ein konfigurierbares Eingangsverzeichnis atomar auf `.json`-Aufträge für automatisierte Batch-Verarbeitungen.
* **Graceful Shutdown:** Garantiert bei OS-Signalen (`SIGINT`, `SIGTERM`), dass laufende Jobs vollständig beendet und Verzeichnisse sauber hinterlassen werden.

---

## Directory Structure

```text
nexfact-server/
├── cmd/
│   └── nexfact-server/
│       └── main.go          # Application Entrypoint & Signal Handling
├── internal/
│   ├── config/              # 3-Stage Configuration Management
│   ├── p2p/                 # P2P JSON-RPC Delegate & Adapter
│   ├── server/              # Lifecycle Manager (P2P Node + Watcher)
│   └── watcher/             # Atomic Directory Watcher
├── config.json.example      # Sample configuration file
├── go.mod
└── README.md

```

---

## Konfiguration (4-Stufen Hierarchie Modell)

`nexfact-server` verwendet ein vierstufiges Modell für die Konfigurationspriorität (**höhere Priorität überschreibt niedrigere**):

1. **CLI-Flags** (höchste Priorität)
2. **Umgebungsvariablen** (`NEXFACT_`)
3. **YAML-Konfigurationsdatei** (`config.yaml`)
4. **Code-Standardwerte** (Fallback-Werte innerhalb der Anwendung)

---

### Konfigurations Datei (`config.yaml`)

Standardmäßig sucht die Anwendung nach einer `config.yaml`-Datei im aktuellen
Arbeitsverzeichnis (oder an dem über `-cfgFile` angegebenen Pfad).

```yaml
# nexfact-server configuration
env_dir: "./env"
rpc_addr: "127.0.0.1:8080"

# Directory Watcher paths
watch_dir: "./watch"
processed_dir: "./processed"
error_dir: "./error"

# Time intervals (parsed natively as Go time.Duration)
poll_interval: 2s
heartbeat_interval: 5s

# Logging
log_level: "info"

```

---

### Rangfolge- und Parametermatrix

| CLI Variable | Environment Variable | Config YAML Key | Default | Description |
| --- | --- | --- | --- | --- |
| `-cfgFile` | — | — | `config.yaml` | Pfad zur YAML Konfigurationsdatei |
| `-env` | `NEXFACT_ENV_DIR` | `env_dir` | `./env` | Pfad zum `nexfact` environment Directory |
| `-rpc-addr` | `NEXFACT_RPC_ADDR` | `rpc_addr` | `127.0.0.1:8080` | Addresse für P2P WebSocket RPC binding |
| `-watch-dir` | `NEXFACT_WATCH_DIR` | `watch_dir` | `./watch` | überwachtes Verzeichnis für eingehende Rechnungen |
| `-processed-dir` | `NEXFACT_PROCESSED_DIR` | `processed_dir` | `./processed` | Verzeichnis, in das erfolgreich verarbeitete Rechnungen verschoben werden |
| `-error-dir` | `NEXGFACT_ERROR_DIR` | `error_dir` | `./error` | Verzeichnis, in das fehlerhafte Rechnungen verschoben werden |
| `-poll-interval` | `NEXFACT_POLL_INTERVAL` | `poll_interval` | `2s` | Polling Interval für directory watcher (e.g. `2s`, `500ms`) |
| `-heartbeat` | `NEXFACT_HEARTBEAT_INTERVAL` | `heartbeat_interval` | `5s` | Heartbeat Interval für P2P WebSocket nodes |
| `-log-level` | `NEXFACT_LOG_LEVEL` | `log_level` | `info` | Log severity level (`debug`, `info`, `warn`, `error`) |

---

### Anwendungsbeispiel

#### CLI-Variable

CLI Parameter überschreiben bestimmte Parameter direkt über das Terminal, ohne die Konfigurationsdatei zu ändern:

```bash
./nexfact-server -cfgFile /etc/nexfact/config.yaml -log-level debug -poll-interval 1s

```

### Environment Variable

Ideal for Docker container deployments:

```bash
export NEXGATE_RPC_ADDR="0.0.0.0:8080"
export NEXGATE_LOG_LEVEL="warn"
./nexfact-server

```

---

## Getting Started

### Prerequisites

* **Go:** 1.26.0 oder neuer
* **Workspace:** Empfohlen im Multi-Module Workspace via `go.work` zusammen mit `nexfact` und `nexutils`.

### Building & Running

**Lokal bauen:**

```bash
go build -o bin/nexfact-server ./cmd/nexfact-server

```

**Starten mit Standard-Konfiguration:**

```bash
./bin/nexfact-server

```

**Starten mit explizitem Config-File:**

```bash
./bin/nexfact-server -config /path/to/config.json

```

---

## Testing

Die Test-Suite nutzt isolierte Temp-Ordner und dynamische Ports:

```bash
# Alle Tests inklusive Race-Detector ausführen
go test -v -race ./...

```

Für lokale Multi-Module Tests empfiehlt sich eine `go.work`-Datei im übergeordneten Verzeichnis:

```go
go 1.26.0

use (
	./nexfact
	./nexfact-server
	./nexutils
)

```

---

## License

Copyright (c) 2026 Georg Hagn.

Lizenziert unter der **Apache License 2.0**.

---
