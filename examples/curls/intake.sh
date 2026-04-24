#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:9090}"
REFRESH="${REFRESH:-0}"

if [ ! -f "/tmp/payload.json" ] || [ "$REFRESH" -eq 1 ]; then
    echo "Generating /tmp/payload.json using agent..."
    ./bin/schedune-agent inspect > /tmp/payload.json
fi

if command -v python3 >/dev/null 2>&1; then
    curl -s -X POST -H "Content-Type: application/json" -d @/tmp/payload.json "${API_BASE}/api/v1alpha1/intake/envelope" | python3 -m json.tool || true
else
    curl -s -X POST -H "Content-Type: application/json" -d @/tmp/payload.json "${API_BASE}/api/v1alpha1/intake/envelope"
    echo ""
fi