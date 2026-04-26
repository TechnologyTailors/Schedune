#!/usr/bin/env bash
set -e

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

# Trap to ensure server is killed on exit
trap "echo 'Shutting down control plane...'; kill $SERVER_PID" EXIT

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

# Ingest fixtures
echo "[*] Ingesting fixtures..."
curl -s -X POST -H "Content-Type: application/json" -d @testdata/fixtures/healthy_arm_production.json http://127.0.0.1:9090/api/v1alpha1/intake/envelope | format_json
echo ""
curl -s -X POST -H "Content-Type: application/json" -d @testdata/fixtures/missing_kvm_x86.json http://127.0.0.1:9090/api/v1alpha1/intake/envelope | format_json
echo ""

echo ""
echo "[*] Listing current cluster nodes:"
curl -s http://127.0.0.1:9090/api/v1alpha1/nodes | format_json
echo ""

echo ""
echo "[*] Running scheduling explainability for ARM microVM intent..."
curl -s -X POST -H "Content-Type: application/json" -d @examples/workload-intents/microvm-firecracker.json http://127.0.0.1:9090/api/v1alpha1/schedule/explain | format_json
echo ""

echo ""
echo "[*] Evaluating launch validation against mocked ARM node..."
# We assume healthy_arm_production.json has node_id "fixture-healthy-arm-prod-001" or similar. Let's look up its node ID by asking the API or parsing the file.
NODE_ID=$(cat testdata/fixtures/healthy_arm_production.json | grep '"node_id"' | head -n 1 | awk -F'"' '{print $4}')

# Run validation using node ID
# wait, the launch specs need the correct node ID. I can use sed to inject it.
cat examples/launch-specs/firecracker-dryrun.json | sed "s/\"node_id\": \".*\"/\"node_id\": \"$NODE_ID\"/" > /tmp/demo-launch.json
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

# Keep script running to keep server alive
wait $SERVER_PID
