#!/usr/bin/env bash
set -euo pipefail

API_ADDR="${API_ADDR:-127.0.0.1:8788}"
WEB_ADDR="${WEB_ADDR:-127.0.0.1:8787}"
TOKEN="${ARCSENT_TOKEN:-${1:-}}"

if [[ -z "${TOKEN}" ]]; then
  echo "Usage: ARCSENT_TOKEN=... scripts/smoke_test.sh"
  echo "   or: scripts/smoke_test.sh <token>"
  exit 1
fi

auth_header=(-H "Authorization: ${TOKEN}")

echo "== Health =="
curl -s "${auth_header[@]}" "http://${API_ADDR}/health" | sed 's/.*/OK: &/'

echo "== Scanners =="
curl -s "${auth_header[@]}" "http://${API_ADDR}/scanners" | sed 's/.*/OK: &/'

echo "== Trigger disk usage =="
curl -s -X POST "${auth_header[@]}" "http://${API_ADDR}/scanners/trigger/system.disk_usage" | sed 's/.*/OK: &/'

echo "== Findings =="
curl -s "${auth_header[@]}" "http://${API_ADDR}/findings" | sed 's/.*/OK: &/'

echo "== Baselines =="
curl -s "${auth_header[@]}" "http://${API_ADDR}/baselines" | sed 's/.*/OK: &/'

echo "== Web UI (public) =="
curl -s "http://${WEB_ADDR}/" >/dev/null && echo "OK: web ui reachable"

echo "Smoke test complete."
