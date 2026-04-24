# Schedune State Machine

Schedune treats every workload launch as a strongly-managed state machine. There are no "fire and forget" commands. Every workload follows a precise track of State, Trace, and Events.

## The Triad

1.  **State:** The current, reconciled reality (`Running`, `Terminating`, `Failed`).
2.  **Trace:** The append-only historical log of how the workload reached its current state (`HostPreflight`, `RuntimeSpawn`, `Cleanup`).
3.  **Events:** Discrete occurrences along the way (`ReadinessFailed`, `ProcessSpawned`).

## Lifecycle States

-   `Pending`: Execution record created, no orchestration started.
-   `Preparing`: In the process of resolving artifacts or staging host-level config.
-   `Validated`: Launch configuration passed host capability checks.
-   `Launching`: The executor is actively spawning the process.
-   `Starting`: Process exists, but has not yet met backend-specific readiness criteria.
-   `Running`: Process is active and ready.
-   `Degraded`: Process is alive but throwing backend errors or impaired.
-   `Exited`: Process exited normally on its own.
-   `Failed`: An unrecoverable error occurred (e.g., spawn failure, readiness timeout).
-   `Terminating`: API termination request received.
-   `Terminated`: The orchestrator cleanly killed the process and released resources.
-   `Unknown`: Telemetry loss, control plane crash, or ambiguous reassociation. The loop has lost track of the workload.

## Allowed Transitions

Transitions are strictly enforced. For example, a workload cannot move directly from `Pending` to `Running`. It must step through `Preparing`, `Validated`, `Launching`, and `Starting`.

If Schedune cannot verify a transition, it moves the workload to `Unknown` rather than fabricating state.
