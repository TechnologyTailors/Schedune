# Troubleshooting

Schedune embraces explainable failure. Errors are heavily structured and traceable via strict reason code namespaces.

**Evaluator Tip:** If you are encountering recurring issues, run `make dev-preflight` or `./bin/schedune doctor` to check your host environment for missing dependencies like `/dev/kvm` or runtime binaries.

## Common Reason Codes

### Capability Codes (`CAP_`)
-   `CAP_KVM_MISSING`: KVM is not enabled in BIOS or kernel module is not loaded.
-   `CAP_KVM_NOT_OPENABLE_PERMS`: `/dev/kvm` exists, but the user running the Schedune agent does not have RW access.

### Scheduling Rejections (`REJECT_`)
-   `REJECT_ARCHITECTURE_MISMATCH`: The `WorkloadIntent` requested aarch64, but the node is x86_64.
-   `REJECT_COMPATIBILITY_CLASS_MISMATCH`: You requested `ArmProduction` pool, but the node is categorized as `X86HoldingPool`.
-   `REJECT_TELEMETRY_STALE`: The node has not successfully emitted an envelope within the `MaxTelemetryAgeSec` defined by the workload.

### Launch and Execution Errors (`ERR_LAUNCH_` / `ERR_EXEC_`)
-   `ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL`: Attempted to launch a `MicroVM`, but failed to provide explicit `kernel_image_path` and `rootfs_path`.
-   `ERR_EXEC_RUNTIME_SPAWN_FAILED`: The target binary failed to run (check PATH or file permissions).

### Recovery Alerts (`ERR_RECOVERY_`)
-   `ERR_RECOVERY_EXECUTION_MISSING`: After a control plane restart, an execution marked as `Running` could no longer be found on the host system. It was transitioned to `Unknown`.
