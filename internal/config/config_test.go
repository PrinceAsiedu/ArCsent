package config

import "testing"

func TestValidateDefaults(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected defaults to validate, got: %v", err)
	}
}

func TestValidateWebUIRequiresToken(t *testing.T) {
	cfg := Default()
	cfg.WebUI.Enabled = true
	cfg.WebUI.AuthToken = ""
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for missing web_ui.auth_token")
	}
}

func TestRedacted(t *testing.T) {
	cfg := Default()
	cfg.WebUI.AuthToken = "secret"
	redacted := cfg.Redacted()
	if redacted.WebUI.AuthToken == "secret" {
		t.Fatalf("expected auth token to be redacted")
	}
}

func TestSignaturesRequireSourcesWhenEnabled(t *testing.T) {
	cfg := Default()
	cfg.Signatures.Enabled = true
	cfg.Signatures.Sources = nil
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error when signatures enabled without sources")
	}
}

func TestShutdownTimeoutDuration(t *testing.T) {
	cfg := Default()
	cfg.Daemon.ShutdownTimeout = "2s"
	if got := cfg.Daemon.ShutdownTimeoutDuration(); got.String() != "2s" {
		t.Fatalf("expected 2s, got %s", got)
	}
	cfg.Daemon.ShutdownTimeout = "invalid"
	if got := cfg.Daemon.ShutdownTimeoutDuration(); got <= 0 {
		t.Fatalf("expected fallback duration, got %s", got)
	}
}

func TestValidateAPIRequiresToken(t *testing.T) {
	cfg := Default()
	cfg.API.Enabled = true
	cfg.API.AuthToken = ""
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for missing api.auth_token")
	}
}

func TestValidateScannerSchedule(t *testing.T) {
	cfg := Default()
	cfg.Scanners = []ScannerConfig{
		{
			Name:    "test",
			Plugin:  "system.disk_usage",
			Enabled: true,
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for missing scanner schedule")
	}
}
