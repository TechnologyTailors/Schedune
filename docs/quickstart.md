# Quickstart

Get a single-node Schedune control plane and agent running in under 5 minutes.

## Prerequisites

- Linux host with `/dev/kvm` accessible.
- Go 1.22+ installed.
- Rust and Cargo installed.
- `qemu-system-aarch64` or `qemu-system-x86_64` (matching your architecture) installed to launch standard VMs.
- `cloud-hypervisor` (optional, for Cloud Hypervisor tests).

## 1. Build and Start the Control Plane

```bash
cd schedune-control-plane
go build ./cmd/intake
./intake &
```

## 2. Build and Run the Node Agent

```bash
cd schedune-agent
cargo build --release
./target/release/schedune_agent inspect > payload.json
```

## 3. Ingest Node Truth

Post the Node capabilities and health envelope to the intake endpoint.

```bash
curl -X POST -H "Content-Type: application/json" -d @payload.json http://localhost:9090/api/v1alpha1/intake/envelope
```

## 4. Evaluate and Schedule

Create a WorkloadIntent representing your workload requirements:

`workload.json`
```json
{
  "schema_version": "v1alpha1",
  "workload_id": "wl-test-001",
  "tenant_id": "tenant-1",
  "runtime_class": "VirtualMachine",
  "required_architecture": "x86_64",
  "max_telemetry_age_sec": 600,
  "requires_kvm": true,
  "required_compatibility_classes": ["X86HoldingPool"]
}
```

Run the scheduler explain endpoint to see which nodes are eligible and why.

```bash
curl -X POST -H "Content-Type: application/json" -d @workload.json http://localhost:9090/api/v1alpha1/schedule/explain
```

## 5. Execute the Launch

Create a dummy image to run:
```bash
qemu-img create -f qcow2 dummy.qcow2 1G
```

Create a LaunchSpec describing the execution intent:

`launch.json`
```json
{
  "schema_version": "v1alpha1",
  "workload_id": "wl-test-001",
  "tenant_id": "tenant-1",
  "node_id": "<INSERT_NODE_ID_FROM_PAYLOAD_HERE>",
  "runtime_class": "VirtualMachine",
  "architecture": "x86_64",
  "image_reference": "dummy.qcow2",
  "vcpu": 2,
  "memory_mb": 1024,
  "launch_mode": "Execute"
}
```

Launch the workload:

```bash
curl -X POST -H "Content-Type: application/json" -d @launch.json http://localhost:9090/api/v1alpha1/launch/execute
```

## 6. Inspect the Lifecycle Trace

Grab the `execution_id` from the previous output.

```bash
curl http://localhost:9090/api/v1alpha1/launch/<EXECUTION_ID>
```

This returns a `LaunchExecutionRecord` containing the full `trace` of preparation, spawn, and liveness.

## 7. Terminate

```bash
curl -X POST http://localhost:9090/api/v1alpha1/launch/<EXECUTION_ID>/terminate
```
