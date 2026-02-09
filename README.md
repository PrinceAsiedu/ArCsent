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
- System plugins: disk usage, file integrity, process monitoring, auth log monitor, network listeners.
- Baseline and anomaly detection.
- Alerting (log channel).
- Local API with token auth.
- Local web UI with token auth.
- BadgerDB local storage.

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

## Configuration

Config is JSON and validated at startup.

**Highlights**
- `signatures.enabled` defaults to `false` (opt-in only).
- `signatures.update_interval` uses Go duration strings (e.g. `24h`).
- `web_ui.enabled` defaults to `false`; when enabled, `web_ui.auth_token` is required.
- `api.enabled` defaults to `false`; when enabled, `api.auth_token` is required.
- `daemon.user` and `daemon.group` may be numeric IDs or names when running as root.
- `daemon.drop_privileges` defaults to `false` (run as root). Set `true` to drop to `daemon.user`/`daemon.group`.
- Scheduler accepts `@every <duration>` or raw duration strings (cron support later).
- Storage path expects a BadgerDB directory (default: `/var/lib/arcsent/badger`).

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
    "db_path": "/var/lib/arcsent/badger"
  },
  "signatures": {
    "enabled": false,
    "update_interval": "24h",
    "sources": ["mitre_attack", "nvd", "osv", "cisa_kev", "exploit_db"],
    "cache_dir": "/var/lib/arcsent/signatures",
    "airgap_import_path": ""
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
