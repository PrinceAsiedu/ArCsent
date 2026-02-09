# Disaster Recovery

**Scope**
Local-only deployment. Focus is on restoring the host and local data.

**Backups**
1. Backup `/etc/arcsent/config.json`.
2. Backup `/var/lib/arcsent/badger` directory.
3. If encryption is enabled, backup the base64 key securely.

**Restore**
1. Restore config and storage directory.
2. Restart the daemon.
3. Verify API and Web UI.

**Validation**
1. Confirm baseline data is present via `/baselines`.
2. Run a manual scan and verify results.
