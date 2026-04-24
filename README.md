# Schedune

**Explainable scheduling and managed runtime lifecycle for heterogeneous ARM and x86 infrastructure.**

**Status:** Alpha / Experimental (Single-node technical preview)
**Control Plane:** Go
**Agent:** Rust
**Persistence:** SQLite
**Supported Runtimes:**
- **KVM/QEMU:** Execute
- **Cloud Hypervisor:** Execute
- **Firecracker:** Validate / Dry-Run (Execution coming soon)

Schedune is an open-source control plane built specifically to help organizations exit expensive legacy virtualization, adopt ARM infrastructure safely, and manage mixed fleets with lower operational risk.

It achieves this by enforcing a strict boundary between **node truth** (what the hardware can actually do), **workload intent** (what the workload requires), and **execution readiness** (proving a launch will succeed before it is attempted).

## What works today

- [x] Agent inspects and emits versioned node truth (Capabilities, Constraints, Facts, Health)
- [x] Control plane ingests `SchedulerEnvelope`
- [x] Workload intent evaluated against node truth
- [x] Scheduler explain endpoint explains rejections
- [x] Dry-run launch validation catches host-level and artifact-level blockers
- [x] KVM-backed VM and Cloud Hypervisor execution works
- [x] Runtime lifecycle is persisted to SQLite
- [x] Restart recovery is supported (orphans surfaced, active workloads rehydrated)

## Intentionally NOT supported yet

- High Availability (HA) control plane
- Live migration & snapshots
- Advanced storage & networking orchestration
- Guest-service readiness (currently only hypervisor readiness is tracked)
- Firecracker full execution (validation-first)

## Prerequisites

- **OS:** Linux
- **Control Plane:** Go 1.22+
- **Agent:** Rust (cargo)
- **Virtualization:** `/dev/kvm` must be present and openable (RW)
- **Binaries:** `qemu-system-aarch64` / `qemu-system-x86_64`, `cloud-hypervisor`, `firecracker` (optional for validation)
- **Host features:** `/dev/net/tun`, cgroups v2 (for Firecracker)

## Quickstart

Get a single-node Schedune control plane and agent running in under 5 minutes.

### 1. Build and Start the Control Plane
```bash
cd schedune-control-plane
go build ./cmd/intake
./intake &
```

### 2. Build and Run the Node Agent
```bash
cd schedune-agent
cargo build --release
./target/release/schedune_agent inspect > payload.json
```

### 3. Ingest Node Truth
```bash
curl -X POST -H "Content-Type: application/json" -d @payload.json http://localhost:9090/api/v1alpha1/intake/envelope
```

### 4. Explain a Scheduling Decision
Create `workload.json`:
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
```bash
curl -X POST -H "Content-Type: application/json" -d @workload.json http://localhost:9090/api/v1alpha1/schedule/explain
```

### 5. Execute a Workload
Create `launch.json`:
```json
{
  "schema_version": "v1alpha1",
  "workload_id": "wl-test-001",
  "tenant_id": "tenant-1",
  "node_id": "<YOUR_NODE_ID_FROM_PAYLOAD>",
  "runtime_class": "VirtualMachine",
  "architecture": "x86_64",
  "image_reference": "/path/to/dummy.qcow2",
  "vcpu": 2,
  "memory_mb": 1024,
  "launch_mode": "Execute"
}
```
```bash
curl -X POST -H "Content-Type: application/json" -d @launch.json http://localhost:9090/api/v1alpha1/launch/execute
```

### 6. Inspect Lifecycle
```bash
curl http://localhost:9090/api/v1alpha1/launch/<EXECUTION_ID>
```

## Documentation

- [Quickstart](docs/quickstart.md)
- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api.md)
- [Runtime Support](docs/runtime-support.md)
- [Troubleshooting](docs/troubleshooting.md)
- [State Machine](docs/state-machine.md)
- [Restart Recovery](docs/recovery.md)

## License

Copyright 2026 Technology Tailors. Licensed under the Apache License, Version 2.0.
