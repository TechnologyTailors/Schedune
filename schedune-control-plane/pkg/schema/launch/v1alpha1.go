package launch

// LaunchSpec defines the runtime configuration to validate or dry-run.
type LaunchSpec struct {
	SchemaVersion      string   `json:"schema_version" binding:"required,eq=v1alpha1"`
	WorkloadID         string   `json:"workload_id" binding:"required"`
	TenantID           string   `json:"tenant_id" binding:"required"`
	NodeID             string   `json:"node_id" binding:"required"`
	RuntimeClass       string   `json:"runtime_class" binding:"required,oneof=VirtualMachine MicroVM"`
	Architecture       string   `json:"architecture" binding:"required,oneof=aarch64 x86_64"`
	ImageReference     string   `json:"image_reference" binding:"required"`
	Vcpu               int      `json:"vcpu" binding:"required,gt=0"`
	MemoryMB           int64    `json:"memory_mb" binding:"required,gt=0"`
	NetworkAttachments []string `json:"network_attachments"`
	LaunchMode         string   `json:"launch_mode" binding:"required,oneof=Validate DryRun Execute"`
}

// LaunchValidationResult explains exactly what host-level blockers exist.
type LaunchValidationResult struct {
	IsValid              bool     `json:"is_valid"`
	BlockingReasonCodes  []string `json:"blocking_reason_codes"`
	Warnings             []string `json:"warnings"`
	RequiredHostFeatures []string `json:"required_host_features"`
	MissingHostFeatures  []string `json:"missing_host_features"`
	RecommendedRuntime   string   `json:"recommended_runtime"`
	ExplainabilityText   string   `json:"explainability_text"`
	ValidationTrace      []string `json:"validation_trace"`
}

type LaunchState string

const (
	StatePending    LaunchState = "Pending"
	StatePreparing  LaunchState = "Preparing"
	StateValidated  LaunchState = "Validated"
	StateLaunching  LaunchState = "Launching"
	StateRunning    LaunchState = "Running"
	StateFailed     LaunchState = "Failed"
	StateTerminated LaunchState = "Terminated"
)

type ExecutionTraceStep struct {
	Stage        string `json:"stage"`
	Status       string `json:"status"` // "Success", "Failed"
	ReasonCode   string `json:"reason_code,omitempty"`
	Message      string `json:"message"`
	TimestampSec int64  `json:"timestamp_sec"`
}

type PreparedLaunch struct {
	RuntimeBackend   string   `json:"runtime_backend"`
	BinaryPath       string   `json:"binary_path"`
	ArtifactPath     string   `json:"artifact_path"`
	CommandArgs      []string `json:"command_args"`
	MemoryMB         int64    `json:"memory_mb"`
	Vcpu             int      `json:"vcpu"`
}

type LaunchExecutionRecord struct {
	ExecutionID   string               `json:"execution_id"`
	WorkloadID    string               `json:"workload_id"`
	NodeID        string               `json:"node_id"`
	Spec          LaunchSpec           `json:"spec"`
	PreparedState *PreparedLaunch      `json:"prepared_state,omitempty"`
	State         LaunchState          `json:"state"`
	Trace         []ExecutionTraceStep `json:"trace"`
	PID           *int                 `json:"pid,omitempty"`
}
