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

`GSF-Nexfact-Server` is the service host and application adapter for the **GSF-Nexfact** invoice
engine. It provides access to the core module via multiple inbound channels (P2P JSON-RPC over
WebSockets as well as local directory polling) and manages configuration, logging, and lifecycle operations.

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
│       ├── main.go              # Application Entrypoint & Signal Handling
│       └── config.yaml.example  # Sample configuration file
├── internal/
│   ├── config/                  # 3-Stage Configuration Management
│   ├── p2p/                     # P2P JSON-RPC Delegate & Adapter
│   ├── server/                  # Lifecycle Manager (P2P Node + Watcher)
│   └── watcher/                 # Atomic Directory Watcher
├── go.mod
└── README.md

```

---

## Configuration (4-Stage Precedence)

`nexfact-server` uses a 4-tier configuration precedence model (**higher priority overrides lower priority**):

1. **CLI Flags** (Highest Priority)
2. **Environment Variables** (`NEXFACT_`)
3. **YAML Configuration File** (`config.yaml`)
4. **Code Defaults** (Fallback values inside the application)

---

### Configuration File (`config.yaml`)

By default, the application looks for a `config.yaml` file in the current working directory (or at the path specified via `-cfgFile`).

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

### Precedence & Parameter Matrix

| CLI Flag | Environment Variable | Config YAML Key | Default | Description |
| --- | --- | --- | --- | --- |
| `-cfgFile` | — | — | `config.yaml` | Path to the YAML configuration file |
| `-env` | `NEXFACT_ENV_DIR` | `env_dir` | `./env` | Path to the `nexfact` environment directory |
| `-rpc-addr` | `NEXFACT_RPC_ADDR` | `rpc_addr` | `127.0.0.1:8080` | Address for P2P WebSocket RPC binding |
| `-watch-dir` | `NEXFACT_WATCH_DIR` | `watch_dir` | `./watch` | Directory monitored for incoming invoices |
| `-processed-dir` | `NEXFACT_PROCESSED_DIR` | `processed_dir` | `./processed` | Directory where successfully processed invoices are moved |
| `-error-dir` | `NEXFACT_ERROR_DIR` | `error_dir` | `./error` | Directory where failed invoices are moved |
| `-poll-interval` | `NEXFACT_POLL_INTERVAL` | `poll_interval` | `2s` | Polling interval for directory watcher (e.g. `2s`, `500ms`) |
| `-heartbeat` | `NEXFACT_HEARTBEAT_INTERVAL` | `heartbeat_interval` | `5s` | Heartbeat interval for P2P WebSocket nodes |
| `-log-level` | `NEXFACT_LOG_LEVEL` | `log_level` | `info` | Log severity level (`debug`, `info`, `warn`, `error`) |

---

### Example Usage

#### Using CLI Overrides

Override specific parameters directly from the terminal without changing the configuration file:

```bash
./nexfact-server -cfgFile /etc/nexfact/config.yaml -log-level debug -poll-interval 1s

```

#### Using Environment Variables

Ideal for Docker container deployments:

```bash
export NEXFACT_RPC_ADDR="0.0.0.0:8080"
export NEXFACT_LOG_LEVEL="warn"
./nexfact-server

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

## Running Integration Tests

`nexfact-server` relies on reference environments, schemas, and test fixtures provided by the shared **`nexfact-assets`** repository.

To run the integration tests locally, ensure `nexfact-assets` is cloned alongside `nexfact-server`. During test execution, `nexfact-assets/setup` automatically copies the required test fixtures into an isolated, temporary directory (`t.TempDir()`) to guarantee side-effect-free test runs:

```go
import "codeberg.org/tiny-frameworks/nexfact-assets/setup"

func TestMyService(t *testing.T) {
    envRoot := setup.SetupTestEnv(t, setup.NATIVE)
    // ... test executes against isolated tempDir
}

---

## Organizational & Standards

* **Copyright:** © 2026 Georg Hagn.
* **Repository:** `codeberg.org/tiny-frameworks/nexfact-server`
* **License:** Apache License, Version 2.0.

*GSF-NexFact-Server is an independent open-source project and is not affiliated with any corporation of a similar name.*

---

## Contact & Support

For inquiries, architectural discussions, or security issues, please contact:

📧 **georghagn [at] tiny-frameworks.io**

---

