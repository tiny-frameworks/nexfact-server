// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codeberg.org/tiny-frameworks/nexfact-server/internal/config"
	"codeberg.org/tiny-frameworks/nexfact-server/internal/server"
	"codeberg.org/tiny-frameworks/nexfact/orchestrator"
	"codeberg.org/tiny-frameworks/nexutils/logger"
)

// Hilfsfunktion: Findet einen freien Port für isolierte Tests
func getFreeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on free port: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestManagerLifecycleAndShutdown(t *testing.T) {
	tempDir := t.TempDir()
	addr := getFreeAddr(t)

	cfg := config.Config{
		RPCAddr:           addr,
		HeartbeatInterval: 1 * time.Second,
		WatchDir:          tempDir + "/watch",
		ProcessedDir:      tempDir + "/processed",
		ErrorDir:          tempDir + "/error",
		PollInterval:      100 * time.Millisecond,
	}

	lgr := logger.Logger.With("test", "manager_lifecycle")

	testEnvParent := getTestEnvParent(t)
	env := filepath.Join(testEnvParent, "env")

	orch, err := orchestrator.New(env)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	mgr := server.NewManager(cfg, orch, lgr)

	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	// 1. Manager starten
	if err := mgr.Start(runCtx); err != nil {
		t.Fatalf("failed to start manager: %v", err)
	}

	// Kurze Pause, um den HTTP-Listener anlaufen zu lassen
	time.Sleep(100 * time.Millisecond)

	// 2. Prüfen, ob der RPC WebSocket Port erreichbar ist
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		t.Fatalf("RPC Node is not listening on %s: %v", addr, err)
	}
	conn.Close()

	// 3. Graceful Shutdown ausführen
	cancelRun()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelShutdown()

	if err := mgr.Stop(shutdownCtx); err != nil {
		t.Fatalf("failed to stop manager gracefully: %v", err)
	}
}

func getTestEnvParent(t *testing.T) string {
	t.Helper()

	// 1. Prüfe ob eine ENV-Variable gesetzt ist (z.B. in go.work / lokalen Tests)
	if envPath := os.Getenv("NEXFACT_TEST_ENV_PARENT"); envPath != "" {
		return envPath
	}

	// 2. Fallback auf lokales lüttes testdata Ordnerchen im Repo
	localPath := filepath.Join("testdata", "testenv")
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// 3. Fallback auf relatives nexfact Nachbar-Repo
	relPath := filepath.Join("..", "..", "..", "nexfact", "orchestrator", "testdata")
	absFile, err := filepath.Abs(relPath)
	if err != nil {
		t.Fatalf("Can not build absolute Path for %s", relPath)
	}
	if _, err := os.Stat(absFile); err == nil {
		return absFile
	}

	t.Fatalf("No valid env found for testing")
	return ""
}
