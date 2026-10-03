// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
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
	// os.Exit is called ONLY HERE so that defers in run() execute safely!
	os.Exit(run())
}

func run() int {

	// 1. load Config from ENV / File / Defaults
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		return 1
	}

	// 2. setup Logger
	logger.SetupLogging(cfg.Logger)
	lgr := logger.Logger.With("component", "nexfact-service", "role", "server")
	lgr.Info("=== Start nexfact-service ===")

	lgr.Info("Configuration loaded",
		"env_dir", cfg.EnvDir,
		"rpc_addr", cfg.RPCAddr,
		"watch_dir", cfg.WatchDir,
		"log_level", cfg.Logger.Level,
	)

	// 3. create Staging-/Watch-Directory
	for _, dir := range []string{cfg.WatchDir, cfg.ProcessedDir, cfg.ErrorDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			lgr.Error("failed to create directory", "dir", dir, "err", err)
			return 1
		}
	}

	// 4. Runtime-Context for DirectoryWatcher & Background Tasks
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	orch, err := orchestrator.New(cfg.EnvDir)
	if err != nil {
		errors.LogError(lgr, err)
		return errors.GetExitCode(err)
	}

	// 5.Manager receives config, orchestrator, and logger (node ​​is created here)
	srvManager := server.NewManager(cfg, orch, lgr)

	// Start services with the runtime context (Watcher uses runCtx, node starts in the background)
	if err := srvManager.Start(runCtx); err != nil {
		errors.LogError(lgr, err)
		return errors.GetExitCode(err)
	}

	// 6. Wait for OS Signals (SIGINT, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	lgr.Info("Shutdown signal received, stopping services...")

	// Terminate runtime context (stops the loop in the DirectoryWatcher)
	cancelRun()

	// Dedicated timeout context for orderly shutdown
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srvManager.Stop(shutdownCtx); err != nil {
		errors.LogError(lgr, err)
		return errors.GetExitCode(err)
	}

	lgr.Info("=== nexfact-service stopped cleanly ===")
	return 0
}
