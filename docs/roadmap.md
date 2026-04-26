# Schedune Roadmap

*Note: Schedune is currently in an early alpha stage. This roadmap is an early public outline and is subject to change as the product shape stabilizes.*

## Achievements So Far

Schedune has made significant progress in establishing a reliable, explainable foundation for managing VMs and MicroVMs:

- **Agent Truth Ingestion:** Node agent accurately inspects and emits immutable, versioned node capabilities.
- **Explainable Eligibility:** Control plane evaluates workloads and provides explicit, typed reason codes for rejection.
- **Backend-Aware Launch Validation:** Catches host-level, artifact-level, and runtime version blockers before execution.
- **Runtime Readiness & Recovery:** Rigorous state machine, persistent states, and restart recovery capable of rehydrating active workloads.
- **Orphan Visibility:** Real `/proc`-backed orphan detection and sweeping without destructive actions.
- **Dry-Run Preparation:** Realistic dry-run capabilities across runtimes to validate specs safely.
- **Tooling & Infrastructure:** Docs website, E2E smoke tests, API contract testing, and CI automated validation workflows.

## Milestones

* **Alpha Foundation:** Establishing control plane API, node agent truth ingestion, and explainable eligibility. *(Completed)*
* **Data Plane V0:** Robust launch validation, dry-run capabilities, state machine recovery, and orphan visibility. *(Current Focus)*
* **First Real Runtime Execution:** Active development of execution paths for QEMU, Cloud Hypervisor, and Firecracker with networking and storage support.
* **Technical Preview:** Feature-complete early release, hardened API schema, and robust CLI tooling for initial evaluation.

## Top 20 Next Tasks

Our immediate focus is refining the Schedune Data Plane V0, hardening launch validation, and establishing the first real execution paths while keeping migration-safe infrastructure principles in mind.

1. Implement Data Plane V0 real execution path for KVM/QEMU.
2. Implement Data Plane V0 real execution path for Cloud Hypervisor.
3. Integrate full execution support for Firecracker.
4. Establish robust runtime readiness probes and health checks.
5. Enhance structured backend rejection evidence across all supported runtimes.
6. Support advanced storage configurations and explicit block device mapping.
7. Support advanced network interface configuration (TAP/TUN integration).
8. Implement resource isolation (cgroups, namespaces) enforcement.
9. Improve agent capability discovery for specific hardware extensions.
10. Finalize the initial `v1alpha1` API schema for launch specs and workload intents.
11. Implement secure artifact fetching and local caching.
12. Establish comprehensive telemetry and logging for active workloads.
13. Refine restart recovery mechanisms for complex execution states.
14. Improve orphan workload classification and graceful reconciliation.
15. Add comprehensive API documentation and OpenAPI specifications.
16. Create a user-friendly CLI for interacting with the Schedune control plane.
17. Expand CI/CD with rigorous multi-architecture (ARM/x86) integration tests.
18. Document migration paths and strategies for safe infrastructure transitions.
19. Provide detailed error remediation guides for common launch failures.
20. Prepare a stable, feature-complete technical preview release for early adopters.
