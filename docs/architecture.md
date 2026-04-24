# Architecture Overview

Schedune follows a strict separation of concerns, heavily prioritizing truth, predictability, and explainability over ad-hoc assumptions.

## Core Concepts

### Northbound: Truth, Intent, and Policy

*   **Agent Truth:** The Schedune Agent (Rust) purely observes the node and emits facts, capabilities, health, and constraints in a structured `SchedulerEnvelope`. It does NOT make scheduling policy decisions.
*   **Intake & Projection:** The Go Control Plane ingests the envelope, normalizes it, and maintains a distinct separation between operational health, compatibility classifications, and schedulable capabilities.
*   **Eligibility:** Given a WorkloadIntent, Schedune filters nodes by rigid hard constraints, freshness bounds, and compatibility matching, returning an `EligibilityResult`.
*   **Explainable Scheduling:** Rejections generate exact, stable reason codes, allowing operators to understand exactly why a workload failed to schedule on a specific host.

### Southbound: Execution Validation and Lifecycle

*   **Launch Validation:** Once scheduled, the chosen `LaunchSpec` is verified against the host capabilities before any execution attempt. E.g., if a Firecracker microVM is requested, the Launch Validator guarantees `/dev/net/tun`, cgroups, and split kernel/rootfs models are met.
*   **Orchestration & Execution:** The LaunchOrchestrator resolves the request to the correct executor (`KvmExecutor`, `CloudHypervisorExecutor`, etc.), runs preparation logic, and spawns the process.
*   **Managed Lifecycle:** Once a workload is running, the Reconciler periodically verifies its liveness and readiness state using backend-specific inspections, maintaining an append-only trace of state transitions.
*   **Durable Recovery:** (Phase 6) State is persisted in SQLite. On Control Plane restart, orphaned executions are gracefully reassociated or classified as `Unknown` to prevent phantom execution states.
