package schema

// Architecture
const (
	ArchAarch64 = "aarch64"
	ArchX86_64  = "x86_64"
	ArchAny     = "any"
)

// Runtime Class
const (
	RuntimeClassVirtualMachine = "VirtualMachine"
	RuntimeClassMicroVM        = "MicroVM"
	RuntimeClassContainer      = "Container"
)

// Runtime Backend
const (
	BackendKvmQemu         = "kvm_qemu"
	BackendCloudHypervisor = "cloud_hypervisor"
	BackendFirecracker     = "firecracker"
)

// Capabilities & Constraints
const (
	CapabilityStateSupported   = "Supported"
	CapabilityStateUnsupported = "Unsupported"
	CapabilityStateUnavailable = "Unavailable"
	CapabilityStateUnknown     = "Unknown"

	CapabilityKvmVmLaunch            = "kvm_vm_launch"
	CapabilityCloudHypervisorLaunch  = "cloud_hypervisor_launch"
	CapabilityFirecrackerLaunch      = "firecracker_launch"
	CapabilityHardwareTpm            = "hardware_tpm"
)

// Health States
const (
	HealthStateHealthy       = "Healthy"
	HealthStateWarning       = "Warning"
	HealthStateDegraded      = "Degraded"
	HealthStateUnschedulable = "Unschedulable"
	HealthStateQuarantined   = "Quarantined"
	HealthStateUnknown       = "Unknown"
)

// Compatibility Classes
const (
	ClassArmProduction  = "ArmProduction"
	ClassX86HoldingPool = "X86HoldingPool"
	ClassUnsupported    = "Unsupported"
	ClassDegraded       = "Degraded"
)

// Reason Codes
const (
	// Capability / Collect
	ReasonCapKvmOpenable                    = "CAP_KVM_OPENABLE"
	ReasonCapKvmMissing                     = "CAP_KVM_MISSING"
	ReasonCapKvmNotOpenablePerms            = "CAP_KVM_NOT_OPENABLE_PERMS"
	ReasonCapCloudHypervisorBinaryPresent   = "CAP_CLOUDHYPERVISOR_BINARY_PRESENT"
	ReasonCapCloudHypervisorBinaryMissing   = "CAP_CLOUDHYPERVISOR_BINARY_MISSING"
	ReasonCapCloudHypervisorReady           = "CAP_CLOUDHYPERVISOR_READY"
	ReasonCapCloudHypervisorPrereqsMissing  = "CAP_CLOUDHYPERVISOR_PREREQS_MISSING"
	ReasonCapFirecrackerBinaryPresent       = "CAP_FIRECRACKER_BINARY_PRESENT"
	ReasonCapFirecrackerBinaryMissing       = "CAP_FIRECRACKER_BINARY_MISSING"
	ReasonCapFirecrackerTunReady            = "CAP_FIRECRACKER_TUN_READY"
	ReasonCapFirecrackerTunMissing          = "CAP_FIRECRACKER_TUN_MISSING"
	ReasonCapFirecrackerCgroupsReady        = "CAP_FIRECRACKER_CGROUPS_READY"
	ReasonCapFirecrackerCgroupsMissing      = "CAP_FIRECRACKER_CGROUPS_MISSING"
	ReasonCapFirecrackerReady               = "CAP_FIRECRACKER_READY"
	ReasonCapFirecrackerPrereqsMissing      = "CAP_FIRECRACKER_PREREQS_MISSING"

	// Rejections
	ReasonRejectArchitectureMismatch       = "REJECT_ARCHITECTURE_MISMATCH"
	ReasonRejectCompatibilityClassMismatch = "REJECT_COMPATIBILITY_CLASS_MISMATCH"
	ReasonRejectMissingKVM                 = "REJECT_MISSING_KVM"
	ReasonRejectMissingTPM                 = "REJECT_MISSING_TPM"
	ReasonRejectNodeUnhealthy              = "REJECT_NODE_UNHEALTHY"
	ReasonRejectTelemetryStale             = "REJECT_TELEMETRY_STALE"
	ReasonRejectForbiddenConstraintPrefix  = "REJECT_FORBIDDEN_CONSTRAINT_"

	// Launch Validation
	ReasonErrLaunchArchMismatch                      = "ERR_LAUNCH_ARCH_MISMATCH"
	ReasonErrLaunchBackendNotSupported               = "ERR_LAUNCH_BACKEND_NOT_SUPPORTED"
	ReasonErrLaunchMissingArtifact                   = "ERR_LAUNCH_MISSING_ARTIFACT"
	ReasonErrLaunchInvalidFirecrackerArtifactModel   = "ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL"
	ReasonErrLaunchMissingCapabilityCloudHypervisor  = "ERR_LAUNCH_MISSING_CAPABILITY_CLOUDHYPERVISOR"
	ReasonErrLaunchMissingCapabilityKvmQemu          = "ERR_LAUNCH_MISSING_CAPABILITY_KVM_QEMU"

	// Execution / Preparation
	ReasonErrPreparationFailed     = "ERR_PREPARATION_FAILED"
	ReasonErrNodeNotFound          = "ERR_NODE_NOT_FOUND"
	ReasonErrValidationFailed      = "ERR_VALIDATION_FAILED"
	ReasonErrExecRuntimeSpawnFailed = "ERR_EXEC_RUNTIME_SPAWN_FAILED"
	ReasonErrExecRuntimeCrashed    = "ERR_EXEC_RUNTIME_CRASHED"
	ReasonErrExecRuntimeExitedEarly = "ERR_EXEC_RUNTIME_EXITED_EARLY"
	ReasonErrTermSignalFailed      = "ERR_TERM_SIGNAL_FAILED"

	// Readiness / Reconciliation
	ReasonErrReadyProbeFailed           = "ERR_READY_PROBE_FAILED"
	ReasonErrReadyTimeout               = "ERR_READY_TIMEOUT"
	ReasonErrReconcileProcessMissing    = "ERR_RECONCILE_PROCESS_MISSING"
	ReasonErrReconcileStatusUnreadable  = "ERR_RECONCILE_STATUS_UNREADABLE"

	// Recovery
	ReasonErrRecoveryExecutionMissing      = "ERR_RECOVERY_EXECUTION_MISSING"
	ReasonErrRecoveryReassociationAmbiguous = "ERR_RECOVERY_REASSOCIATION_AMBIGUOUS"
	ReasonErrRecoveryRehydrateFailed       = "ERR_RECOVERY_REHYDRATE_FAILED"
	ReasonErrRecoveryStaleHandle           = "ERR_RECOVERY_STALE_HANDLE"
	ReasonRecoveryConfirmed                = "RECOVERY_CONFIRMED"
	ReasonRecoveryTerminatedMissing        = "RECOVERY_TERMINATED_MISSING"
)
