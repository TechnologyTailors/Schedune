# Schedune Agent Schema (v1alpha1)

## Overview
The Schedune Agent emits a rigidly structured JSON object called the `SchedulerEnvelope`. This envelope defines the `truth` of the node (hardware facts, capabilities, constraints, and health) to the control plane. The control plane relies on this schema for all scheduling, validation, and dashboard operations.

## Schema Versioning
*   **Version:** `v1alpha1`
*   **Policy:** Additive changes are non-breaking. Enum modifications or removed fields require a version bump.

## Taxonomy & Namespaces
Reason codes are governed by a strict prefix taxonomy:
*   `FACT_*`: Sourced from basic inventory.
*   `CAP_*`: Emitted by capability discovery probes.
*   `CONSTRAINT_*`: Hard blockers evaluated by collectors.
*   `ALARM_*`: Active operational alarms.
*   `CLASS_*`: Emitted by the classification engine (e.g., `CLASS_ARM_PROD_READY`).

## Core Principles
1.  **Typed Facts:** Inventory is grouped into typed structs (`CpuFacts`, `MemoryFacts`, `OsFacts`) rather than loose stringly-typed key-value pairs.
2.  **Support State vs Provenance:** A capability defines its `state` (`Supported`, `Unsupported`, `Unknown`, `Unavailable`), and explicitly declares its `provenance` (`Observed`, `Inferred`) to provide trust semantics.
3.  **Freshness Semantics:** Capabilities explicitly define `observed_at_sec` and `stale_after_sec`.
4.  **Health vs Eligibility:** A node can be operationally `Healthy` (e.g., no hardware alarms) but have a compatibility class of `Unsupported` (e.g., KVM disabled). Operational health does not equate to scheduling eligibility.
