// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/tiny-frameworks/nexfact-server/internal/config"
	"codeberg.org/tiny-frameworks/nexfact-server/internal/server"
	"codeberg.org/tiny-frameworks/nexfact/orchestrator"

	"codeberg.org/tiny-frameworks/nexutils/errors"
	"codeberg.org/tiny-frameworks/nexutils/logger"
)

func main() {
	// os.Exit wird NUR HIER aufgerufen, damit defers in run() sicher ausführen!
	os.Exit(run())
}

func run() int {
	// 1. Config aus ENV / File / Defaults laden
	cfg := config.Load(os.Args[1:])

	// 2. Logger aufsetzen
	logger.SetupLogging(cfg.Logger)
	lgr := logger.Logger.With("component", "nexfact-service", "role", "server")
	lgr.Info("=== Start nexfact-service ===")

	lgr.Info("Configuration loaded",
		"env_dir", cfg.EnvDir,
		"rpc_addr", cfg.RPCAddr,
		"watch_dir", cfg.WatchDir,
		"log_level", cfg.Logger.Level,
	)

	// 3. Staging-/Watch-Verzeichnisse automatisch erstellen
	for _, dir := range []string{cfg.WatchDir, cfg.ProcessedDir, cfg.ErrorDir} {
		_ = os.MkdirAll(dir, 0755)
	}

	// 4. Laufzeit-Context für den DirectoryWatcher & Background Tasks
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	orch, err := orchestrator.New(cfg.EnvDir)
	if err != nil {
		errors.LogError(lgr, err)
		return errors.GetExitCode(err)
	}

	// 5. Manager erhält Config, Orchestrator und Logger (Node wird hier erzeugt)
	srvManager := server.NewManager(cfg, orch, lgr)

	// Dienste mit dem Laufzeit-Context starten (Watcher nutzt runCtx, Node startet im Hintergrund)
	if err := srvManager.Start(runCtx); err != nil {
		errors.LogError(lgr, err)
		return errors.GetExitCode(err)
	}

	// 6. Warten auf OS Signals (SIGINT, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	lgr.Info("Shutdown signal received, stopping services...")

	// Laufzeit-Context beenden (stoppt die Schleife im DirectoryWatcher)
	cancelRun()

	// Dedizierter Timeout-Context für das geordnete Beenden
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srvManager.Stop(shutdownCtx); err != nil {
		errors.LogError(lgr, err)
		return errors.GetExitCode(err)
	}

	lgr.Info("=== nexfact-service stopped cleanly ===")
	return 0
}
