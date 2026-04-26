# Live Lab Demo

The `demo-live-lab` script provides an interactive demonstration of Schedune's `VirtualMachine` runtime lifecycle utilizing KVM and QEMU.

It runs locally on an x86_64 Linux host or nested-KVM x86_64 environment that possesses KVM extensions and the `qemu-system` binaries.

## What it does

The demo proves host node ingestion, runtime lifecycle, and backend readiness by completing the following sequence:

1. **Preflight Checks:** Ensures you have KVM extensions and QEMU.
2. **Environment Reset:** Drops the local state.
3. **Agent Inspect & Control Plane Start:** Scrapes host capabilities locally, generating a `node_id`, and pushes them via Schedune Intake.
4. **Artifact Creation:** Generates a 10MB empty `qcow2` image.
    * *Note: The demo intentionally uses an empty `qcow2` because its focus is proving the underlying KVM execution and capability engine, not booting a specific Guest OS or Application.*
5. **Explain & Predict:** Analyzes why an architecture-specific VM requiring KVM maps strictly to the local Node's Holding Pool.
6. **Validate & Dry-Run:** Confirms the Launch Specification and simulates execution.
7. **Execute:** Directly spawns a `qemu-system-*` process on the node via Schedune.
8. **Lifecycle & Readiness:** Polls until the control socket reports the virtual machine is actively listening.
9. **Trace & Events:** Shows the recorded telemetry trace and state machine events for the Launch.
10. **Termination:** Issues an API call to tear down the underlying process.
11. **Orphans:** Confirms no lingering processes exist in the Recovery engine.

## Usage

```bash
# Build Schedune binaries
make build

# Run the Live Lab interactively
make demo-live-lab

# Run the Live Lab and exit automatically
make demo-live-lab-once
```
