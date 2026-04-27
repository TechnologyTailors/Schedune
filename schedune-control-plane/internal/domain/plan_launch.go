package domain

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/plan"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
)

// BuildLaunchPlan orchestrates the connection between scheduling and launch spec templating.
func BuildLaunchPlan(
	intent workload.WorkloadIntent,
	template launch.LaunchSpec,
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

	result.SelectedNode = &selectedNode

	// 3. Template Hydration
	plannedSpec := template
	plannedSpec.NodeID = selectedNode
	plannedSpec.LaunchMode = mode
	// Ensure these fields from intent overrule template defaults if missing/empty
	plannedSpec.WorkloadID = intent.WorkloadID
	plannedSpec.TenantID = intent.TenantID

	result.PlannedLaunchSpec = &plannedSpec
	result.NextActions = append(result.NextActions, plan.ActionSubmitLaunch)

	return result
}
