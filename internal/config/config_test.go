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

	// 1. Stufe 2 vorbereiten: Config File anlegen
	jsonContent := `{
		"rpc_addr": ":8080",
		"watch_dir": "/custom/watch"
	}`
	_ = os.WriteFile(configFile, []byte(jsonContent), 0644)

	// 2. Stufe 3 vorbereiten: ENV Variable setzen (soll RPCAddr überschreiben)
	t.Setenv("NEXFACT_RPC_ADDR", ":7070")

	// Config laden
	cfg := config.Load([]string{configFile})

	// Assertions:
	// - RPCAddr muss aus ENV (:7070) stammen (Stufe 3 schlägt Stufe 2)
	if cfg.RPCAddr != ":7070" {
		t.Errorf("expected RPCAddr from ENV ':7070', got '%s'", cfg.RPCAddr)
	}

	// - WatchDir muss aus Config-File (/custom/watch) stammen (Stufe 2)
	if cfg.WatchDir != "/custom/watch" {
		t.Errorf("expected WatchDir from config.json '/custom/watch', got '%s'", cfg.WatchDir)
	}

	// - ProcessedDir muss auf Default ("./data/processed") stehen (Stufe 1)
	if cfg.ProcessedDir != "./data/processed" {
		t.Errorf("expected ProcessedDir default './data/processed', got '%s'", cfg.ProcessedDir)
	}
}
