# Quickstart

Get a single-node Schedune control plane and agent running in under 5 minutes.

## Prerequisites

- Linux host (Schedune requires Linux for execution).
- Go 1.22+ installed.
- Rust and Cargo installed.
- Required for VMs: `/dev/kvm` accessible to the current user.
- `qemu-system-aarch64` or `qemu-system-x86_64` (matching your architecture) installed.
- `cloud-hypervisor` (optional, for Cloud Hypervisor execution/validation).
- `firecracker` (optional, for MicroVM validation).

## Evaluator Flow

You can evaluate Schedune using our automated demo mode or by walking through the manual flow.

### Automated Demo Mode

The easiest way to see Schedune in action:

```bash
make demo
```

This will run the preflight checks, build the binaries, reset the database, start the control plane, ingest your local node's capabilities, and run a workload scheduling explanation.

### Manual Step-by-Step Flow

#### 1. Preflight Check

Ensure your system is capable of running Schedune:

```bash
make dev-preflight
```

#### 2. Build the Binaries

Build both the control plane (`./bin/schedune`) and the agent (`./bin/schedune-agent`):

```bash
make build
```

#### 3. Start the Control Plane

Start the control plane in the background on port `9090`:

```bash
make dev-up
```

#### 4. Ingest Local Node

Generate the node capability payload using the agent and POST it to the control plane:

```bash
make example-intake
```
*(This runs `./bin/schedune-agent inspect` and pushes the payload to `/api/v1alpha1/intake/envelope`)*

#### 5. Schedule Explain

Evaluate a sample workload intent against the ingested node:

```bash
make example-schedule
```
*(This evaluates an x86 VM intent and explains the eligibility outcome.)*

#### 6. Validate Launch

Validate a launch payload to ensure the backend is available and capabilities match, without starting the VM:

```bash
make example-launch-validate
```

#### 7. Execute Launch

If your host has the required binary (`cloud-hypervisor`), execute the launch:

```bash
make example-launch-execute
```
*Note the returned `execution_id`.*

#### 8. Inspect Lifecycle

Use the `execution_id` to inspect the VM's state, readiness, trace, and events:

```bash
# Check the launch status and trace
curl http://127.0.0.1:9090/api/v1alpha1/launch/<EXECUTION_ID>

# Check specific readiness
make example-readiness EXECUTION_ID=<EXECUTION_ID>

# Check execution events
curl http://127.0.0.1:9090/api/v1alpha1/launch/<EXECUTION_ID>/events
```

#### 9. Inspect Orphans

Schedune automatically sweeps for orphan processes. You can list them via:

```bash
make example-orphans
```

#### 10. Terminate

When you are done, terminate the execution and stop the control plane:

```bash
curl -X POST http://127.0.0.1:9090/api/v1alpha1/launch/<EXECUTION_ID>/terminate
make dev-down
```

## Common Preflight Outcomes

When running `make dev-preflight` or `./bin/schedune doctor`, you might encounter the following states:

- **Missing `/dev/kvm`:** `[INFO] /dev/kvm: missing`. You can still run the control plane, agent inspect, and API tests, but VM launches will fail.
- **Cloud Hypervisor Missing:** `[INFO] Cloud Hypervisor: not found`. Cloud Hypervisor validates and executes will fail, but QEMU might still work.
- **Firecracker Partial:** `[INFO] Firecracker: not found` or `[INFO] /dev/net/tun: missing`. Firecracker is currently only supported for validation/dry-runs. Missing it won't block other tasks.
- **API Port in Use:** `[FAIL] API Port: 127.0.0.1:9090 is in use`. You must free the port before starting the control plane.
