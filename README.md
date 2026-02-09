# ArCsent

ArCsent is a **local-only, privacy-first** security monitoring daemon written in Go. It runs on your machine, performs scheduled health and integrity checks, detects anomalies, and exposes a local API + web UI.

## Key Principles

- Local-only by default (binds to `127.0.0.1`).
- No telemetry or external calls unless explicitly enabled.
- Least privilege (drops root after startup).
- Secure-by-default config validation.

## Architecture Diagram

```
┌─────────────────────────────────────────────┐
│               ArCsent Daemon                │
├──────────────┬───────────────┬──────────────┤
│ Scheduler    │ Plugins       │ Detection    │
│ (jobs)       │ (scanners)    │ (baseline)   │
├──────────────┴───────────────┴──────────────┤
│ Alerting     │ Storage       │ API + Web UI │
└──────────────┴───────────────┴──────────────┘
```

## Features

- Scheduler with overlap protection and timeouts.
- Cron schedule support, retry/backoff, and persistent job state.
- System plugins: disk usage, file integrity, process monitoring, auth log monitor, network listeners.
- Baseline and anomaly detection.
- Rule engine, drift detection, and correlation across scanners.
- Alerting (log channel).
- Alerting with dedup + retries, channels (log, webhook, syslog, email).
- Local API with token auth.
- Local web UI with token auth.
- BadgerDB local storage.
- Optional storage encryption, retention pruning, and export endpoints.

## Documentation

- `ARCHITECTURE.md`
- `RUNBOOK.md`
- `INCIDENT_RESPONSE.md`
- `DISASTER_RECOVERY.md`
- `API.md`
- `SECURITY.md`
- `CONTRIBUTING.md`
- `CHANGELOG.md`

## Quick Start (Local)

```bash
go build ./...
./arcsent -config configs/config.json
```

Enable API + Web UI in `configs/config.json`:

```json
"api": {
  "enabled": true,
  "bind_addr": "127.0.0.1:8788",
  "read_only": true,
  "auth_token": "change-me"
},
"web_ui": {
  "enabled": true,
  "bind_addr": "127.0.0.1:8787",
  "read_only": true,
  "auth_token": "change-me"
}
```

Visit `http://127.0.0.1:8787` and enter your token.

Verify endpoints:
- API health: `curl -H "Authorization: <token>" http://127.0.0.1:8788/health`
- Metrics: `curl -H "Authorization: <token>" http://127.0.0.1:8788/metrics`

## CLI (Ops)

Use the built-in CLI against the local API:

```bash
ARCSENT_TOKEN=your-token ./arcsent ctl status
ARCSENT_TOKEN=your-token ./arcsent ctl scanners
ARCSENT_TOKEN=your-token ./arcsent ctl trigger system.disk_usage
ARCSENT_TOKEN=your-token ./arcsent ctl signatures status
ARCSENT_TOKEN=your-token ./arcsent ctl signatures update
ARCSENT_TOKEN=your-token ./arcsent ctl export results -format csv
ARCSENT_TOKEN=your-token ./arcsent ctl metrics
```

Pretty-print JSON output:

```bash
ARCSENT_TOKEN=your-token ./arcsent ctl status -pretty
```

Bash completion (optional):

```bash
source scripts/arcsent_ctl_completion.bash
```

Healthcheck (CLI):

```bash
ARCSENT_TOKEN=your-token scripts/healthcheck.sh
```

Backup/restore (local):

```bash
sudo ARCSENT_DATA_DIR=/var/lib/arcsent ARCSENT_CONFIG=/etc/arcsent/config.json scripts/backup.sh
sudo scripts/restore.sh /var/lib/arcsent/backups/arcsent-backup-<timestamp>.tar.gz
```

## Configuration

Config is JSON and validated at startup.

**Highlights**
- `signatures.enabled` defaults to `false` (opt-in only).
- `signatures.update_interval` uses Go duration strings (e.g. `24h`).
- `signatures.source_urls` lets you override or add per-source URLs (see below).
- `web_ui.enabled` defaults to `false`; when enabled, `web_ui.auth_token` is required.
- `api.enabled` defaults to `false`; when enabled, `api.auth_token` is required.
- Config reload is supported via `SIGHUP` (see Runbook).
- `daemon.user` and `daemon.group` may be numeric IDs or names when running as root.
- `daemon.drop_privileges` defaults to `false` (run as root). Set `true` to drop to `daemon.user`/`daemon.group`.
- Scheduler accepts `@every <duration>`, raw duration, or 5-field cron expressions.
- Retry/backoff: `max_retries`, `retry_backoff`, `retry_max`.
- Detection supports rules, drift detection, and correlation windows.
- Alerting supports dedup window and retries, plus optional channels.
- Storage path expects a BadgerDB directory (default: `/var/lib/arcsent/badger`).
- Storage retention: `storage.retention_days` (prunes old results/baselines).
- Storage encryption: `storage.encryption_key_base64` (32-byte base64 key).

