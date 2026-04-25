#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:9090}"
PAYLOAD_FILE="/tmp/payload.json"
SPEC_FILE="examples/launch-specs/cloudhypervisor-validate.json"

if [ ! -f "$PAYLOAD_FILE" ]; then
    echo "Error: /tmp/payload.json is missing. Please run 'make example-intake' first to ingest a node."
    exit 1
fi

NODE_ID=$(grep -o '"hostname": *"[^"]*"' "$PAYLOAD_FILE" | cut -d'"' -f4 | head -n 1)
if [ -z "$NODE_ID" ]; then
    NODE_ID="unknown"
fi

TMP_SPEC=$(mktemp)
trap 'rm -f "$TMP_SPEC"' EXIT

# Safely replace PUT_NODE_ID_HERE with the dynamic NODE_ID using a small Python snippet
if command -v python3 >/dev/null 2>&1; then
    python3 -c "import json, sys; d = json.load(sys.stdin); d['node_id'] = '${NODE_ID}'; json.dump(d, sys.stdout)" < "$SPEC_FILE" > "$TMP_SPEC"
    curl -s -X POST -H "Content-Type: application/json" -d @"$TMP_SPEC" "${API_BASE}/api/v1alpha1/launch/validate" | python3 -m json.tool || true
else
    sed "s/PUT_NODE_ID_HERE/${NODE_ID}/g" "$SPEC_FILE" > "$TMP_SPEC"
    curl -s -X POST -H "Content-Type: application/json" -d @"$TMP_SPEC" "${API_BASE}/api/v1alpha1/launch/validate"
    echo ""
fi