#!/usr/bin/env bash
set -euo pipefail

DATA_DIR=${ARCSENT_DATA_DIR:-/var/lib/arcsent}
CONFIG=${ARCSENT_CONFIG:-/etc/arcsent/config.json}
OUT=${ARCSENT_BACKUP_OUT:-/var/lib/arcsent/backups}

if [[ $EUID -ne 0 ]]; then
  echo "backup.sh must be run as root" >&2
  exit 1
fi

timestamp=$(date -u +"%Y%m%dT%H%M%SZ")
archive="$OUT/arcsent-backup-$timestamp.tar.gz"
manifest="$OUT/arcsent-backup-$timestamp.sha256"

mkdir -p "$OUT"

tar -czf "$archive" \
  --warning=no-file-changed \
  -C / \
  "${DATA_DIR#/}" \
  "${CONFIG#/}"

sha256sum "$archive" > "$manifest"

echo "Backup created: $archive"
echo "Checksum manifest: $manifest"
