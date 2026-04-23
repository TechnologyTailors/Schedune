# Schedune

**Schedune** is an explainable infrastructure control plane designed for heterogeneous ARM and x86 fleets.

Unlike generic orchestrators or traditional hypervisors, Schedune is built specifically to help organizations exit expensive legacy virtualization, adopt ARM infrastructure safely, and manage mixed fleets with lower operational risk.

It achieves this by enforcing a strict boundary between **node truth** (what the hardware can actually do), **workload intent** (what the workload requires), and **execution readiness** (proving a launch will succeed before it is attempted).

## The Business Value

Schedune is built around three core economic promises:

1. **Lower Infrastructure Cost:** Capitalize on the proven price-performance advantages of modern ARM silicon (like AWS Graviton, Google Axion, and Oracle Ampere) without sacrificing the ability to run legacy x86 workloads in designated holding pools.
2. **Lower Migration Risk:** Cross-ISA migration isn't magic. Schedune explicitly surfaces compatibility classes, launch prerequisites, and hard blockers so you know *before* scheduling whether an imported VM will actually run.
3. **Lower Operational Risk:** At 3:00 AM, operators don't need opaque "No Nodes Available" errors. They need to know exactly why a workload was rejected, whether a node's telemetry is stale, or if a required virtualization capability (like `/dev/kvm`) is missing. Schedune makes every placement and launch decision fully explainable.

## Architecture

The platform consists of two primary boundaries:

### The Northbound Layer (Scheduling & Eligibility)
Answers: *What is the node? What does the workload need? Why was it accepted or rejected?*

*   **Schedune Agent (Rust):** A lightweight, memory-safe daemon that runs on the host. It does not make policy decisions. It solely emits versioned, strongly-typed "truth" (Capabilities, Constraints, Facts, and Health) about the node's hardware and virtualization readiness.
*   **Control Plane (Go):** Ingests the agent's truth, projects it into queryable state, evaluates workload intent (e.g., "Requires KVM, Requires ARM"), and makes deterministic, explainable scheduling decisions.

### The Southbound Layer (Execution Readiness)
Answers: *Can this node actually launch the selected runtime? What exact host-side blockers exist?*

*   **Data Plane V0:** Before any shell commands are executed, Schedune validates the required runtime features (e.g., KVM or Firecracker prerequisites) against the node's capabilities, allowing operators to dry-run launch specifications and catch failures before execution.

## Getting Started

### The Control Plane
Written in Go, providing the intake APIs and scheduling logic.
```bash
cd schedune-control-plane
go build ./cmd/intake
./intake
```

### The Node Agent
Written in Rust, providing host telemetry and capability discovery.
```bash
cd schedune-agent
cargo build --release
# To see what the control plane sees:
./target/release/schedune_agent inspect
```

## License

Copyright 2026 Technology Tailors. Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
