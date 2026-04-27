# API Reference

Schedune provides a clear, RESTful interface for scheduling, validating, executing, and inspecting workloads.

## Endpoints

### Data Plane Intake
*   `POST /api/v1alpha1/intake/envelope`: Ingests the `SchedulerEnvelope` emitted by the Schedune Agent. Updates the node's normalized truth in the SQLite store.

### Explainability
*   `GET /api/v1alpha1/nodes/:id/explain`: Returns deep operational transparency about a node, outlining its architecture, constraints, health, and explicitly declaring what backend classes it is eligible to run (e.g., `can_run_kvm: true`).

### Orchestration
*   `POST /api/v1alpha1/schedule/explain`: Given a `WorkloadIntent`, evaluates all nodes and returns `RankedNodes` and `RejectedNodes` with explicit `HardRejectionCodes`.
*   `POST /api/v1alpha1/schedule/select`: Simulates placement and returns the optimal node ID for the given intent.
*   `POST /api/v1alpha1/plan/launch`: Bridges scheduling and execution. Returns a hydrated `LaunchSpec`, explains rejections, and optionally performs a dry run preparation without executing the runtime.

### Execution
*   `POST /api/v1alpha1/launch/validate`: Given a `LaunchSpec`, assesses if the specific host has the prerequisites to run the chosen backend without launching the process.
*   `POST /api/v1alpha1/launch/dry-run`: Runs the `Validate` loop and also steps through the `Prepare` phase of the executor to verify argument syntax and artifact existence.
*   `POST /api/v1alpha1/launch/execute`: Spawns the workload as a managed state machine. Returns the initial `LaunchExecutionRecord`.
*   `GET /api/v1alpha1/launch/:id`: Retrieves the live, reconciled `LaunchExecutionRecord` including its liveness, readiness, and trace.
*   `POST /api/v1alpha1/launch/:id/terminate`: Gracefully issues a kill to the underlying PID and sweeps the trace to `Terminated`.
