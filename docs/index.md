# Schedune

Schedune is an alpha-stage, explainable control plane (Go) and node agent (Rust) for scheduling and managing VMs and MicroVMs across heterogeneous ARM and x86 infrastructure.

**Main Purpose:** Schedune tells infrastructure teams which machine can safely run a workload, proves why, validates runtime readiness before launch, then tracks what happened afterward.

**Who it is for:** Platform and infrastructure teams operating or evaluating mixed ARM, x86, private-cloud, and edge fleets. It is not designed for solo app developers deploying a single application.

**Example Scenario:**
- **Fleet:** Mixed machines (ARM servers, x86 servers, edge mini PCs).
- **Intent:** Run a workload (e.g., payments service VM) requiring 4 vCPU, 8GB RAM, and KVM.
- **Evaluation:** Schedune inspects the fleet. It selects `arm-01` (healthy, KVM works) and explicitly rejects others like `arm-02` (missing KVM), `x86-02` (stale telemetry), and `edge-01` (insufficient memory).
- **Validation:** Before launch, it validates KVM, binaries, and safe socket paths.
- **Tracking:** After launch, it tracks the PID, backend readiness, events, and recovery state.

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
