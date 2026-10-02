// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"codeberg.org/tiny-frameworks/nexutils/logger"
)

type Config struct {
	EnvDir string `json:"env_dir"`

	// P2P / RPC
	RPCAddr           string        `json:"rpc_addr"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`

	// Directory Watcher
	WatchDir     string        `json:"watch_dir"`
	ProcessedDir string        `json:"processed_dir"`
	ErrorDir     string        `json:"error_dir"`
	PollInterval time.Duration `json:"poll_interval"`

	// Logging
	Logger *logger.LoggerConfig `json:"log"`
}

// Load performs the 3-stage loading process:
// Stage 1: Defaults -> Stage 2: Config file (if present) -> Stage 3: ENV overrides
func Load(configFilePath string) Config {

	// Stage 1: Hard Defaults in Code
	cfg := Config{
		EnvDir:            "./env",
		RPCAddr:           ":9090",
		HeartbeatInterval: 30 * time.Second,
		WatchDir:          "./data/input",
		ProcessedDir:      "./data/processed",
		ErrorDir:          "./data/error",
		PollInterval:      2 * time.Second,
		Logger: &logger.LoggerConfig{
			Level: logger.LevelInfo,
		},
	}

	// Stage 2: reasd Config-File (if present)
	if configFilePath == "" {
		if _, err := os.Stat("config.json"); err == nil {
			configFilePath = "config.json"
		}
	}

	if configFilePath != "" {
		if fileBytes, err := os.ReadFile(configFilePath); err == nil {
			if err := json.Unmarshal(fileBytes, &cfg); err != nil {
				log.Printf("Failed to parse config file %s, falling back to defaults. err: %s", configFilePath, err.Error())
			} else {
				log.Printf("Loaded configuration from file %s", configFilePath)
			}
		} else {
			log.Printf("Config file %s specified but not readable Error: %v", configFilePath, err)
		}
	}

	// Stage 3:ENV variables override existing values ​​(highest priority)
	applyEnvOverrides(&cfg)

	return cfg
}

func applyEnvOverrides(cfg *Config) {
	if val := os.Getenv("NEXFACT_ENV_DIR"); val != "" {
		cfg.EnvDir = val
	}
	if val := os.Getenv("NEXFACT_RPC_ADDR"); val != "" {
		cfg.RPCAddr = val
	}
	if val := os.Getenv("NEXFACT_HEARTBEAT_INT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.HeartbeatInterval = d
		}
	}
	if val := os.Getenv("NEXFACT_WATCH_DIR"); val != "" {
		cfg.WatchDir = val
	}
	if val := os.Getenv("NEXFACT_PROCESSED_DIR"); val != "" {
		cfg.ProcessedDir = val
	}
	if val := os.Getenv("NEXFACT_ERROR_DIR"); val != "" {
		cfg.ErrorDir = val
	}
	if val := os.Getenv("NEXFACT_POLL_INT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.PollInterval = d
		}
	}

	// Logger configuration
	if val := os.Getenv("NEXFACT_LOG_LEVEL"); val != "" {
		if lvl, err := logger.ParseLevel(val); err != nil {
			cfg.Logger.Level = logger.LevelInfo
			log.Printf("Could not parse %s as Level", val)
		} else {
			cfg.Logger.Level = lvl
		}
	}
	if val := os.Getenv("NEXFACT_LOG_FILE"); val != "" {
		cfg.Logger.Filename = val
	}
	if val := os.Getenv("NEXFACT_LOG_EXPIRY"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.Logger.Expiry = d
		}
	}
	if val := os.Getenv("NEXFACT_LOG_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.Logger.Timeout = d
		}
	}
	if val := os.Getenv("NEXFACT_LOG_USELOCK"); val != "" {
		if d, err := strconv.ParseBool(val); err == nil {
			cfg.Logger.UseLocking = d
		}
	}
}
