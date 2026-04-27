package plan

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
)

type LaunchTemplateSpec struct {
	SchemaVersion string `json:"schema_version" binding:"required,eq=v1alpha1"`
	WorkloadID    string `json:"workload_id" binding:"required"`
	TenantID      string `json:"tenant_id" binding:"required"`
	NodeID        string `json:"node_id,omitempty"`
	RuntimeClass  string `json:"runtime_class" binding:"required,oneof=VirtualMachine MicroVM"`
	Architecture  string `json:"architecture" binding:"required,oneof=aarch64 x86_64"`

	// Legacy fields (Preserved for compatibility)
	ImageReference     string   `json:"image_reference,omitempty"`
	KernelImagePath    string   `json:"kernel_image_path,omitempty"`
	RootfsPath         string   `json:"rootfs_path,omitempty"`
	NetworkAttachments []string `json:"network_attachments,omitempty"`

	// New Strongly Typed Fields
	Storage  []launch.StorageAttachmentSpec `json:"storage,omitempty"`
	Networks []launch.NetworkAttachmentSpec `json:"networks,omitempty"`
	Security *launch.SecurityContextSpec    `json:"security,omitempty"`

	Vcpu                     int                               `json:"vcpu" binding:"required,gt=0"`
	MemoryMB                 int64                             `json:"memory_mb" binding:"required,gt=0"`
	LaunchMode               string                            `json:"launch_mode,omitempty"`
	RuntimeBackendPreference string                            `json:"runtime_backend_preference,omitempty"`
	AllowBackendFallback     bool                              `json:"allow_backend_fallback,omitempty"`
	RuntimeVersion           *launch.RuntimeVersionRequirement `json:"runtime_version,omitempty"`
}

type LaunchPlanRequest struct {
	SchemaVersion  string                  `json:"schema_version" binding:"required,eq=v1alpha1"`
	WorkloadIntent workload.WorkloadIntent `json:"workload_intent" binding:"required"`
	LaunchTemplate LaunchTemplateSpec      `json:"launch_template" binding:"required"`
	Mode           string                  `json:"mode" binding:"required,oneof=validate dry_run"`
	TargetNodeID   string                  `json:"target_node_id,omitempty"`
}

type EligibilityEvidence struct {
	NodeID             string   `json:"node_id"`
	IsEligible         bool     `json:"is_eligible"`
	HardRejectionCodes []string `json:"hard_rejection_codes"`
	FreshnessOK        bool     `json:"freshness_ok"`
	CompatibilityOK    bool     `json:"compatibility_ok"`
	HealthOK           bool     `json:"health_ok"`
	MatchedFeatures    []string `json:"matched_features"`
	Warnings           []string `json:"warnings"`
}

type LaunchPlanStatus string

const (
	PlanStatusReady          LaunchPlanStatus = "Ready"
	PlanStatusRejected       LaunchPlanStatus = "Rejected"
	PlanStatusValidationFail LaunchPlanStatus = "ValidationFail"
	PlanStatusConflict       LaunchPlanStatus = "Conflict"
)

type LaunchPlanNextAction string

const (
	ActionSubmitLaunch  LaunchPlanNextAction = "SubmitLaunch"
	ActionRetryLater    LaunchPlanNextAction = "RetryLater"
	ActionFixValidation LaunchPlanNextAction = "FixValidation"
	ActionNone          LaunchPlanNextAction = "None"
)

type LaunchPlanResult struct {
	SchemaVersion     string                         `json:"schema_version"`
	Status            LaunchPlanStatus               `json:"status"`
	SelectedNode      *string                        `json:"selected_node"`
	RejectedNodes     []EligibilityEvidence          `json:"rejected_nodes"`
	PlannedLaunchSpec *launch.LaunchSpec             `json:"planned_launch_spec,omitempty"`
	ValidationResult  *launch.LaunchValidationResult `json:"validation_result,omitempty"`
	DryRunPrepared    bool                           `json:"dry_run_prepared"`
	Warnings          []string                       `json:"warnings"`
	NextActions       []LaunchPlanNextAction         `json:"next_actions"`
}
