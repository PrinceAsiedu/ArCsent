package config

import (
	"encoding/base64"
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
	Detection  DetectionConfig  `json:"detection"`
	Alerting   AlertingConfig   `json:"alerting"`
	Security   SecurityConfig   `json:"security"`
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
	DBPath              string `json:"db_path"`
	RetentionDays       int    `json:"retention_days"`
	EncryptionKeyBase64 string `json:"encryption_key_base64"`
}

type SignaturesConfig struct {
	Enabled          bool              `json:"enabled"`
	UpdateInterval   string            `json:"update_interval"`
	Sources          []string          `json:"sources"`
	CacheDir         string            `json:"cache_dir"`
	AirgapImportPath string            `json:"airgap_import_path"`
	SourceURLs       map[string]string `json:"source_urls"`
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

type DetectionConfig struct {
	CorrelationWindow      string       `json:"correlation_window"`
	CorrelationMinScanners int          `json:"correlation_min_scanners"`
	CorrelationCooldown    string       `json:"correlation_cooldown"`
	DriftConsecutive       int          `json:"drift_consecutive"`
	Rules                  []RuleConfig `json:"rules"`
}

type RuleConfig struct {
	Name        string  `json:"name"`
	Scanner     string  `json:"scanner"`
	Metric      string  `json:"metric"`
	Operator    string  `json:"operator"`
	Threshold   float64 `json:"threshold"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
}

type AlertingConfig struct {
	Enabled      bool                 `json:"enabled"`
	DedupWindow  string               `json:"dedup_window"`
	RetryMax     int                  `json:"retry_max"`
	RetryBackoff string               `json:"retry_backoff"`
	Channels     []AlertChannelConfig `json:"channels"`
}

type AlertChannelConfig struct {
	Type     string   `json:"type"`
	Enabled  bool     `json:"enabled"`
	Severity []string `json:"severity"`

	URL string `json:"url"`

	SyslogNetwork string `json:"syslog_network"`
	SyslogAddress string `json:"syslog_address"`
	SyslogTag     string `json:"syslog_tag"`

	SMTPServer string   `json:"smtp_server"`
	SMTPUser   string   `json:"smtp_user"`
	SMTPPass   string   `json:"smtp_pass"`
	From       string   `json:"from"`
	To         []string `json:"to"`
	Subject    string   `json:"subject"`
}

type SecurityConfig struct {
	SelfIntegrity  bool   `json:"self_integrity"`
	ExpectedSHA256 string `json:"expected_sha256"`
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
			DBPath:              "/var/lib/arcsent/badger",
			RetentionDays:       30,
			EncryptionKeyBase64: "",
		},
		Signatures: SignaturesConfig{
			Enabled:        false,
			UpdateInterval: "24h",
			Sources:        signatures.DefaultSources(),
			CacheDir:       "/var/lib/arcsent/signatures",
			SourceURLs:     map[string]string{},
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
		Detection: DetectionConfig{
			CorrelationWindow:      "5m",
			CorrelationMinScanners: 2,
			CorrelationCooldown:    "5m",
			DriftConsecutive:       3,
			Rules:                  []RuleConfig{},
		},
		Alerting: AlertingConfig{
			Enabled:      false,
			DedupWindow:  "5m",
			RetryMax:     3,
			RetryBackoff: "2s",
			Channels: []AlertChannelConfig{
				{Type: "log", Enabled: true},
			},
		},
		Security: SecurityConfig{
			SelfIntegrity:  false,
			ExpectedSHA256: "",
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
	if c.Storage.RetentionDays < 0 {
		errs = append(errs, "storage.retention_days must be >= 0")
	}
	if c.Storage.EncryptionKeyBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(c.Storage.EncryptionKeyBase64)
		if err != nil {
			errs = append(errs, "storage.encryption_key_base64 must be valid base64")
		} else if len(decoded) != 32 {
			errs = append(errs, "storage.encryption_key_base64 must decode to 32 bytes")
		}
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
		for src, url := range c.Signatures.SourceURLs {
			if !signatures.IsKnownSource(src) && !strings.HasPrefix(src, "custom:") {
				errs = append(errs, fmt.Sprintf("signatures.source_urls contains unknown source: %s", src))
			}
			if strings.TrimSpace(url) == "" {
				errs = append(errs, fmt.Sprintf("signatures.source_urls contains empty URL for %s", src))
			}
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

	if c.Detection.CorrelationWindow != "" {
		if _, err := time.ParseDuration(c.Detection.CorrelationWindow); err != nil {
			errs = append(errs, "detection.correlation_window must be a valid duration")
		}
	}
	if c.Detection.CorrelationCooldown != "" {
		if _, err := time.ParseDuration(c.Detection.CorrelationCooldown); err != nil {
			errs = append(errs, "detection.correlation_cooldown must be a valid duration")
		}
	}
	if c.Detection.CorrelationMinScanners < 1 {
		errs = append(errs, "detection.correlation_min_scanners must be >= 1")
	}
	if c.Detection.DriftConsecutive < 1 {
		errs = append(errs, "detection.drift_consecutive must be >= 1")
	}
	for i, rule := range c.Detection.Rules {
		if rule.Name == "" {
			errs = append(errs, fmt.Sprintf("detection.rules[%d].name is required", i))
		}
		if rule.Scanner == "" {
			errs = append(errs, fmt.Sprintf("detection.rules[%d].scanner is required", i))
		}
		if rule.Metric == "" {
			errs = append(errs, fmt.Sprintf("detection.rules[%d].metric is required", i))
		}
		switch strings.ToLower(rule.Operator) {
		case "gt", "gte", "lt", "lte", "eq":
		default:
			errs = append(errs, fmt.Sprintf("detection.rules[%d].operator must be one of gt,gte,lt,lte,eq", i))
		}
	}

	if c.Alerting.DedupWindow != "" {
		if _, err := time.ParseDuration(c.Alerting.DedupWindow); err != nil {
			errs = append(errs, "alerting.dedup_window must be a valid duration")
		}
	}
	if c.Alerting.RetryBackoff != "" {
		if _, err := time.ParseDuration(c.Alerting.RetryBackoff); err != nil {
			errs = append(errs, "alerting.retry_backoff must be a valid duration")
		}
	}
	if c.Alerting.RetryMax < 0 {
		errs = append(errs, "alerting.retry_max must be >= 0")
	}
	for i, ch := range c.Alerting.Channels {
		if ch.Type == "" {
			errs = append(errs, fmt.Sprintf("alerting.channels[%d].type is required", i))
		}
	}
	if c.Security.SelfIntegrity && c.Security.ExpectedSHA256 == "" {
		errs = append(errs, "security.expected_sha256 is required when self_integrity is enabled")
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
	if clone.Signatures.SourceURLs != nil {
		redacted := map[string]string{}
		for key := range clone.Signatures.SourceURLs {
			redacted[key] = "REDACTED"
		}
		clone.Signatures.SourceURLs = redacted
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

func (d DetectionConfig) CorrelationWindowDuration() time.Duration {
	if d.CorrelationWindow == "" {
		return 0
	}
	parsed, err := time.ParseDuration(d.CorrelationWindow)
	if err != nil {
		return 0
	}
	return parsed
}

func (d DetectionConfig) CorrelationCooldownDuration() time.Duration {
	if d.CorrelationCooldown == "" {
		return 0
	}
	parsed, err := time.ParseDuration(d.CorrelationCooldown)
	if err != nil {
		return 0
	}
	return parsed
}

func (s SignaturesConfig) UpdateIntervalDuration() time.Duration {
	if s.UpdateInterval == "" {
		return 0
	}
	parsed, err := time.ParseDuration(s.UpdateInterval)
	if err != nil {
		return 0
	}
	return parsed
}

func (s SignaturesConfig) SourceURLOverrides() map[string]string {
	out := map[string]string{}
	for key, value := range s.SourceURLs {
		out[key] = value
	}
	return out
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
