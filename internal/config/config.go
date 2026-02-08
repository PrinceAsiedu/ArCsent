package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ipsix/arcsent/internal/signatures"
)

const (
	DefaultConfigPath = "configs/config.json"
)

type Config struct {
	Daemon     DaemonConfig     `json:"daemon"`
	Storage    StorageConfig    `json:"storage"`
	Signatures SignaturesConfig `json:"signatures"`
	WebUI      WebUIConfig      `json:"web_ui"`
}

type DaemonConfig struct {
	LogLevel        string `json:"log_level"`
	LogFormat       string `json:"log_format"`
	User            string `json:"user"`
	Group           string `json:"group"`
	ShutdownTimeout string `json:"shutdown_timeout"`
}

type StorageConfig struct {
	DBPath string `json:"db_path"`
}

type SignaturesConfig struct {
	Enabled          bool     `json:"enabled"`
	UpdateInterval   string   `json:"update_interval"`
	Sources          []string `json:"sources"`
	CacheDir         string   `json:"cache_dir"`
	AirgapImportPath string   `json:"airgap_import_path"`
}

type WebUIConfig struct {
	Enabled   bool   `json:"enabled"`
	BindAddr  string `json:"bind_addr"`
	ReadOnly  bool   `json:"read_only"`
	AuthToken string `json:"auth_token"`
}

func Default() Config {
	return Config{
		Daemon: DaemonConfig{
			LogLevel:        "info",
			LogFormat:       "json",
			User:            "",
			Group:           "",
			ShutdownTimeout: "10s",
		},
		Storage: StorageConfig{
			DBPath: "/var/lib/arcsent/arcsent.db",
		},
		Signatures: SignaturesConfig{
			Enabled:        false,
			UpdateInterval: "24h",
			Sources:        signatures.DefaultSources(),
			CacheDir:       "/var/lib/arcsent/signatures",
		},
		WebUI: WebUIConfig{
			Enabled:  false,
			BindAddr: "127.0.0.1:8787",
			ReadOnly: true,
		},
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	cfg := Default()

	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	applyEnvOverrides(&cfg)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var errs []string

	switch strings.ToLower(c.Daemon.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, "daemon.log_level must be one of: debug, info, warn, error")
	}

	switch strings.ToLower(c.Daemon.LogFormat) {
	case "json", "text":
	default:
		errs = append(errs, "daemon.log_format must be one of: json, text")
	}

	if c.Daemon.ShutdownTimeout != "" {
		if _, err := time.ParseDuration(c.Daemon.ShutdownTimeout); err != nil {
			errs = append(errs, "daemon.shutdown_timeout must be a valid duration (e.g. 10s)")
		}
	}

	if c.Storage.DBPath == "" {
		errs = append(errs, "storage.db_path is required")
	} else if !filepath.IsAbs(c.Storage.DBPath) {
		errs = append(errs, "storage.db_path must be an absolute path")
	}

	if c.Signatures.Enabled {
		if c.Signatures.UpdateInterval == "" {
			errs = append(errs, "signatures.update_interval is required when enabled")
		} else if _, err := time.ParseDuration(c.Signatures.UpdateInterval); err != nil {
			errs = append(errs, "signatures.update_interval must be a valid duration (e.g. 24h)")
		}

		if len(c.Signatures.Sources) == 0 {
			errs = append(errs, "signatures.sources must include at least one source when enabled")
		}

		for _, src := range c.Signatures.Sources {
			if signatures.IsKnownSource(src) {
				continue
			}
			if strings.HasPrefix(src, "custom:") {
				continue
			}
			errs = append(errs, fmt.Sprintf("signatures.sources contains unknown source: %s", src))
		}

		if c.Signatures.CacheDir == "" {
			errs = append(errs, "signatures.cache_dir is required when enabled")
		} else if !filepath.IsAbs(c.Signatures.CacheDir) {
			errs = append(errs, "signatures.cache_dir must be an absolute path")
		}
	}

	if c.Signatures.AirgapImportPath != "" && !filepath.IsAbs(c.Signatures.AirgapImportPath) {
		errs = append(errs, "signatures.airgap_import_path must be an absolute path if set")
	}

	if c.WebUI.Enabled {
		if c.WebUI.BindAddr == "" {
			errs = append(errs, "web_ui.bind_addr is required when enabled")
		}
		if c.WebUI.AuthToken == "" {
			errs = append(errs, "web_ui.auth_token is required when enabled")
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func (d DaemonConfig) ShutdownTimeoutDuration() time.Duration {
	if d.ShutdownTimeout == "" {
		return 10 * time.Second
	}
	parsed, err := time.ParseDuration(d.ShutdownTimeout)
	if err != nil {
		return 10 * time.Second
	}
	return parsed
}

func (c Config) Redacted() Config {
	clone := c
	if clone.WebUI.AuthToken != "" {
		clone.WebUI.AuthToken = "REDACTED"
	}
	return clone
}

func applyEnvOverrides(cfg *Config) {
	if v, ok := os.LookupEnv("ARCSENT_WEB_UI_ENABLED"); ok {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.WebUI.Enabled = parsed
		}
	}
	if v, ok := os.LookupEnv("ARCSENT_WEB_UI_TOKEN"); ok && v != "" {
		cfg.WebUI.AuthToken = v
	}
	if v, ok := os.LookupEnv("ARCSENT_SIGNATURES_ENABLED"); ok {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.Signatures.Enabled = parsed
		}
	}
}
