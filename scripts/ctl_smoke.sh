#!/usr/bin/env bash
set -euo pipefail

BIN=${ARCSENT_BIN:-./arcsent}
ADDR=${ARCSENT_ADDR:-http://127.0.0.1:8788}
TOKEN=${ARCSENT_TOKEN:-}

if [[ -z "$TOKEN" ]]; then
  echo "ARCSENT_TOKEN is required" >&2
  exit 1
fi

echo "== Status =="
$BIN ctl -addr "$ADDR" -token "$TOKEN" status

echo "== Health =="
$BIN ctl -addr "$ADDR" -token "$TOKEN" health

echo "== Scanners =="
$BIN ctl -addr "$ADDR" -token "$TOKEN" scanners

echo "== Metrics =="
$BIN ctl -addr "$ADDR" -token "$TOKEN" metrics

echo "== Signatures Status =="
$BIN ctl -addr "$ADDR" -token "$TOKEN" signatures status

echo "== Trigger disk usage =="
$BIN ctl -addr "$ADDR" -token "$TOKEN" trigger system.disk_usage

echo "Smoke test complete."
