// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"codeberg.org/tiny-frameworks/nexfact-server/internal/config"
	"codeberg.org/tiny-frameworks/nexfact-server/internal/server"
	"codeberg.org/tiny-frameworks/nexfact-server/internal/testsetup"
	"codeberg.org/tiny-frameworks/nexfact/orchestrator"
)

func TestManagerLifecycleAndShutdown(t *testing.T) {

	tempDir := t.TempDir()
	addr := testsetup.GetFreeAddr(t)
	envRoot, _ := testsetup.SetupTestEnv(t, testsetup.NATIVE)

	cfg := config.Config{
		RPCAddr:           addr,
		HeartbeatInterval: 1 * time.Second,
		WatchDir:          tempDir + "/watch",
		ProcessedDir:      tempDir + "/processed",
		ErrorDir:          tempDir + "/error",
		PollInterval:      100 * time.Millisecond,
	}

	orch, err := orchestrator.New(envRoot)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	nullHandler := slog.NewTextHandler(io.Discard, nil)
	nullLogger := slog.New(nullHandler)
	mgr := server.NewManager(cfg, orch, nullLogger)

	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	// 1. Start Manager
	if err := mgr.Start(runCtx); err != nil {
		t.Fatalf("failed to start manager: %v", err)
	}

	// Brief delay to allow the HTTP listener to start up
	time.Sleep(100 * time.Millisecond)

	// 2. Check if the RPC WebSocket port is reachable.
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		t.Fatalf("RPC Node is not listening on %s: %v", addr, err)
	}
	conn.Close()

	// 3. Graceful Shutdown
	cancelRun()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelShutdown()

	if err := mgr.Stop(shutdownCtx); err != nil {
		t.Fatalf("failed to stop manager gracefully: %v", err)
	}
}
