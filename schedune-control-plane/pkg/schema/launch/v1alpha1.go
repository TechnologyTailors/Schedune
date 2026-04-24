package launch

// LaunchSpec defines the runtime configuration to validate or dry-run.
type LaunchSpec struct {
	SchemaVersion            string   `json:"schema_version" binding:"required,eq=v1alpha1"`
	WorkloadID               string   `json:"workload_id" binding:"required"`
	TenantID                 string   `json:"tenant_id" binding:"required"`
	NodeID                   string   `json:"node_id" binding:"required"`
	RuntimeClass             string   `json:"runtime_class" binding:"required,oneof=VirtualMachine MicroVM"`
	Architecture             string   `json:"architecture" binding:"required,oneof=aarch64 x86_64"`
	ImageReference           string   `json:"image_reference"` // Optional for Firecracker
	KernelImagePath          string   `json:"kernel_image_path"`
	RootfsPath               string   `json:"rootfs_path"`
	Vcpu                     int      `json:"vcpu" binding:"required,gt=0"`
	MemoryMB                 int64    `json:"memory_mb" binding:"required,gt=0"`
	NetworkAttachments       []string `json:"network_attachments"`
	LaunchMode               string   `json:"launch_mode" binding:"required,oneof=Validate DryRun Execute"`
	RuntimeBackendPreference string   `json:"runtime_backend_preference"`
	AllowBackendFallback     bool     `json:"allow_backend_fallback"`
}

// LaunchValidationResult explains exactly what host-level blockers exist.
type LaunchValidationResult struct {
	IsValid              bool              `json:"is_valid"`
	SelectedBackend      string            `json:"selected_backend"`
	RejectedBackends     map[string]string `json:"rejected_backends"` // Backend -> Reason code
	BlockingReasonCodes  []string          `json:"blocking_reason_codes"`
	Warnings             []string          `json:"warnings"`
	RequiredHostFeatures []string          `json:"required_host_features"`
	MissingHostFeatures  []string          `json:"missing_host_features"`
	RecommendedRuntime   string            `json:"recommended_runtime"`
	ExplainabilityText   string            `json:"explainability_text"`
	ValidationTrace      []string          `json:"validation_trace"`
}

type LaunchState string

const (
	StatePending     LaunchState = "Pending"
	StatePreparing   LaunchState = "Preparing"
	StateValidated   LaunchState = "Validated"
	StateLaunching   LaunchState = "Launching"
	StateStarting    LaunchState = "Starting"
	StateRunning     LaunchState = "Running"
	StateDegraded    LaunchState = "Degraded"
	StateExited      LaunchState = "Exited"
	StateFailed      LaunchState = "Failed"
	StateTerminating LaunchState = "Terminating"
	StateTerminated  LaunchState = "Terminated"
	StateUnknown     LaunchState = "Unknown"
)

type ExecutionTraceStep struct {
	Stage        string `json:"stage"`
	Status       string `json:"status"` // "Success", "Failed"
	ReasonCode   string `json:"reason_code,omitempty"`
	Message      string `json:"message"`
	TimestampSec int64  `json:"timestamp_sec"`
}

type PreparedQemuLaunch struct {
	BinaryPath   string   `json:"binary_path"`
	ArtifactPath string   `json:"artifact_path"`
	CommandArgs  []string `json:"command_args"`
}

type PreparedCloudHypervisorLaunch struct {
	BinaryPath   string   `json:"binary_path"`
	ArtifactPath string   `json:"artifact_path"`
	CommandArgs  []string `json:"command_args"`
}

type PreparedFirecrackerLaunch struct {
	BinaryPath      string   `json:"binary_path"`
	KernelImagePath string   `json:"kernel_image_path"`
	RootfsPath      string   `json:"rootfs_path"`
	CommandArgs     []string `json:"command_args"`
}

type PreparedLaunch struct {
	RuntimeBackend  string                         `json:"runtime_backend"`
	MemoryMB        int64                          `json:"memory_mb"`
	Vcpu            int                            `json:"vcpu"`
	KvmQemu         *PreparedQemuLaunch            `json:"kvm_qemu,omitempty"`
	CloudHypervisor *PreparedCloudHypervisorLaunch `json:"cloud_hypervisor,omitempty"`
	Firecracker     *PreparedFirecrackerLaunch     `json:"firecracker,omitempty"`
}

type LaunchExecutionRecord struct {
	ExecutionID       string               `json:"execution_id"`
	WorkloadID        string               `json:"workload_id"`
	NodeID            string               `json:"node_id"`
	Spec              LaunchSpec           `json:"spec"`
	PreparedState     *PreparedLaunch      `json:"prepared_state,omitempty"`
	State             LaunchState          `json:"state"`
	RuntimeLiveness   string               `json:"runtime_liveness"`  // e.g. "Alive", "Dead", "Unknown"
	RuntimeReadiness  string               `json:"runtime_readiness"` // e.g. "NotStarted", "Ready", "Failed"
	CreatedAtSec      int64                `json:"created_at_sec"`
	UpdatedAtSec      int64                `json:"updated_at_sec"`
	StartedAtSec      *int64               `json:"started_at_sec,omitempty"`
	ReadyAtSec        *int64               `json:"ready_at_sec,omitempty"`
	TerminatedAtSec   *int64               `json:"terminated_at_sec,omitempty"`
	PID               *int                 `json:"pid,omitempty"`
	ExitCode          *int                 `json:"exit_code,omitempty"`
	FailureReasonCode *string              `json:"failure_reason_code,omitempty"`
	Trace             []ExecutionTraceStep `json:"trace"`
	Generation        int64                `json:"generation"`
	RecoveryEpoch     string               `json:"recovery_epoch,omitempty"`
	LastObservedAtSec *int64               `json:"last_observed_at_sec,omitempty"`
}

type RuntimeEvent struct {
	EventID      string      `json:"event_id"`
	ExecutionID  string      `json:"execution_id"`
	EventType    string      `json:"event_type"`
	TimestampSec int64       `json:"timestamp_sec"`
	ReasonCode   string      `json:"reason_code,omitempty"`
	PayloadJSON  interface{} `json:"payload_json,omitempty"`
}
