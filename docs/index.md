# Schedune

Schedune is a production-grade, explainable control plane (Go) and node agent (Rust) for scheduling and managing VMs and MicroVMs across heterogeneous ARM and x86 infrastructure.

**Main Purpose:** Schedune tells infrastructure teams which machine can safely run a workload, proves why, validates runtime readiness before launch, then tracks what happened afterward.

**Who it is for:** Platform and infrastructure teams operating or evaluating mixed ARM, x86, private-cloud, and edge fleets. It is not designed for solo app developers deploying a single application.

**Example Scenario:** Your fleet consists of ARM nodes, x86 nodes, and edge boxes. A workload intent requests VM isolation, CPU, RAM, and architecture constraints. Schedune rejects incompatible nodes (e.g., due to missing KVM, stale telemetry, insufficient memory, or policy mismatch), validates the runtime path before launch, and tracks PID, readiness, events, and recovery state afterward.

**Status:** v0.1.0-alpha

## Where Schedune is today

Schedune is currently in Alpha. The lower-half control plane is working, which includes:
- Typed node truth, intake, and projection
- Eligibility explainability
- Launch validation and dry-run
- Initial runtime execution paths
- State, trace, and event tracking
- Recovery and orphan visibility
- Fixture demos

**ARM Compatibility Note:** Today, the workload architecture and intent is supplied by the user or examples. Future value will come from image/container/import evidence, but this is not yet implemented.

Schedune **is not yet** capable of full workload compatibility discovery/import, live migration, High Availability (HA), full storage/networking, or guest-internal app health.

## Features

- **Strict separation of concerns:** Agent emits immutable truth; Control plane evaluates eligibility, schedules, and validates launch before execution.
- **Explainability first:** Uses structured, typed reason codes instead of opaque strings.
- **Lifecycle management:** Rigorous state machine with append-only traces, robust restart recovery (rehydration), and real `/proc`-backed orphan sweeping.
- **Supported Runtimes:** KVM/QEMU (Execution), Cloud Hypervisor (Execution), Firecracker (Validation/Dry-run only).

*Note: Expect rapid changes as the product shape stabilizes.*
