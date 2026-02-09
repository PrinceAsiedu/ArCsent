# Runbook

This runbook is for local deployments only.

**Start**
1. Set tokens in `configs/config.json` for API and Web UI.
2. Enable scanners you want to run.
3. Start the daemon: `./arcsent -config configs/config.json`.

**Verify**
1. API health: `curl -H "Authorization: <token>" http://127.0.0.1:8788/health`
2. Web UI: open `http://127.0.0.1:8787/` and enter token.

**Trigger a Scan**
1. `curl -X POST -H "Authorization: <token>" http://127.0.0.1:8788/scanners/trigger/system.disk_usage`

**Rotate Tokens**
1. Update `api.auth_token` and `web_ui.auth_token` in config.
2. Restart the daemon.

**Common Checks**
1. Verify daemon is running and listening on localhost only.
2. Check `results/history` and `findings` endpoints.
3. Validate storage path permissions (`/var/lib/arcsent/badger` by default).
