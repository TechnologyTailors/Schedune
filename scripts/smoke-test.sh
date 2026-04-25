#!/usr/bin/env bash
set -euo pipefail

# Ensure Python 3 is available for JSON parsing and injection
if ! command -v python3 >/dev/null 2>&1; then
    echo "Error: python3 is required for automated smoke tests (JSON validation)."
    exit 1
fi

API_BASE="${API_BASE:-http://127.0.0.1:9090}"
DB_PATH="$(pwd)/var/schedune.db"
export DB_PATH

echo "============================================"
echo "    Schedune Automated E2E Smoke Test       "
echo "============================================"

# Prepare clean environment
rm -f "$DB_PATH"
TMP_DIR=$(mktemp -d)

# Process management
CP_PID=""
cleanup() {
    echo "Cleaning up..."
    if [ -n "$CP_PID" ] && kill -0 "$CP_PID" 2>/dev/null; then
        kill "$CP_PID"
        wait "$CP_PID" 2>/dev/null || true
    fi
    rm -rf "$TMP_DIR"
    rm -f "$DB_PATH"
}
trap cleanup EXIT

echo "Starting Control Plane in background..."
./bin/schedune server > "$TMP_DIR/server.log" 2>&1 &
CP_PID=$!

echo "Waiting for API to be reachable..."
MAX_RETRIES=30
RETRY_COUNT=0
while ! curl -s "${API_BASE}/api/v1alpha1/recovery/orphans" > /dev/null; do
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        echo "Error: Control plane failed to start or is not reachable."
        cat "$TMP_DIR/server.log"
        exit 1
    fi
    sleep 0.5
    RETRY_COUNT=$((RETRY_COUNT + 1))
done
echo "Control plane is reachable."

echo "--------------------------------------------"
echo "1. Inspecting local node (Rust Agent)..."
PAYLOAD_FILE="$TMP_DIR/payload.json"
./bin/schedune-agent inspect > "$PAYLOAD_FILE"

NODE_ID=$(grep -o '"hostname": *"[^"]*"' "$PAYLOAD_FILE" | cut -d'"' -f4 | head -n 1)
if [ -z "$NODE_ID" ]; then
    echo "Error: Could not extract node_id from agent payload."
    exit 1
fi
echo "Generated payload for node_id: $NODE_ID"

echo "--------------------------------------------"
echo "2. Ingesting Node (POST /api/v1alpha1/intake/envelope)..."
INTAKE_RESP="$TMP_DIR/intake_resp.json"
HTTP_CODE=$(curl -s -w "%{http_code}" -X POST -H "Content-Type: application/json" -d @"$PAYLOAD_FILE" "${API_BASE}/api/v1alpha1/intake/envelope" -o "$INTAKE_RESP")

if [ "$HTTP_CODE" -ne 200 ]; then
    echo "Error: Intake API failed with HTTP $HTTP_CODE"
    cat "$INTAKE_RESP"
    exit 1
fi
# Validate JSON
python3 -m json.tool < "$INTAKE_RESP" > /dev/null
echo "Intake API success."

echo "--------------------------------------------"
echo "3. Scheduling Explanation (POST /api/v1alpha1/schedule/explain)..."
WORKLOAD_INTENT="examples/workload-intents/vm-x86.json"
EXPLAIN_RESP="$TMP_DIR/explain_resp.json"
HTTP_CODE=$(curl -s -w "%{http_code}" -X POST -H "Content-Type: application/json" -d @"$WORKLOAD_INTENT" "${API_BASE}/api/v1alpha1/schedule/explain" -o "$EXPLAIN_RESP")

if [ "$HTTP_CODE" -ne 200 ]; then
    echo "Error: Schedule Explain API failed with HTTP $HTTP_CODE"
    cat "$EXPLAIN_RESP"
    exit 1
fi
python3 -m json.tool < "$EXPLAIN_RESP" > /dev/null
echo "Schedule Explain API success."

echo "--------------------------------------------"
echo "4. Launch Validation (POST /api/v1alpha1/launch/validate)..."
LAUNCH_SPEC="examples/launch-specs/cloudhypervisor-validate.json"
TMP_SPEC="$TMP_DIR/launch_spec.json"
VALIDATE_RESP="$TMP_DIR/validate_resp.json"

# Safely inject node_id
python3 -c "import json, sys; d = json.load(sys.stdin); d['node_id'] = '${NODE_ID}'; json.dump(d, sys.stdout)" < "$LAUNCH_SPEC" > "$TMP_SPEC"

HTTP_CODE=$(curl -s -w "%{http_code}" -X POST -H "Content-Type: application/json" -d @"$TMP_SPEC" "${API_BASE}/api/v1alpha1/launch/validate" -o "$VALIDATE_RESP")

if [ "$HTTP_CODE" -ne 200 ]; then
    echo "Error: Launch Validate API failed with HTTP $HTTP_CODE"
    cat "$VALIDATE_RESP"
    exit 1
fi
python3 -m json.tool < "$VALIDATE_RESP" > /dev/null
echo "Launch Validate API success. (is_valid status not asserted for generic CI environments)"

echo "--------------------------------------------"
echo "Smoke test completed successfully."
exit 0
