<sup>🌍 **Language:** 🇩🇪 [German →](README.de.md)</sup>

---

[![Codeberg Release](https://img.shields.io/codeberg/v/release/tiny-frameworks/nexfact-server?logo=codeberg&logoColor=white)](https://codeberg.org/tiny-frameworks/nexfact-server)
[![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

| | |
|----|----|
|![GSF-NexFact-Logo](nexfact.png)| ***GSF-NexFact-Server***<br>the service host and application adapter for the **GSF-Nexfact** invoice processing engine.|

<sup>***GSF*** stands for ***Go Small Frameworks*** — minimalist tools for robust applications.</sup>

---

### Repository

This project is developed and maintained on Codeberg.

**Canonical repository:**

> https://codeberg.org/tiny-frameworks/nexfact-server

GitHub is an automated mirror of the Codeberg repository:

**GitHub mirror:**

> https://github.com/tiny-frameworks/nexfact-server

---

## GSF-NexFact-Server

`GSF-Nexfact-Server` is the service host and application adapter for the **GSF-Nexfact** invoice processing engine. It exposes the core module across multiple inbound communication channels—specifically P2P JSON-RPC over WebSockets and atomic local directory polling—while managing configuration, logging, and application lifecycle.

This project is part of the **tiny-frameworks** initiative.

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

* **P2P WebSocket RPC:** Enables synchronous invoice processing via the `nexutils/p2p/rpc` protocol (e.g., for Pharo Smalltalk via `nexutils-st` or remote Go peers).
* **Directory Watcher:** Atomically monitors a configurable input folder for `.json` job files for automated batch processing.
* **Graceful Shutdown:** Ensures active jobs complete cleanly and directories are left in a consistent state upon receiving OS signals (`SIGINT`, `SIGTERM`).

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

## Configuration (3-Stage Precedence)

Configuration follows a 3-stage hierarchy (**higher stages override lower ones**):

1. **Environment Variables** (Highest Precedence)
2. **Configuration File** (`config.json`)
3. **Defaults** (Hardcoded fallback values)

### Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `NEXFACT_ENV_DIR` | `./env` | Path to the nexfact environment directory |
| `NEXFACT_RPC_ADDR` | `:9090` | Listening address for the P2P WebSocket RPC server |
| `NEXFACT_HEARTBEAT_INT` | `30s` | Interval for P2P peer heartbeats |
| `NEXFACT_WATCH_DIR` | `./data/input` | Input folder for the directory watcher |
| `NEXFACT_PROCESSED_DIR` | `./data/processed` | Destination folder for successfully processed jobs |
| `NEXFACT_ERROR_DIR` | `./data/error` | Destination folder for failed jobs |
| `NEXFACT_POLL_INT` | `2s` | Polling interval for the directory watcher |
| `NEXFACT_LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |

### Configuration File (`config.json`)

Optionally, a `config.json` file can be passed via command-line flags or placed in the working directory:

```json
{
  "env_dir": "/var/lib/nexfact/env",
  "rpc_addr": "0.0.0.0:9090",
  "heartbeat_interval": "30s",
  "watch_dir": "./data/input",
  "processed_dir": "./data/processed",
  "error_dir": "./data/error",
  "poll_interval": "2s",
  "logger": {
    "level": "info"
  }
}

```

---

## Getting Started

### Prerequisites

* **Go:** 1.26.0 or newer
* **Workspace:** Recommended setup using a multi-module workspace via `go.work` alongside `nexfact` and `nexutils`.

### Building & Running

**Build locally:**

```bash
go build -o bin/nexfact-server ./cmd/nexfact-server

```

**Run with default configuration:**

```bash
./bin/nexfact-server

```

**Run with an explicit configuration file:**

```bash
./bin/nexfact-server -config /path/to/config.json

```

---

## Testing

The test suite utilizes isolated temporary directories and dynamic ports:

```bash
# Run all tests with the Go race detector enabled
go test -v -race ./...

```

For local multi-module testing, a `go.work` file in the parent workspace directory is recommended:

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

Licensed under the **Apache License 2.0**.

---

