#!/usr/bin/env bash
set -e

# Stop running processes on exit
trap 'kill $(jobs -p) 2>/dev/null || true' EXIT

echo "============================================"
echo "      Schedune Alpha Evaluator Demo         "
echo "============================================"
echo "This script will:"
echo "1. Run the preflight checks."
echo "2. Build the binaries."
echo "3. Start the Control Plane."
echo "4. Ingest your local node."
echo "5. Explain a workload scheduling decision."
echo ""

echo ""
echo "Building..."
make build

echo "Running Doctor..."
./bin/schedune doctor
echo ""

rm -f var/schedune.db
echo "Starting Control Plane..."
./bin/schedune server &
CP_PID=$!

sleep 2 # Wait for HTTP API

echo "--------------------------------------------"
echo "1. Inspecting local node (Rust Agent)..."
./bin/schedune-agent inspect > /tmp/payload.json
echo "Saved payload to /tmp/payload.json"

NODE_ID=$(grep -o '"hostname": *"[^"]*"' /tmp/payload.json | cut -d'"' -f4 | head -n 1)
if [ -z "$NODE_ID" ]; then
    # Fallback if grep fails
    NODE_ID="unknown"
fi

echo "--------------------------------------------"
echo "2. Ingesting Node (POST /api/v1alpha1/intake/envelope)..."
curl -s -X POST -H "Content-Type: application/json" -d @/tmp/payload.json http://127.0.0.1:9090/api/v1alpha1/intake/envelope

echo ""
echo "--------------------------------------------"
echo "3. Scheduling Explanation (POST /api/v1alpha1/schedule/explain)..."
echo "Simulating a workload request for an x86 VM..."
curl -s -X POST -H "Content-Type: application/json" -d @examples/workload-intents/vm-x86.json http://127.0.0.1:9090/api/v1alpha1/schedule/explain | python3 -m json.tool

echo "--------------------------------------------"
echo "Demo Complete!"
echo "The Control Plane is still running on http://127.0.0.1:9090."
echo ""
echo "To try a dry-run launch manually, edit examples/launch-specs/cloudhypervisor-validate.json"
echo "to set node_id = \"$NODE_ID\" and run:"
echo "  curl -X POST -H \"Content-Type: application/json\" -d @examples/launch-specs/cloudhypervisor-validate.json http://localhost:9090/api/v1alpha1/launch/dry-run"
echo ""
echo "Press Ctrl+C to terminate the Control Plane."

wait $CP_PID
