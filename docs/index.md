# Schedune

Schedune is a production-grade, explainable control plane (Go) and node agent (Rust) for scheduling and managing VMs and MicroVMs across heterogeneous ARM and x86 infrastructure.

**Status:** v0.1.0-alpha

## Features

- **Strict separation of concerns:** Agent emits immutable truth; Control plane evaluates eligibility, schedules, and validates launch before execution.
- **Explainability first:** Uses structured, typed reason codes instead of opaque strings.
- **Lifecycle management:** Rigorous state machine with append-only traces, robust restart recovery (rehydration), and real `/proc`-backed orphan sweeping.
- **Supported Runtimes:** KVM/QEMU (Execution), Cloud Hypervisor (Execution), Firecracker (Validation/Dry-run only).

*Note: Schedune is currently in Alpha. Expect rapid changes as the product shape stabilizes.*
