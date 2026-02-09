#!/usr/bin/env bash
set -euo pipefail

BIN=${ARCSENT_BIN:-./arcsent}
CONFIG=${ARCSENT_CONFIG:-/etc/arcsent/config.json}
SERVICE_SRC=${ARCSENT_SERVICE_SRC:-deploy/systemd/arcsent.service}
SERVICE_DST=${ARCSENT_SERVICE_DST:-/etc/systemd/system/arcsent.service}
DATA_DIR=${ARCSENT_DATA_DIR:-/var/lib/arcsent}
ENV_FILE=${ARCSENT_ENV_FILE:-/etc/arcsent/arcsent.env}
TOKEN=${ARCSENT_TOKEN:-}

if [[ $EUID -ne 0 ]]; then
  echo "install_systemd.sh must be run as root" >&2
  exit 1
fi

if [[ ! -f "$BIN" ]]; then
  echo "Binary not found at $BIN" >&2
  exit 1
fi

if [[ ! -f "$SERVICE_SRC" ]]; then
  echo "Service file not found at $SERVICE_SRC" >&2
  exit 1
fi

id -u arcsent >/dev/null 2>&1 || useradd --system --home "$DATA_DIR" --shell /usr/sbin/nologin arcsent
mkdir -p "$(dirname "$CONFIG")" "$DATA_DIR"
mkdir -p "$(dirname "$ENV_FILE")"
chown -R arcsent:arcsent "$DATA_DIR"

install -m 0755 "$BIN" /usr/local/bin/arcsent
install -m 0755 scripts/watchdog.sh /usr/local/bin/arcsent-watchdog
install -m 0644 "$SERVICE_SRC" "$SERVICE_DST"

if [[ -n "$TOKEN" ]]; then
  cat >"$ENV_FILE" <<EOF
ARCSENT_API_TOKEN=$TOKEN
ARCSENT_WEB_UI_TOKEN=$TOKEN
EOF
  chmod 0600 "$ENV_FILE"
fi

if [[ ! -f "$CONFIG" ]]; then
  echo "Config not found at $CONFIG. Copying default."
  install -m 0644 configs/config.json "$CONFIG"
  chown arcsent:arcsent "$CONFIG"
fi

systemctl daemon-reload
systemctl enable arcsent

echo "Installed. Start with: systemctl start arcsent"
