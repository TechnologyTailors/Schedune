#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:9090}"

echo "=== Planning a VM Launch (Placement -> Validation -> DryRun) ==="

if command -v python3 >/dev/null 2>&1; then
    curl -s -X POST -H "Content-Type: application/json" -d @examples/plans/vm-plan.json "${API_BASE}/api/v1alpha1/plan/launch" | python3 -m json.tool || true
else
    curl -s -X POST -H "Content-Type: application/json" -d @examples/plans/vm-plan.json "${API_BASE}/api/v1alpha1/plan/launch"
    echo ""
fi
