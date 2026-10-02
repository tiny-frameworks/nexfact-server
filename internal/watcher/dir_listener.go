// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"codeberg.org/tiny-frameworks/nexfact-server/internal/config"
	"codeberg.org/tiny-frameworks/nexfact/orchestrator"
	"codeberg.org/tiny-frameworks/nexutils/errors"
)

type DirectoryWatcher struct {
	cfg  config.Config
	orch *orchestrator.Orchestrator
	log  *slog.Logger
	wg   sync.WaitGroup // Trackt aktive Batch-Processes
}

func New(cfg config.Config, orch *orchestrator.Orchestrator, log *slog.Logger) *DirectoryWatcher {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 2 * time.Second
	}
	return &DirectoryWatcher{cfg: cfg, orch: orch, log: log}
}

// Start starts Ticker-Loop in own Goroutine
func (w *DirectoryWatcher) Start(ctx context.Context) error {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.loop(ctx)
	}()
	return nil
}

// Stop waits until the running goroutine, including any pending jobs, has finished.
func (w *DirectoryWatcher) Stop() {
	// The context has already been cancelled by main.go / the manager.
	// Here, we are simply waiting for the loop to exit cleanly.
	w.wg.Wait()
	w.log.Info("DirectoryWatcher stopped cleanly")
}

func (w *DirectoryWatcher) loop(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	// IMMEDIATE initial scan upon startup
	w.processFiles(ctx)

	for {
		select {
		case <-ctx.Done():
			// Context cancelled -> immediately terminate ticker loop
			w.log.Info("DirectoryWatcher stopping signal received...")
			return

		case <-ticker.C:
			// Process directory contents
			w.processFiles(ctx)
		}
	}
}

func (w *DirectoryWatcher) processFiles(ctx context.Context) {
	// Check if the context was cancelled before the directory scan
	select {
	case <-ctx.Done():
		return
	default:
	}

	files, err := os.ReadDir(w.cfg.WatchDir)
	if err != nil {
		nerr := errors.Wrap(
			errors.ReadError,
			fmt.Sprintf("Failed to read watch directory %s", w.cfg.WatchDir),
			"internal.watcher.dir_listener",
			err,
		)
		errors.LogError(w.log, nerr)
		return
	}

	for _, entry := range files {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		// Check for a shutdown signal at each iteration step.
		select {
		case <-ctx.Done():
			return
		default:
			// Generate full path
			fullPath := filepath.Join(w.cfg.WatchDir, entry.Name())
			w.processSingleFile(fullPath)
		}
	}
}

func (w *DirectoryWatcher) processSingleFile(fileToProcess string) {

	jsonBytes, err := os.ReadFile(fileToProcess)
	if err != nil {
		nerr := errors.Wrap(
			errors.ReadError,
			fmt.Sprintf("Failed to read file, path: %s", fileToProcess),
			"internal.watcher.dir_listener",
			err,
		)
		errors.LogError(w.log, nerr)
		return
	}

	fileName := filepath.Base(fileToProcess)

	// Handover to the modular nexfact Engine Core
	out, err := w.orch.RunJsonJob(jsonBytes)
	if err != nil {
		errors.LogError(w.log, err)
		targetPath := filepath.Join(w.cfg.ErrorDir, fileName)
		_ = os.Rename(fileToProcess, targetPath)
		return
	}

	w.log.Info("File processed successfully", "job_id", out.JobID, "out_path", out.OutputPath)
	targetPath := filepath.Join(w.cfg.ProcessedDir, fileName)
	err = os.Rename(fileToProcess, targetPath)
}
