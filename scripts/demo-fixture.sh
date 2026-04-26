#!/usr/bin/env bash
set -e

DEMO_ONCE=0
for arg in "$@"; do
    if [ "$arg" == "--once" ]; then
        DEMO_ONCE=1
    fi
done
if [ "${SCHEDUNE_DEMO_ONCE:-0}" -eq 1 ]; then
    DEMO_ONCE=1
fi

echo "============================================"
echo "      Schedune Fixture Evaluator Demo       "
echo "============================================"
echo "This mode runs the Schedune Control Plane without an active local Agent."
echo "It ingests static test fixtures to simulate a cluster state."
echo ""

# Go to the root directory
cd "$(dirname "$0")/.."

# Build the control plane
echo "[*] Building control plane..."
cd schedune-control-plane
go build -o ../bin/schedune-cp ./cmd/schedune
cd ..

# Clean old state
echo "[*] Cleaning local state..."
rm -f ./var/schedune.db

# Start the server in background
echo "[*] Starting Control Plane on :9090..."
./bin/schedune-cp server > /dev/null 2>&1 &
SERVER_PID=$!

# Trap to ensure server is killed and temp files cleaned on exit
trap "echo 'Shutting down control plane...'; kill $SERVER_PID 2>/dev/null || true; rm -f /tmp/fixture_arm.json /tmp/fixture_x86.json /tmp/freshen.py /tmp/validate.py /tmp/demo-launch.json" EXIT

# Wait for healthz
echo "[*] Waiting for control plane readiness..."
RETRIES=10
while [ $RETRIES -gt 0 ]; do
  if curl -s -f http://127.0.0.1:9090/api/v1alpha1/healthz > /dev/null; then
    echo "[*] Control plane is up!"
    break
  fi
  sleep 1
  RETRIES=$((RETRIES-1))
done

if [ $RETRIES -eq 0 ]; then
  echo "Error: Control plane failed to start."
  exit 1
fi

format_json() {
  if command -v python3 >/dev/null 2>&1; then
      python3 -m json.tool || true
  else
      cat
      echo ""
  fi
}

echo "[*] Freshening fixtures to avoid stale telemetry rejections..."
if command -v python3 >/dev/null 2>&1; then
cat << 'EOF' > /tmp/freshen.py
import json, time, sys
now = int(time.time())
stale_after = now + 3600

with open(sys.argv[1], 'r') as f:
    data = json.load(f)

data['timestamp_sec'] = now
if 'capabilities' in data:
    for cap in data['capabilities']:
        cap['observed_at_sec'] = now
        cap['stale_after_sec'] = stale_after

with open(sys.argv[2], 'w') as f:
    json.dump(data, f)
EOF
  python3 /tmp/freshen.py testdata/fixtures/cloudhypervisor_ready_arm.json /tmp/fixture_arm.json
  python3 /tmp/freshen.py testdata/fixtures/missing_kvm_x86.json /tmp/fixture_x86.json
else
  echo "[!] python3 not found. Falling back to raw fixtures (telemetry may be stale)."
  cp testdata/fixtures/cloudhypervisor_ready_arm.json /tmp/fixture_arm.json
  cp testdata/fixtures/missing_kvm_x86.json /tmp/fixture_x86.json
fi

# Ingest fixtures
echo "[*] Ingesting fixtures..."
curl -s -X POST -H "Content-Type: application/json" -d @/tmp/fixture_arm.json http://127.0.0.1:9090/api/v1alpha1/intake/envelope | format_json
echo ""
curl -s -X POST -H "Content-Type: application/json" -d @/tmp/fixture_x86.json http://127.0.0.1:9090/api/v1alpha1/intake/envelope | format_json
echo ""

echo ""
echo "[*] Listing current cluster nodes:"
curl -s http://127.0.0.1:9090/api/v1alpha1/nodes | format_json
echo ""

echo ""
echo "[*] Running scheduling explainability for ARM VM intent (Positive Path)..."
curl -s -X POST -H "Content-Type: application/json" -d @examples/workload-intents/vm-arm64.json http://127.0.0.1:9090/api/v1alpha1/schedule/explain | format_json
echo ""

echo ""
echo "[*] Running scheduling explainability for x86 VM intent (Expected Rejection Path)..."
curl -s -X POST -H "Content-Type: application/json" -d @examples/workload-intents/vm-x86.json http://127.0.0.1:9090/api/v1alpha1/schedule/explain | format_json
echo ""

echo ""
echo "[*] Evaluating launch validation against ARM node (cloud_hypervisor)..."
if command -v python3 >/dev/null 2>&1; then
  NODE_ID=$(python3 -c "import json; print(json.load(open('/tmp/fixture_arm.json'))['node_id'])")
else
  NODE_ID=$(grep -o '"node_id"[[:space:]]*:[[:space:]]*"[^"]*"' /tmp/fixture_arm.json | head -n 1 | cut -d'"' -f4)
fi

# Run validation using node ID
if command -v python3 >/dev/null 2>&1; then
cat << 'EOF' > /tmp/validate.py
import json, sys
with open(sys.argv[1], 'r') as f:
    data = json.load(f)
data['node_id'] = sys.argv[2]
data['architecture'] = 'aarch64'
with open(sys.argv[3], 'w') as f:
    json.dump(data, f)
EOF
  python3 /tmp/validate.py examples/launch-specs/cloudhypervisor-validate.json "$NODE_ID" /tmp/demo-launch.json
else
  cat examples/launch-specs/cloudhypervisor-validate.json | sed "s/\"node_id\": \".*\"/\"node_id\": \"$NODE_ID\"/" | sed "s/\"architecture\": \"x86_64\"/\"architecture\": \"aarch64\"/" > /tmp/demo-launch.json
fi

curl -s -X POST -H "Content-Type: application/json" -d @/tmp/demo-launch.json http://127.0.0.1:9090/api/v1alpha1/launch/validate | format_json
echo ""

echo ""
echo "=========================================================="
echo "Demo complete! The Schedune control plane is still running."
echo "You can interact with it on http://127.0.0.1:9090"
echo ""
echo "Try running:"
echo "  make example-nodes"
echo "  make example-node-explain NODE_ID=$NODE_ID"
echo ""
echo "Press Ctrl+C to stop the server."
echo "=========================================================="

if [ "$DEMO_ONCE" -eq 1 ]; then
    echo "Exiting one-shot demo mode."
    exit 0
fi

# Keep script running to keep server alive
wait $SERVER_PID
