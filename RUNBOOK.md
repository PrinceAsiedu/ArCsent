# Runbook

This runbook is for local deployments only.

**Start**
1. Set tokens in `configs/config.json` for API and Web UI.
2. Enable scanners you want to run.
3. Start the daemon: `./arcsent -config configs/config.json`.

**Verify**
1. API health: `curl -H "Authorization: <token>" http://127.0.0.1:8788/health`
2. Web UI: open `http://127.0.0.1:8787/` and enter token.
3. CLI status: `ARCSENT_TOKEN=<token> ./arcsent ctl status`
4. CLI metrics: `ARCSENT_TOKEN=<token> ./arcsent ctl metrics`
5. Full healthcheck: `ARCSENT_TOKEN=<token> scripts/healthcheck.sh`

**Trigger a Scan**
1. `curl -X POST -H "Authorization: <token>" http://127.0.0.1:8788/scanners/trigger/system.disk_usage`
2. `ARCSENT_TOKEN=<token> ./arcsent ctl trigger system.disk_usage`

**Scheduling**
- Use duration schedules (`10m`) or cron expressions (`*/5 * * * *`).
- Configure retries with `max_retries`, `retry_backoff`, `retry_max`.

**Detection**
- Rules can be defined under `detection.rules`.
- Drift detection triggers after `drift_consecutive` anomalies.
- Correlation triggers when multiple scanners report findings within `correlation_window`.

**Alerting**
- Enable alerting in `alerting.enabled`.
- Channels: `log`, `webhook`, `syslog`, `email`.
- Use `dedup_window` and retry settings to control delivery.

**Rotate Tokens**
1. Update `api.auth_token` and `web_ui.auth_token` in config.
2. Restart the daemon.

**Reload Config (SIGHUP)**
1. Update `configs/config.json`.
2. Send `SIGHUP` to the daemon (for example: `pkill -HUP arcsent`).
3. Review logs for any “requires restart” warnings (bind address or storage changes).
4. Optional: `./arcsent -reload -pid <pid>` (or set `ARCSENT_PID`).

**Common Checks**
1. Verify daemon is running and listening on localhost only.
2. Check `results/history` and `findings` endpoints.
3. Validate storage path permissions (`/var/lib/arcsent/badger` by default).

**Export**
- JSON: `curl -H "Authorization: <token>" http://127.0.0.1:8788/export/results`
- CSV: `curl -H "Authorization: <token>" http://127.0.0.1:8788/export/results?format=csv`
 - CLI: `ARCSENT_TOKEN=<token> ./arcsent ctl export results -format csv`

**CLI Pretty Print**
- `ARCSENT_TOKEN=<token> ./arcsent ctl status -pretty`

**CLI Completion**
- `source scripts/arcsent_ctl_completion.bash`

**CLI Smoke Test**
- `ARCSENT_TOKEN=<token> scripts/ctl_smoke.sh`

**Backup**
- `sudo ARCSENT_DATA_DIR=/var/lib/arcsent ARCSENT_CONFIG=/etc/arcsent/config.json scripts/backup.sh`

**Restore**
- `sudo scripts/restore.sh /var/lib/arcsent/backups/arcsent-backup-<timestamp>.tar.gz`

**Watchdog**
- `ARCSENT_TOKEN=<token> scripts/watchdog.sh`

**Metrics**
- `curl -H "Authorization: <token>" http://127.0.0.1:8788/metrics`
