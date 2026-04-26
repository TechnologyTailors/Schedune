#!/bin/bash
set -euo pipefail

ONCE_MODE=false
if [[ "${1:-}" == "--once" ]]; then
    ONCE_MODE=true
fi

# Define paths
BIN_DIR="bin"
AGENT_BIN="${BIN_DIR}/schedune-agent"
CONTROL_PLANE_BIN="${BIN_DIR}/schedune"
DB_PATH="var/schedune.db"
LIVE_LAB_DIR="var/live-lab"
QEMU_IMG_PATH="${LIVE_LAB_DIR}/test-vm.qcow2"
HOST="127.0.0.1:9090"
API_BASE="http://${HOST}/api/v1alpha1"

export PATH="$PATH:$HOME/.cargo/bin"

# Cleanup function
cleanup() {
    echo -e "\n[CLEANUP] Cleaning up..."
    if [[ -n "${CP_PID:-}" ]] && kill -0 "$CP_PID" 2>/dev/null; then
        echo "[CLEANUP] Stopping control plane (PID: $CP_PID)..."
        kill "$CP_PID"
        wait "$CP_PID" 2>/dev/null || true
    fi
    echo "[CLEANUP] Cleanup complete."
}
trap cleanup EXIT

echo "[INFO] Starting Schedune Live Lab Demo..."

# 1. Preflight
echo "[INFO] Running Preflight Checks..."
if [[ ! -f "$CONTROL_PLANE_BIN" ]] || [[ ! -f "$AGENT_BIN" ]]; then
    echo "[ERROR] Required binaries not found. Run 'make build' first."
    exit 1
fi

for cmd in python3 curl qemu-img; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "[ERROR] Required command not found: $cmd"
        exit 1
    fi
done

if [[ "$(uname -m)" == "x86_64" ]]; then
    if ! command -v qemu-system-x86_64 >/dev/null 2>&1; then
        echo "[ERROR] Required command not found: qemu-system-x86_64"
        exit 1
    fi
fi

if ! ./$CONTROL_PLANE_BIN doctor; then
    echo "[ERROR] Preflight failed. Cannot run live lab demo. Ensure KVM and QEMU are available."
    exit 1
fi

# 2. Reset DB and create dirs
echo "[INFO] Resetting environment..."
rm -f "$DB_PATH"
mkdir -p "$LIVE_LAB_DIR"

# 3. Start control plane
echo "[INFO] Starting Control Plane..."
./$CONTROL_PLANE_BIN server &
CP_PID=$!

echo "[INFO] Waiting for Control Plane API to become ready..."
API_READY=false
for i in {1..15}; do
    if curl -fsS "http://${HOST}/api/v1alpha1/healthz" > /dev/null 2>&1; then
        API_READY=true
        break
    fi
    sleep 1
done

if [[ "$API_READY" == "false" ]]; then
    echo "[ERROR] Control Plane API failed to become ready."
    exit 1
fi

# 4. Agent inspect and ingest
echo "[INFO] Running Agent Inspect..."
AGENT_OUTPUT=$(./$AGENT_BIN inspect)

echo "[INFO] Ingesting live node truth..."
curl -s -X POST "${API_BASE}/intake/envelope" \
     -H "Content-Type: application/json" \
     -d "$AGENT_OUTPUT" > /dev/null

# 5. Derive Node ID
NODE_ID=$(python3 -c "import sys, json; print(json.loads(sys.stdin.read())['node_id'])" <<< "$AGENT_OUTPUT")
echo "[OK] Node ID: $NODE_ID"

# 6. Create tiny local QEMU qcow2 artifact
echo "[INFO] Creating QEMU image artifact at $QEMU_IMG_PATH..."
echo "[NOTE] We use an empty 10MB qcow2. The demo proves host/runtime lifecycle"
echo "       and backend readiness, not guest OS or application readiness."
rm -f "$QEMU_IMG_PATH"
qemu-img create -f qcow2 "$QEMU_IMG_PATH" 10M > /dev/null

# 7. Generate base launch spec
echo "[INFO] Generating Launch Spec..."
cat <<EOF > "${LIVE_LAB_DIR}/launch-spec.json"
{
  "schema_version": "v1alpha1",
  "workload_id": "live-lab-demo-vm",
  "tenant_id": "tenant-live-lab",
  "node_id": "$NODE_ID",
  "runtime_class": "VirtualMachine",
  "runtime_backend_preference": "kvm_qemu",
  "architecture": "x86_64",
  "memory_mb": 128,
  "vcpu": 1,
  "storage": [
    {
      "host_path": "$PWD/$QEMU_IMG_PATH",
      "format": "qcow2"
    }
  ]
}
EOF

