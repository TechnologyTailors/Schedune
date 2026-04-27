#!/bin/bash
set -e

echo "=== Planning a VM Launch (Placement -> Validation -> DryRun) ==="
curl -s -X POST http://127.0.0.1:9090/api/v1alpha1/plan/launch \
     -H "Content-Type: application/json" \
     -d @examples/plans/vm-plan.json \
     | jq .
