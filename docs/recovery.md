# Restart Recovery

Schedune is designed under the assumption that the Control Plane may crash, reboot, or become disconnected from its nodes at any time. Schedune guarantees **Restart Resilience**.

## The Philosophy

-   **Persistence:** The SQLite execution store is the source of truth for *what should be running*.
-   **Inspection:** The runtime process table is the source of truth for *what is actually running*.
-   **No Guessing:** Schedune will never silently relaunch, implicitly kill, or pretend a workload is healthy after a crash. If evidence is ambiguous, it goes to `Unknown`.

## Bootstrap Sequence

When the Schedune Control Plane starts, it performs a **Recovery Bootstrap**:

1.  **Load Recoverable Executions:** Pulls all non-terminal (`Starting`, `Running`, `Degraded`, `Terminating`, `Unknown`) records from the database.
2.  **Stamp Recovery Epoch:** Associates the current boot process with a new UUID epoch for auditability.
3.  **Inspect Identities:** Uses the PID and backend markers to verify the process still exists on the host.
4.  **Classify and Transition:**
    -   Process exists and matches identity: Emits `ExecutionRehydrated`, state remains `Running`.
    -   Process missing but was Running: Safely transitions to `Unknown` (it may have exited legitimately during the blackout).
    -   Mid-flight during crash (e.g., `Launching` before a PID was assigned): Emits `ERR_RECOVERY_STALE_HANDLE` and transitions to `Unknown`.
    -   `Terminating` process is missing: Concludes the termination completed and transitions to `Terminated`.

## Orphan Reconciliation (Upcoming)

Orphans are runtime processes that appear to belong to Schedune but have no durable record. Schedune surfaces orphan candidates with an explicit taxonomy (e.g., `OrphanUnmanagedBackendProcess`, `OrphanStaleExecutionArtifact`) but intentionally **does not automatically kill or adopt them** without explicit operator policy.
