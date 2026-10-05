// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"codeberg.org/tiny-frameworks/nexfact-server/internal/config"
	"codeberg.org/tiny-frameworks/nexutils/logger"
)

func TestConfig_Defaults(t *testing.T) {
	// Ensures that os.Args is reset in isolation for the test.
	resetArgs(t, "nexfact-server")
	// get absolute Path prefix
	absPath, _ := filepath.Abs("./")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error loading config defaults: %v", err)
	}

	fromEnv := "./env"
	tmpPath := filepath.Join(absPath, fromEnv)
	if cfg.EnvDir != tmpPath {
		t.Errorf("expected default EnvDir %q, got %q", fromEnv, cfg.EnvDir)
	}
	if cfg.RPCAddr != "127.0.0.1:8080" {
		t.Errorf("expected default RPCAddr %q, got %q", "127.0.0.1:8080", cfg.RPCAddr)
	}
	if cfg.PollInterval != 2*time.Second {
		t.Errorf("expected default PollInterval 2s, got %v", cfg.PollInterval)
	}
	if cfg.Logger == nil || cfg.Logger.Level != logger.LevelInfo {
		t.Errorf("expected default LogLevel Info, got %v", cfg.Logger)
	}
}

func TestConfig_YAMLFile(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "custom_config.yaml")

	yamlContent := `
env_dir: "/var/lib/nexfact/env"
rpc_addr: "0.0.0.0:9090"
poll_interval: 500ms
heartbeat_interval: 10s
log:
  level: debug
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test yaml file: %v", err)
	}

	resetArgs(t, "nexfact-server", "-cfgFile", cfgPath)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error loading config from yaml: %v", err)
	}

	if cfg.EnvDir != "/var/lib/nexfact/env" {
		t.Errorf("expected EnvDir from YAML %q, got %q", "/var/lib/nexfact/env", cfg.EnvDir)
	}
	if cfg.RPCAddr != "0.0.0.0:9090" {
		t.Errorf("expected RPCAddr from YAML %q, got %q", "0.0.0.0:9090", cfg.RPCAddr)
	}
	if cfg.PollInterval != 500*time.Millisecond {
		t.Errorf("expected PollInterval 500ms, got %v", cfg.PollInterval)
	}
	if cfg.Logger.Level != logger.LevelDebug {
		t.Errorf("expected LogLevel Debug, got %v", cfg.Logger.Level)
	}
}

func TestConfig_EnvOverrides(t *testing.T) {
	resetArgs(t, "nexfact-server")

	// t.Setenv ensures that the environment variables are reset after the test.
	t.Setenv("NEXFACT_ENV_DIR", "/env/from/env")
	t.Setenv("NEXFACT_RPC_ADDR", "10.0.0.1:8080")
	t.Setenv("NEXFACT_POLL_INT", "1s")
	t.Setenv("NEXFACT_LOG_LEVEL", "warn")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error loading config with ENV overrides: %v", err)
	}

	if cfg.EnvDir != "/env/from/env" {
		t.Errorf("expected EnvDir from ENV %q, got %q", "/env/from/env", cfg.EnvDir)
	}
	if cfg.RPCAddr != "10.0.0.1:8080" {
		t.Errorf("expected RPCAddr from ENV %q, got %q", "10.0.0.1:8080", cfg.RPCAddr)
	}
	if cfg.PollInterval != 1*time.Second {
		t.Errorf("expected PollInterval 1s, got %v", cfg.PollInterval)
	}
	if cfg.Logger.Level != logger.LevelWarn {
		t.Errorf("expected LogLevel Warn, got %v", cfg.Logger.Level)
	}
}

func TestConfig_FullPrecedence(t *testing.T) {
	// get absolute Path prefix
	absPath, _ := filepath.Abs("./")

	// 1. create YAML Datei
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.yaml")

	yamlContent := `
env_dir: "from_yaml"
rpc_addr: "from_yaml:8080"
watch_dir: "from_yaml_watch"
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test yaml file: %v", err)
	}

	// 2. Set ENV variables (Rank 2 overrides Rank 3)
	t.Setenv("NEXFACT_ENV_DIR", "from_env")
	t.Setenv("NEXFACT_RPC_ADDR", "from_env:8080")

	// 3. Passing CLI flags (Rank 1 overrides Ranks 2 and 3)
	resetArgs(t, "nexfact-server",
		"-cfgFile", cfgPath,
		"-env", "from_cli",
		"-log-level", "error",
	)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error during full precedence test: %v", err)
	}

	// CLI overwrites ENV & YAML
	fromCli := "from_cli"
	tmpPath := filepath.Join(absPath, fromCli)
	if cfg.EnvDir != tmpPath {
		t.Errorf("expected EnvDir %q (from CLI), got %q", fromCli, cfg.EnvDir)
	}

	// ENV overwrites YAML (because no CLI-Flag for rpc-addr was set)
	if cfg.RPCAddr != "from_env:8080" {
		t.Errorf("expected RPCAddr %q (from ENV), got %q", "from_env:8080", cfg.RPCAddr)
	}

	// YAML is used (because neither ENV nor CLI were set)
	fromYaml := "from_yaml_watch"
	tmpPath = filepath.Join(absPath, fromYaml)
	if cfg.WatchDir != tmpPath {
		t.Errorf("expected WatchDir %q (from YAML), got %q", fromYaml, cfg.WatchDir)
	}

	// CLI overwrites LogLevel
	if cfg.Logger.Level != logger.LevelError {
		t.Errorf("expected LogLevel Error, got %v", cfg.Logger.Level)
	}
}

// resetArgs resets os.Args for the duration of the test.
func resetArgs(t *testing.T, args ...string) {
	t.Helper()
	oldArgs := os.Args
	os.Args = args

	t.Cleanup(func() {
		os.Args = oldArgs
	})
}
