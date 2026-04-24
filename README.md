# Schedune

**Schedune is an open-source control plane for explainable scheduling and managed runtime lifecycle across heterogeneous ARM and x86 infrastructure.**

**Status:** Alpha / Experimental
**Control plane:** Go
**Agent:** Rust
**Persistence:** SQLite
**Runtimes:**
- **KVM/QEMU:** execute
- **Cloud Hypervisor:** execute or validate
- **Firecracker:** validate/dry-run, execution status if applicable

Unlike generic orchestrators or traditional hypervisors, Schedune is built specifically to help organizations exit expensive legacy virtualization, adopt ARM infrastructure safely, and manage mixed fleets with lower operational risk.

## What Schedune does today

- **Node capability ingestion:** Agent inspects and emits versioned node truth.
- **Workload eligibility explanation:** Explains why workloads are rejected.
- **Backend-aware launch validation:** Catches host-level and artifact-level blockers.
- **Runtime lifecycle management:** Persistent states, append-only traces.
- **Restart recovery:** Rehydrates active workloads after a crash, surfaces orphans.
- **Orphan visibility:** Explicit orphan detection sweeping without destructive actions.

## Quickstart

Get a single-node Schedune control plane and agent running in under 5 minutes.

### 1. Clone & Doctor

Check if your local Linux host is ready:

```bash
make doctor
```

### 2. Run the Demo

Run the end-to-end evaluator journey. This builds the components, starts the control plane, inspects your local node, ingests the truth, and evaluates a sample workload intent.

```bash
make demo
```

### 3. Manual Steps

If you want to run it manually:

```bash
make build
make dev-up

# Ingest your node
./bin/schedune-agent inspect > payload.json
curl -X POST -H "Content-Type: application/json" -d @payload.json http://localhost:9090/api/v1alpha1/intake/envelope

# Schedule explanation
curl -X POST -H "Content-Type: application/json" -d @examples/workload-intents/vm-x86.json http://localhost:9090/api/v1alpha1/schedule/explain
```

## Architecture

*   **Agent Truth:** The Schedune Agent (Rust) purely observes the node and emits facts, capabilities, health, and constraints in a structured `SchedulerEnvelope`.
*   **Intake & Projection:** The Go Control Plane ingests the envelope, normalizes it, and maps it to schedulable capabilities.
*   **Eligibility & Scheduling:** WorkloadIntents are evaluated against constraints and freshness, returning `EligibilityResult`s with exact reason codes.
*   **Execution & Recovery:** The LaunchOrchestrator manages the runtime lifecycle, tracking readiness signals and persisting traces in SQLite.

## Current Limitations

Schedune is currently an alpha project. It explicitly **does not** support:
- High Availability (HA) control plane
- Auto-orphan adoption
- Live migration
- Advanced guest-service readiness (only hypervisor readiness is tracked)

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
