#!/usr/bin/env bash
set -euo pipefail

BIN=${ARCSENT_BIN:-./arcsent}
CONFIG=${ARCSENT_CONFIG:-/etc/arcsent/config.json}
SERVICE_SRC=${ARCSENT_SERVICE_SRC:-deploy/systemd/arcsent.service}
SERVICE_DST=${ARCSENT_SERVICE_DST:-/etc/systemd/system/arcsent.service}
DATA_DIR=${ARCSENT_DATA_DIR:-/var/lib/arcsent}
TOKEN=${ARCSENT_TOKEN:-}

if [[ $EUID -ne 0 ]]; then
  echo "install_local.sh must be run as root" >&2
  exit 1
fi

if [[ -z "$TOKEN" ]]; then
  echo "ARCSENT_TOKEN is required" >&2
  exit 1
fi

if [[ ! -f "$BIN" ]]; then
  echo "Binary not found at $BIN" >&2
  exit 1
fi

id -u arcsent >/dev/null 2>&1 || useradd --system --home "$DATA_DIR" --shell /usr/sbin/nologin arcsent
mkdir -p "$(dirname "$CONFIG")" "$DATA_DIR"
chown -R arcsent:arcsent "$DATA_DIR"

install -m 0755 "$BIN" /usr/local/bin/arcsent
install -m 0644 "$SERVICE_SRC" "$SERVICE_DST"

cat >"$CONFIG" <<EOF
{
  "daemon": {
    "log_level": "info",
    "log_format": "json",
    "user": "arcsent",
    "group": "arcsent",
    "shutdown_timeout": "10s",
    "drop_privileges": true
  },
  "storage": {
    "db_path": "$DATA_DIR/badger",
    "retention_days": 30,
    "encryption_key_base64": ""
  },
  "signatures": {
    "enabled": false,
    "update_interval": "24h",
    "sources": [
      "mitre_attack",
      "nvd",
      "osv",
      "cisa_kev",
      "exploit_db"
    ],
    "cache_dir": "$DATA_DIR/signatures",
    "airgap_import_path": "",
    "source_urls": {}
  },
  "api": {
    "enabled": true,
    "bind_addr": "127.0.0.1:8788",
    "read_only": true,
    "auth_token": "$TOKEN"
  },
  "web_ui": {
    "enabled": true,
    "bind_addr": "127.0.0.1:8787",
    "read_only": true,
    "auth_token": "$TOKEN"
  },
  "scanners": [],
  "detection": {
    "correlation_window": "5m",
    "correlation_min_scanners": 2,
    "correlation_cooldown": "5m",
    "drift_consecutive": 3,
    "rules": []
  },
  "alerting": {
    "enabled": false,
    "dedup_window": "5m",
    "retry_max": 3,
    "retry_backoff": "2s",
    "channels": [
      { "type": "log", "enabled": true, "severity": ["info", "low", "medium", "high", "critical"] }
    ]
  },
  "security": {
    "self_integrity": false,
    "expected_sha256": ""
  }
}
EOF

chown arcsent:arcsent "$CONFIG"

systemctl daemon-reload
systemctl enable --now arcsent

echo "Installed and started. UI: http://127.0.0.1:8787"
