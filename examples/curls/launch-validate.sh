#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:9090}"

if command -v python3 >/dev/null 2>&1; then
    curl -s -X POST -H "Content-Type: application/json" -d @examples/launch-specs/cloudhypervisor-validate.json "${API_BASE}/api/v1alpha1/launch/validate" | python3 -m json.tool || true
else
    curl -s -X POST -H "Content-Type: application/json" -d @examples/launch-specs/cloudhypervisor-validate.json "${API_BASE}/api/v1alpha1/launch/validate"
    echo ""
fi