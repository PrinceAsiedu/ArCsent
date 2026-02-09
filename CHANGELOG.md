# Changelog

## Unreleased

- Initial local-only security monitoring daemon scaffold.
- Added signatures updater with opt-in feed downloads, airgap import support, and status tracking.
- Added `/signatures/status` and `/signatures/update` API endpoints.
- Added `signatures.source_urls` for per-source feed overrides.
- Added Prometheus-style `/metrics` endpoint.
- Added SIGHUP config reload with scheduler + signatures + alerting refresh.
- Added signatures status panel in the Web UI.
- Added `-reload` CLI flag to send SIGHUP to a running daemon.
- Added metrics panel in the Web UI.
- Added new system plugins: `system.cpu_memory`, `system.load_avg`, `system.uptime`.
- Added `arcsent ctl` CLI for local ops (status, scanners, trigger, signatures, exports, metrics).
- Added CLI pretty-print option and bash completion script.
- Added CLI smoke test script.
- Added healthcheck script, systemd installer helper, and logrotate config.
- Added backup/restore and watchdog scripts, plus AppArmor profile template.
- Added one-command local installer, backup checksums, and watchdog systemd timer units.
- Added config validation + storage check CLI commands and secrets env file support.
