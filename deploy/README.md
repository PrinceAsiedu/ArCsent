# Deployment

This project is intended for **local deployment** only.

## systemd

1. Create user/group:

```bash
sudo useradd --system --home /var/lib/arcsent --shell /usr/sbin/nologin arcsent
sudo mkdir -p /etc/arcsent /var/lib/arcsent
sudo chown -R arcsent:arcsent /etc/arcsent /var/lib/arcsent
```

2. Install binary and config:

```bash
sudo install -m 0755 /path/to/arcsent /usr/local/bin/arcsent
sudo install -m 0640 configs/config.json /etc/arcsent/config.json
```

3. Install unit:

```bash
sudo install -m 0644 deploy/systemd/arcsent.service /etc/systemd/system/arcsent.service
sudo systemctl daemon-reload
sudo systemctl enable --now arcsent
```

### Quick install

```bash
sudo ARCSENT_BIN=./arcsent ARCSENT_CONFIG=/etc/arcsent/config.json scripts/install_systemd.sh
```

### Healthcheck

```bash
ARCSENT_TOKEN=your-token scripts/healthcheck.sh
```

### Log rotation

Optional logrotate config is available at `deploy/logrotate/arcsent`.

### Backup/Restore

```bash
sudo ARCSENT_DATA_DIR=/var/lib/arcsent ARCSENT_CONFIG=/etc/arcsent/config.json scripts/backup.sh
sudo scripts/restore.sh /var/lib/arcsent/backups/arcsent-backup-<timestamp>.tar.gz
```

### Watchdog (optional)

```bash
ARCSENT_TOKEN=your-token scripts/watchdog.sh
```

### AppArmor (optional)

AppArmor profile template: `deploy/apparmor/arcsent.apparmor`.

## Docker (local-only)

```bash
docker build -f deploy/Dockerfile -t arcsent:local .
docker run --rm -p 127.0.0.1:8787:8787 -p 127.0.0.1:8788:8788 arcsent:local
```