**Signatures & Feeds**

Arcsent can periodically download public TTP and vulnerability feeds into the local cache. This is **opt-in only** and runs on the configured interval (default: daily).

Built-in sources:
- `mitre_attack` (TTPs)
- `nvd`
- `osv`
- `cisa_kev`
- `exploit_db`
- `mitre_capec` (optional)
- `mitre_cwe` (optional)
- `epss` (optional)
- `ghsa` (optional)

Notes:
- Some optional sources require you to provide `signatures.source_urls` with a mirror URL.
- If `signatures.airgap_import_path` is set, Arcsent will **import from that path** and skip network downloads.

**Scanners**

Enable scanners in `scanners`:

```json
{
  "name": "disk-usage",
  "plugin": "system.disk_usage",
  "enabled": true,
  "schedule": "10m",
  "timeout": "10s",
  "allow_overlap": false,
  "run_on_start": true,
  "config": {
    "path": "/",
    "warn_percent": 85,
    "crit_percent": 95
  }
}
```

Additional plugins you can enable:

- `system.auth_log` (parses recent auth log lines for failed logins)
- `system.network_listeners` (counts listening TCP/UDP sockets)
- `system.cpu_memory` (CPU, memory, and swap utilization snapshot)
- `system.load_avg` (load averages and runnable threads)
- `system.uptime` (uptime and idle seconds)

**Detection Rules**

Define rules under `detection.rules` to trigger findings from metrics:

```json
{
  "name": "disk-high",
  "scanner": "system.disk_usage",
  "metric": "used_pct",
  "operator": "gte",
  "threshold": 90,
  "severity": "high",
  "description": "Disk usage exceeded 90%"
}
```

## Example Full Config

```json
{
  "daemon": {
    "log_level": "info",
    "log_format": "json",
    "user": "",
    "group": "",
    "shutdown_timeout": "10s"
  },
  "storage": {
    "db_path": "/var/lib/arcsent/badger",
    "retention_days": 30,
    "encryption_key_base64": ""
  },
  "signatures": {
    "enabled": false,
    "update_interval": "24h",
    "sources": ["mitre_attack", "nvd", "osv", "cisa_kev", "exploit_db"],
    "cache_dir": "/var/lib/arcsent/signatures",
    "airgap_import_path": "",
    "source_urls": {
      "osv": "https://example.com/mirrors/osv.zip"
    }
  },
  "api": {
    "enabled": true,
    "bind_addr": "127.0.0.1:8788",
    "read_only": true,
    "auth_token": "change-me"
  },
  "web_ui": {
    "enabled": true,
    "bind_addr": "127.0.0.1:8787",
    "read_only": true,
    "auth_token": "change-me"
  },
  "scanners": [
    {
      "name": "disk-usage",
      "plugin": "system.disk_usage",
      "enabled": true,
      "schedule": "10m",
      "timeout": "10s",
      "allow_overlap": false,
      "run_on_start": true,
      "config": {
        "path": "/",
        "warn_percent": 85,
        "crit_percent": 95
      }
    }
  ]
}
```

## API (Local-only)

All endpoints require the token.

- `GET /health`
- `GET /status`
- `GET /scanners`
- `POST /scanners/trigger/{plugin}`
- `GET /results/latest`
- `GET /results/history`
- `GET /findings`
- `GET /baselines`
- `GET /export/results` (JSON or CSV via `?format=csv`)
- `GET /export/baselines` (JSON or CSV via `?format=csv`)
- `GET /signatures/status`
- `POST /signatures/update`
- `GET /metrics` (Prometheus text format)

Same endpoints are available under `/api/*`.

## Web UI

The UI is embedded and served locally. It calls the local API via `/api/*` and requires the same token.
The landing page is public; API calls remain token-protected.

## Tooling

- Lint: `make lint`
- Tests: `make test`
- Vulnerability scan: `make vuln`
- SBOM (requires `syft`): `make sbom`

## Deployment (Local)

See `deploy/README.md` for systemd and Docker (local-only) instructions.

## Development

```bash
go test ./...
golangci-lint run ./...
```

## Troubleshooting

- API/UI not reachable: confirm `bind_addr` and token.
- No scan results: ensure scanners are enabled and scheduled.
- Storage errors: verify `/var/lib/arcsent/badger` permissions.

## Smoke Test

Run a basic local smoke test:

```bash
ARCSENT_TOKEN=your-token scripts/smoke_test.sh
```

## Roadmap

- Cron schedule support.
- Additional plugins (network, auth log).
- Encrypted storage at rest.

## Security Posture (Local-only)

- Bind API/UI to `127.0.0.1` only.
- Use strong tokens for API/UI.
- Do not enable external alerting unless explicitly required.
- Run as an unprivileged user whenever possible.
