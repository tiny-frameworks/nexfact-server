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

	// Instantiate the real authenticator here (e.g., from DB/config)
	// (Or leave as nil for the DummyAuthenticator)
	var userAuth rpc.UserAuthenticator // = myUserStore

	// Creates the RPC node, including the UserAuthenticator
	delegate := p2p.NewDelegate(orch, cfg.RPCAddr, cfg.HeartbeatInterval, userAuth, lgr)

	// Creates Directory Watcher
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

	// 1. Start directory watcher (uses runCtx for ticker loop)
	if err := m.watcher.Start(ctx); err != nil {
		return err
	}

	// 2. Start the RPC node in the background (since ListenAndServe blocks)
	go func() {
		if err := m.rpcDelegate.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			m.log.Error("RPC Node HTTP server error", "err", err)
		}
	}()

	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	m.log.Info("Stopping nexfact-server services...")

	// 1. Stop Directory Watcher
	m.watcher.Stop()

	// 2. Shut down RPC node gracefully (closes peers & HTTP server)
	if err := m.rpcDelegate.Stop(); err != nil {
		m.log.Error("Error stopping RPC Node", "err", err)
		return err
	}

	return nil
}
