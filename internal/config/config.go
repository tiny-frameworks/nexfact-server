// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"codeberg.org/tiny-frameworks/nexutils/logger"
	yaml "github.com/goccy/go-yaml"
)

type Config struct {
	ConfigFile string `yaml:"config_file"`
	EnvDir     string `yaml:"env_dir"`
	RPCAddr    string `yaml:"rpc_addr"`

	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	PollInterval      time.Duration `yaml:"poll_interval"`

	WatchDir     string `yaml:"watch_dir"`
	ProcessedDir string `yaml:"processed_dir"`
	ErrorDir     string `yaml:"error_dir"`

	Logger *logger.LoggerConfig `yaml:"log"`
}

// Load performs the 4-stage loading process:
// Stage 1: Defaults -> Stage 2: Config file (if present) -> Stage 3: ENV overrides -> Stage 4: CLI overrides

// DefaultConfig hard-coded default-values (Stage 4 - lowest priority).
func defaultConfig() Config {
	return Config{
		ConfigFile:        "config.yaml",
		EnvDir:            "./env",
		RPCAddr:           "127.0.0.1:8080",
		WatchDir:          "./watch",
		ProcessedDir:      "./processed",
		ErrorDir:          "./error",
		PollInterval:      2 * time.Second,
		HeartbeatInterval: 5 * time.Second,
		Logger: &logger.LoggerConfig{
			Level: logger.LevelInfo,
		},
	}
}

func Load() (*Config, error) {

	// 1: initialize defaults
	cfg := defaultConfig()

	// 2. initialize cli-Flags
	fs := flag.NewFlagSet("nexfact-server", flag.ContinueOnError)

	cfgFileFlag := fs.String("cfgFile", "", "Path to YAML configuration file")
	envFlag := fs.String("env", "", "Path to nexfact environment directory")
	rpcAddrFlag := fs.String("rpc-addr", "", "Server address for WebSocket/P2P RPC")
	watchDirFlag := fs.String("watch-dir", "", "Directory to monitor for incoming invoices")
	processedDirFlag := fs.String("processed-dir", "", "Directory for successfully processed invoices")
	errorDirFlag := fs.String("error-dir", "", "Directory for failed invoices")
	pollIntervalFlag := fs.Duration("poll-interval", 0, "Polling interval for directory watcher (e.g. 2s, 500ms)")
	heartbeatFlag := fs.Duration("heartbeat", 0, "Heartbeat interval for P2P nodes (e.g. 5s)")
	logLevelFlag := fs.String("log-level", "", "Logging level (debug, info, warn, error)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	// 3. Determine path to config file (flag > default)
	if *cfgFileFlag != "" {
		cfg.ConfigFile = *cfgFileFlag
	}

	// Stage 2:Load YAML configuration file (if present)
	if data, err := os.ReadFile(cfg.ConfigFile); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal yaml config from %s: %w", cfg.ConfigFile, err)
		}
	} else if *cfgFileFlag != "" {
		// If a config file was explicitly requested via a flag, it must exist.
		return nil, fmt.Errorf("specified config file not found: %w", err)
	}

	// Nil pointer protection for logger config
	ensureLoggerConfig(&cfg)

	// Stage 3: Apply environment variables (NEXFACT_)
	applyEnvOverrides(&cfg)

	// Stage 4: Evaluate specific CLI flags (override only explicitly set flags)
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "env":
			cfg.EnvDir = *envFlag
		case "rpc-addr":
			cfg.RPCAddr = *rpcAddrFlag
		case "watch-dir":
			cfg.WatchDir = *watchDirFlag
		case "processed-dir":
			cfg.ProcessedDir = *processedDirFlag
		case "error-dir":
			cfg.ErrorDir = *errorDirFlag
		case "poll-interval":
			cfg.PollInterval = *pollIntervalFlag
		case "heartbeat":
			cfg.HeartbeatInterval = *heartbeatFlag
		case "log-level":
			if lvl, err := logger.ParseLevel(*logLevelFlag); err != nil {
				cfg.Logger.Level = logger.LevelInfo
				fmt.Fprintf(os.Stderr, "config: invalid log-level %q, falling back to info\n", *logLevelFlag)
			} else {
				cfg.Logger.Level = lvl
			}
		}
	})

	return &cfg, nil
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
			fmt.Fprintf(os.Stderr, "Could not parse %s as Level", val)
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

func ensureLoggerConfig(cfg *Config) {
	if cfg.Logger == nil {
		cfg.Logger = &logger.LoggerConfig{
			Level: logger.LevelInfo,
		}
	}
}
