#!/usr/bin/env bash
set -euo pipefail

ARCHIVE=${1:-}
MANIFEST=${2:-}
DATA_DIR=${ARCSENT_DATA_DIR:-/var/lib/arcsent}
CONFIG=${ARCSENT_CONFIG:-/etc/arcsent/config.json}

if [[ -z "$ARCHIVE" ]]; then
  echo "Usage: scripts/restore.sh <backup.tar.gz> [checksum.sha256]" >&2
  exit 1
fi

if [[ $EUID -ne 0 ]]; then
  echo "restore.sh must be run as root" >&2
  exit 1
fi

systemctl stop arcsent || true

mkdir -p "$DATA_DIR" "$(dirname "$CONFIG")"

if [[ -n "$MANIFEST" ]]; then
  (cd "$(dirname "$MANIFEST")" && sha256sum -c "$(basename "$MANIFEST")")
fi

tar -xzf "$ARCHIVE" -C /

chown -R arcsent:arcsent "$DATA_DIR" || true
chown arcsent:arcsent "$CONFIG" || true

systemctl start arcsent || true

echo "Restore complete."
