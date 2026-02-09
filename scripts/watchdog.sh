#!/usr/bin/env bash
set -euo pipefail

ADDR=${ARCSENT_ADDR:-http://127.0.0.1:8788}
TOKEN=${ARCSENT_TOKEN:-}
SERVICE=${ARCSENT_SERVICE:-arcsent}
RETRIES=${ARCSENT_WATCHDOG_RETRIES:-3}
SLEEP=${ARCSENT_WATCHDOG_SLEEP:-5}

if [[ -z "$TOKEN" ]]; then
  echo "ARCSENT_TOKEN is required" >&2
  exit 1
fi

check_health() {
  curl -fsS -H "Authorization: $TOKEN" "$ADDR/health" >/dev/null 2>&1
}

fails=0
for ((i=0; i<RETRIES; i++)); do
  if check_health; then
    echo "healthy"
    exit 0
  fi
  fails=$((fails+1))
  sleep "$SLEEP"
done

echo "healthcheck failed ($fails). restarting $SERVICE"
systemctl restart "$SERVICE"
