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

// Load führt das 3-stufige Lade-Verfahren durch:
// Stufe 1: Defaults -> Stufe 2: Config-File (falls vorhanden) -> Stufe 3: ENV-Overrides
// func Load(configFilePath string, log *logger.Logger) Config {
func Load(arguments []string) Config {

	var configFilePath string

	// Stufe 1: Harte Defaults im Code
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

	// Stufe 2: Config-Datei einlesen (falls vorhanden)
	configFilePath = ""
	if len(arguments) > 0 {
		configFilePath = arguments[0]
	}
	if configFilePath == "" {
		if _, err := os.Stat("config.json"); err == nil {
			configFilePath = "config.json"
		}
	}

	if configFilePath != "" {
		if fileBytes, err := os.ReadFile(configFilePath); err == nil {
			if err := json.Unmarshal(fileBytes, &cfg); err != nil {
				log.Printf("Failed to parse config file %s, falling back to defaults. err: %v", configFilePath, err)
			} else {
				log.Printf("Loaded configuration from file %s", configFilePath)
			}
		} else {
			log.Printf("Config file %s specified but not readable Error: %v", configFilePath, err)
		}
	}

	// Stufe 3: ENV-Variablen überschreiben bestehende Werte (Höchste Priorität)
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
