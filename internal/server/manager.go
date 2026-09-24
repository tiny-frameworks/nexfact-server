// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"codeberg.org/tiny-frameworks/nexfact-server/internal/config"
	"codeberg.org/tiny-frameworks/nexfact-server/internal/p2p"
	"codeberg.org/tiny-frameworks/nexfact-server/internal/watcher"
	"codeberg.org/tiny-frameworks/nexfact/orchestrator"
	"codeberg.org/tiny-frameworks/nexutils/p2p/rpc"
)

type Manager struct {
	cfg         config.Config
	log         *slog.Logger
	orch        *orchestrator.Orchestrator
	rpcDelegate *rpc.Node
	watcher     *watcher.DirectoryWatcher
}

func NewManager(cfg config.Config, orch *orchestrator.Orchestrator, lgr *slog.Logger) *Manager {

	// Erzeugt den RPC Node (erhält intern sein eigenes Context)
	delegate := p2p.NewDelegate(orch, cfg.RPCAddr, cfg.HeartbeatInterval, lgr)

	// Erzeugt den Directory Watcher
	dirWatcher := watcher.New(config.Config{
		WatchDir:     cfg.WatchDir,
		ProcessedDir: cfg.ProcessedDir,
		ErrorDir:     cfg.ErrorDir,
		PollInterval: cfg.PollInterval,
	}, orch, lgr)

	return &Manager{
		cfg:         cfg,
		log:         lgr,
		orch:        orch,
		rpcDelegate: delegate,
		watcher:     dirWatcher,
	}
}

func (m *Manager) Start(ctx context.Context) error {
	m.log.Info("Starting nexfact-server services...")

	// 1. Directory Watcher starten (nutzt runCtx für Ticker-Loop)
	if err := m.watcher.Start(ctx); err != nil {
		return err
	}

	// 2. RPC Node im Hintergrund starten (da ListenAndServe blockiert)
	go func() {
		if err := m.rpcDelegate.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			m.log.Error("RPC Node HTTP server error", "err", err)
		}
	}()

	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	m.log.Info("Stopping nexfact-server services...")

	// 1. Directory Watcher stoppen
	m.watcher.Stop()

	// 2. RPC Node geordnet herunterfahren (schließt Peers & HTTP-Server)
	if err := m.rpcDelegate.Stop(); err != nil {
		m.log.Error("Error stopping RPC Node", "err", err)
		return err
	}

	return nil
}
