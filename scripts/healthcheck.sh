#!/usr/bin/env bash
set -euo pipefail

ADDR=${ARCSENT_ADDR:-http://127.0.0.1:8788}
TOKEN=${ARCSENT_TOKEN:-}

if [[ -z "$TOKEN" ]]; then
  echo "ARCSENT_TOKEN is required" >&2
  exit 1
fi

echo "== Health =="
curl -fsS -H "Authorization: $TOKEN" "$ADDR/health"
echo ""

echo "== Status =="
curl -fsS -H "Authorization: $TOKEN" "$ADDR/status"
echo ""

echo "== Metrics =="
curl -fsS -H "Authorization: $TOKEN" "$ADDR/metrics"
echo ""

echo "Healthcheck complete."
