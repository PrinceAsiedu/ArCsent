# ArCsent

Local, privacy-first security monitoring daemon in Go. Phase 1 scaffolding includes
secure config loading, opt-in vulnerability/TTP signature updates, and supply chain
tooling hooks.

## Quick Start

```bash
go build ./...
./arcsent -config configs/config.json
```

## Tooling

- Lint: `make lint`
- Tests: `make test`
- Vulnerability scan: `make vuln`
- SBOM (requires `syft`): `make sbom`

## Config Highlights

- `signatures.enabled` defaults to `false` (opt-in only).
- `signatures.update_interval` uses Go duration strings (e.g. `24h`).
- `web_ui.enabled` defaults to `false`; when enabled, `web_ui.auth_token` is required.
- `daemon.user` and `daemon.group` may be numeric IDs or names when running as root.
- Scheduler currently accepts `@every <duration>` or raw duration strings (cron support later).