# 8. Schedule Explain
echo "[INFO] Explaining schedule eligibility for x86_64 VM requiring KVM and X86HoldingPool..."
curl -s "${API_BASE}/schedule/explain" \
     -H "Content-Type: application/json" \
     -d '{
           "schema_version": "v1alpha1",
           "workload_id": "live-lab-demo-vm",
           "tenant_id": "tenant-live-lab",
           "runtime_class": "VirtualMachine",
           "required_architecture": "x86_64",
           "max_telemetry_age_sec": 600,
           "requires_kvm": true,
           "required_compatibility_classes": ["X86HoldingPool"]
         }' | python3 -m json.tool

# Helper to inject launch_mode
inject_mode() {
    local mode=$1
    python3 -c "import sys, json; d=json.load(sys.stdin); d['launch_mode']='$mode'; print(json.dumps(d))" < "${LIVE_LAB_DIR}/launch-spec.json"
}

# 9. Validate Launch
echo "[INFO] Validating Launch..."
VALIDATE_RESPONSE=$(inject_mode "Validate" | curl -s "${API_BASE}/launch/validate" \
     -H "Content-Type: application/json" \
     -d @-)
echo "$VALIDATE_RESPONSE" | python3 -m json.tool

VALIDATION_STATUS=$(python3 -c "import sys, json; print(json.loads(sys.stdin.read()).get('is_valid', False))" <<< "$VALIDATE_RESPONSE")
if [[ "$VALIDATION_STATUS" != "True" ]]; then
    echo "[ERROR] Launch validation failed. Response: $VALIDATE_RESPONSE"
    exit 1
fi

# 10. Dry Run Launch
echo "[INFO] Dry-running Launch..."
inject_mode "DryRun" | curl -s "${API_BASE}/launch/dry-run" \
     -H "Content-Type: application/json" \
     -d @- | python3 -m json.tool

# 11. Execute Launch
if [[ "$ONCE_MODE" == "false" ]]; then
    read -p "Press Enter to EXECUTE a real KVM-backed VM... (or Ctrl+C to abort)"
fi

echo "[INFO] Executing Launch..."
EXEC_RESPONSE=$(inject_mode "Execute" | curl -s "${API_BASE}/launch/execute" \
     -H "Content-Type: application/json" \
     -d @-)

echo "$EXEC_RESPONSE" | python3 -m json.tool
EXECUTION_ID=$(python3 -c "import sys, json; d=json.loads(sys.stdin.read()); print(d.get('execution_id', ''))" <<< "$EXEC_RESPONSE")

if [[ -z "$EXECUTION_ID" || "$EXECUTION_ID" == "null" ]]; then
    echo "[ERROR] Execution failed to return an execution_id."
    exit 1
fi
echo "[OK] Execution ID: $EXECUTION_ID"

# 12. Poll Readiness
echo "[INFO] Polling readiness for up to 10 seconds..."
IS_READY=false
for i in {1..10}; do
    READINESS=$(curl -s "${API_BASE}/launch/${EXECUTION_ID}/readiness")
    STATUS=$(python3 -c "import sys, json; print(json.loads(sys.stdin.read()).get('runtime_readiness', ''))" <<< "$READINESS")
    echo "  Status: $STATUS"
    if [[ "$STATUS" == "Ready" ]]; then
        IS_READY=true
        break
    elif [[ "$STATUS" == "Failed" || "$STATUS" == "Unknown" ]]; then
        echo "[ERROR] Readiness polling failed with terminal status: $STATUS"
        exit 1
    fi
    sleep 1
done

if [[ "$IS_READY" == "false" ]]; then
    echo "[ERROR] Execution failed to become Ready within timeout."
    exit 1
fi

# 13. Show Trace and Events
echo "[INFO] Trace:"
curl -s "${API_BASE}/launch/${EXECUTION_ID}/trace" | python3 -m json.tool || true
echo "[INFO] Events:"
curl -s "${API_BASE}/launch/${EXECUTION_ID}/events" | python3 -m json.tool || true

# 14. Terminate Execution
if [[ "$ONCE_MODE" == "false" ]]; then
    read -p "Press Enter to TERMINATE the execution..."
fi

echo "[INFO] Terminating Execution..."
curl -s -X POST "${API_BASE}/launch/${EXECUTION_ID}/terminate" | python3 -m json.tool || true

echo "[INFO] Checking for Orphans..."
curl -s "${API_BASE}/recovery/orphans" | python3 -m json.tool || true

echo "[SUCCESS] Live Lab Demo completed successfully!"