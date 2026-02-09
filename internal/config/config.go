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
	API        APIConfig        `json:"api"`
	WebUI      WebUIConfig      `json:"web_ui"`
	Scanners   []ScannerConfig  `json:"scanners"`
}

type DaemonConfig struct {
	LogLevel        string `json:"log_level"`
	LogFormat       string `json:"log_format"`
	User            string `json:"user"`
	Group           string `json:"group"`
	ShutdownTimeout string `json:"shutdown_timeout"`
	DropPrivileges  bool   `json:"drop_privileges"`
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

type APIConfig struct {
	Enabled   bool   `json:"enabled"`
	BindAddr  string `json:"bind_addr"`
	ReadOnly  bool   `json:"read_only"`
	AuthToken string `json:"auth_token"`
}

type ScannerConfig struct {
	Name         string                 `json:"name"`
	Plugin       string                 `json:"plugin"`
	Enabled      bool                   `json:"enabled"`
	Schedule     string                 `json:"schedule"`
	Timeout      string                 `json:"timeout"`
	MaxRetries   int                    `json:"max_retries"`
	RetryBackoff string                 `json:"retry_backoff"`
	RetryMax     string                 `json:"retry_max"`
	AllowOverlap bool                   `json:"allow_overlap"`
	RunOnStart   bool                   `json:"run_on_start"`
	Config       map[string]interface{} `json:"config"`
}

func Default() Config {
	return Config{
		Daemon: DaemonConfig{
			LogLevel:        "info",
			LogFormat:       "json",
			User:            "",
			Group:           "",
			ShutdownTimeout: "10s",
			DropPrivileges:  false,
		},
		Storage: StorageConfig{
			DBPath: "/var/lib/arcsent/badger",
		},
		Signatures: SignaturesConfig{
			Enabled:        false,
			UpdateInterval: "24h",
			Sources:        signatures.DefaultSources(),
			CacheDir:       "/var/lib/arcsent/signatures",
		},
		API: APIConfig{
			Enabled:  false,
			BindAddr: "127.0.0.1:8788",
			ReadOnly: true,
		},
		WebUI: WebUIConfig{
			Enabled:  false,
			BindAddr: "127.0.0.1:8787",
			ReadOnly: true,
		},
		Scanners: []ScannerConfig{},
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
	if c.Daemon.DropPrivileges {
		if os.Geteuid() == 0 {
			if c.Daemon.User == "" || c.Daemon.Group == "" {
				errs = append(errs, "daemon.user and daemon.group are required when drop_privileges is true")
			}
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

	if c.API.Enabled {
		if c.API.BindAddr == "" {
			errs = append(errs, "api.bind_addr is required when enabled")
		}
		if c.API.AuthToken == "" {
			errs = append(errs, "api.auth_token is required when enabled")
		}
	}

	for i, sc := range c.Scanners {
		if sc.Name == "" {
			errs = append(errs, fmt.Sprintf("scanners[%d].name is required", i))
		}
		if sc.Plugin == "" {
			errs = append(errs, fmt.Sprintf("scanners[%d].plugin is required", i))
		}
		if sc.Enabled {
			if sc.Schedule == "" {
				errs = append(errs, fmt.Sprintf("scanners[%d].schedule is required when enabled", i))
			}
		}
		if sc.Timeout != "" {
			if _, err := time.ParseDuration(sc.Timeout); err != nil {
				errs = append(errs, fmt.Sprintf("scanners[%d].timeout must be a valid duration", i))
			}
		}
		if sc.RetryBackoff != "" {
			if _, err := time.ParseDuration(sc.RetryBackoff); err != nil {
				errs = append(errs, fmt.Sprintf("scanners[%d].retry_backoff must be a valid duration", i))
			}
		}
		if sc.RetryMax != "" {
			if _, err := time.ParseDuration(sc.RetryMax); err != nil {
				errs = append(errs, fmt.Sprintf("scanners[%d].retry_max must be a valid duration", i))
			}
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
	if clone.API.AuthToken != "" {
		clone.API.AuthToken = "REDACTED"
	}
	return clone
}

func (s ScannerConfig) TimeoutDuration() time.Duration {
	if s.Timeout == "" {
		return 0
	}
	parsed, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return 0
	}
	return parsed
}

func (s ScannerConfig) RetryBackoffDuration() time.Duration {
	if s.RetryBackoff == "" {
		return 0
	}
	parsed, err := time.ParseDuration(s.RetryBackoff)
	if err != nil {
		return 0
	}
	return parsed
}

func (s ScannerConfig) RetryMaxDuration() time.Duration {
	if s.RetryMax == "" {
		return 0
	}
	parsed, err := time.ParseDuration(s.RetryMax)
	if err != nil {
		return 0
	}
	return parsed
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
	if v, ok := os.LookupEnv("ARCSENT_API_ENABLED"); ok {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.API.Enabled = parsed
		}
	}
	if v, ok := os.LookupEnv("ARCSENT_API_TOKEN"); ok && v != "" {
		cfg.API.AuthToken = v
	}
}
