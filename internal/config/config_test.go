// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"codeberg.org/tiny-frameworks/nexfact-server/internal/config"
)

func TestConfigThreeStagePrecedence(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")

	// 1. Stage 2 prepare: creae Config File
	jsonContent := `{
		"rpc_addr": ":8080",
		"watch_dir": "/custom/watch"
	}`
	_ = os.WriteFile(configFile, []byte(jsonContent), 0644)

	// 2. Stage 3 prepare: set ENV Variables (should overwrite RPCAddr)
	t.Setenv("NEXFACT_RPC_ADDR", ":7070")

	// load Config
	cfg := config.Load(configFile)

	// Assertions:
	// - RPCAddr must come from ENV (:7070) (Level 3 overrides Level 2)
	if cfg.RPCAddr != ":7070" {
		t.Errorf("expected RPCAddr from ENV ':7070', got '%s'", cfg.RPCAddr)
	}

	// - WatchDir must come from the config file (/custom/watch) (Level 2)
	if cfg.WatchDir != "/custom/watch" {
		t.Errorf("expected WatchDir from config.json '/custom/watch', got '%s'", cfg.WatchDir)
	}

	// - ProcessedDir must be set to the default ("./data/processed") (Level 1).
	if cfg.ProcessedDir != "./data/processed" {
		t.Errorf("expected ProcessedDir default './data/processed', got '%s'", cfg.ProcessedDir)
	}
}
