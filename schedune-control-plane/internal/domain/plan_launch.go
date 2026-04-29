package domain

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/plan"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
)

// BuildLaunchPlan orchestrates the connection between scheduling and launch spec templating.
func BuildLaunchPlan(
	intent workload.WorkloadIntent,
	template plan.LaunchTemplateSpec,
	mode string,
	targetNodeID string,
	candidateNodes []NodeRecord,
) plan.LaunchPlanResult {

	result := plan.LaunchPlanResult{
		SchemaVersion: "v1alpha1",
		Status:        plan.PlanStatusReady,
		RejectedNodes: []plan.EligibilityEvidence{},
		Warnings:      []string{},
		NextActions:   []plan.LaunchPlanNextAction{},
	}

	// 0. Consistency Check
	if intent.WorkloadID != template.WorkloadID {
		result.Status = plan.PlanStatusConflict
		result.Warnings = append(result.Warnings, "Intent and Template WorkloadID mismatch")
		result.NextActions = append(result.NextActions, plan.ActionNone)
		return result
	}
	if intent.TenantID != template.TenantID {
		result.Status = plan.PlanStatusConflict
		result.Warnings = append(result.Warnings, "Intent and Template TenantID mismatch")
		result.NextActions = append(result.NextActions, plan.ActionNone)
		return result
	}
	if intent.RuntimeClass != template.RuntimeClass {
		result.Status = plan.PlanStatusConflict
		result.Warnings = append(result.Warnings, "Intent and Template RuntimeClass mismatch")
		result.NextActions = append(result.NextActions, plan.ActionNone)
		return result
	}
	if intent.RequiredArchitecture != "any" && intent.RequiredArchitecture != template.Architecture {
		result.Status = plan.PlanStatusConflict
		result.Warnings = append(result.Warnings, "Intent RequiredArchitecture and Template Architecture mismatch")
		result.NextActions = append(result.NextActions, plan.ActionNone)
		return result
	}

	// 1. Run Scheduling
	decision := ScheduleV0(intent, candidateNodes)

	// Map domain rejection to schema rejection to avoid circular dependencies
	for _, rejected := range decision.RejectedNodes {
		hardRejectionCodes := rejected.HardRejectionCodes
		if hardRejectionCodes == nil {
			hardRejectionCodes = []string{}
		}
		matchedFeatures := rejected.MatchedFeatures
		if matchedFeatures == nil {
			matchedFeatures = []string{}
		}
		warnings := rejected.Warnings
		if warnings == nil {
			warnings = []string{}
		}

		result.RejectedNodes = append(result.RejectedNodes, plan.EligibilityEvidence{
			NodeID:             rejected.NodeID,
			IsEligible:         rejected.IsEligible,
			HardRejectionCodes: hardRejectionCodes,
			FreshnessOK:        rejected.FreshnessOK,
			CompatibilityOK:    rejected.CompatibilityOK,
			HealthOK:           rejected.HealthOK,
			MatchedFeatures:    matchedFeatures,
			Warnings:           warnings,
		})
	}

	if decision.SelectedNode == nil {
		result.Status = plan.PlanStatusRejected
		result.Warnings = append(result.Warnings, "No eligible nodes found for workload intent")
		result.NextActions = append(result.NextActions, plan.ActionRetryLater)
		return result
	}

	selectedNode := *decision.SelectedNode

	// 2. Conflict Check
	if targetNodeID != "" && targetNodeID != selectedNode {
		result.Status = plan.PlanStatusConflict
		result.Warnings = append(result.Warnings, "Target node ID conflicts with scheduler decision")
		result.NextActions = append(result.NextActions, plan.ActionNone)
		return result
	}
	if template.NodeID != "" && template.NodeID != selectedNode {
		result.Status = plan.PlanStatusConflict
		result.Warnings = append(result.Warnings, "Launch template node ID conflicts with scheduler decision")
		result.NextActions = append(result.NextActions, plan.ActionNone)
		return result
	}

	result.SelectedNode = &selectedNode

	// Map mode to valid LaunchMode enum
	launchModeEnum := "Validate"
	if mode == "dry_run" {
		launchModeEnum = "DryRun"
	}

	// 3. Template Hydration
	plannedSpec := launch.LaunchSpec{
		SchemaVersion:            template.SchemaVersion,
		WorkloadID:               intent.WorkloadID,
		TenantID:                 intent.TenantID,
		NodeID:                   selectedNode,
		RuntimeClass:             template.RuntimeClass,
		Architecture:             template.Architecture,
		ImageReference:           template.ImageReference,
		KernelImagePath:          template.KernelImagePath,
		RootfsPath:               template.RootfsPath,
		NetworkAttachments:       template.NetworkAttachments,
		Storage:                  template.Storage,
		Networks:                 template.Networks,
		Security:                 template.Security,
		Vcpu:                     template.Vcpu,
		MemoryMB:                 template.MemoryMB,
		LaunchMode:               launchModeEnum,
		RuntimeBackendPreference: template.RuntimeBackendPreference,
		AllowBackendFallback:     template.AllowBackendFallback,
		RuntimeVersion:           template.RuntimeVersion,
	}

	result.PlannedLaunchSpec = &plannedSpec
	result.NextActions = append(result.NextActions, plan.ActionSubmitLaunch)

	return result
}
