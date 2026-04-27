package plan

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
)

type LaunchPlanRequest struct {
	SchemaVersion  string                  `json:"schema_version" binding:"required,eq=v1alpha1"`
	WorkloadIntent workload.WorkloadIntent `json:"workload_intent" binding:"required"`
	LaunchTemplate launch.LaunchSpec       `json:"launch_template" binding:"required"`
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
