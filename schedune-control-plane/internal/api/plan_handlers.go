package api

import (
	"net/http"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/plan"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type PlanHandler struct {
	nodeStore *store.InMemoryStore
	resolver  *StaticExecutorResolver
}

func NewPlanHandler(nodeStore *store.InMemoryStore) *PlanHandler {
	resolver := &StaticExecutorResolver{
		executors: map[string]runtime.Executor{
			"kvm_qemu":         &runtime.KvmExecutor{},
			"cloud_hypervisor": &runtime.CloudHypervisorExecutor{},
			"firecracker":      &runtime.FirecrackerExecutor{},
		},
	}
	return &PlanHandler{
		nodeStore: nodeStore,
		resolver:  resolver,
	}
}

// PlanLaunch acts as the bridge connecting scheduling to a concrete launch without executing.
func (h *PlanHandler) PlanLaunch(c *gin.Context) {
	var req plan.LaunchPlanRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("LaunchPlanRequest validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid LaunchPlanRequest", "details": err.Error()})
		return
	}

	allNodes := h.nodeStore.ListAllNodes()

	// 1. Build Pure Domain Plan
	planResult := domain.BuildLaunchPlan(
		req.WorkloadIntent,
		req.LaunchTemplate,
		req.Mode,
		req.TargetNodeID,
		allNodes,
	)

	// If the plan is rejected or there's a conflict, return early.
	if planResult.Status != plan.PlanStatusReady || planResult.PlannedLaunchSpec == nil {
		c.JSON(http.StatusOK, planResult)
		return
	}

	selectedNodeID := *planResult.SelectedNode
	node, err := h.nodeStore.GetNode(selectedNodeID)
	if err != nil {
		planResult.Status = plan.PlanStatusRejected
		planResult.Warnings = append(planResult.Warnings, "Selected node disappeared from store before validation")
		planResult.NextActions = []plan.LaunchPlanNextAction{plan.ActionRetryLater}
		c.JSON(http.StatusOK, planResult)
		return
	}

	// 2. validate Launch on Selected Node
	validationResult := domain.ValidateLaunch(*planResult.PlannedLaunchSpec, node)
	planResult.ValidationResult = &validationResult

	if !validationResult.IsValid {
		planResult.Status = plan.PlanStatusValidationFail
		planResult.NextActions = []plan.LaunchPlanNextAction{plan.ActionFixValidation}
		c.JSON(http.StatusOK, planResult)
		return
	}

	// 3. Dry Run Prepared Evidence (if requested)
	if req.Mode == "dry_run" {
		exec, err := h.resolver.Resolve(validationResult.SelectedBackend)
		if err != nil {
			planResult.Status = plan.PlanStatusValidationFail
			planResult.Warnings = append(planResult.Warnings, "Failed to resolve executor for dry run: "+err.Error())
			planResult.NextActions = []plan.LaunchPlanNextAction{plan.ActionFixValidation}
			c.JSON(http.StatusOK, planResult)
			return
		}

		_, err = exec.Prepare(*planResult.PlannedLaunchSpec)
		if err != nil {
			planResult.Status = plan.PlanStatusValidationFail
			reason := schema.ReasonErrPreparationFailed
			planResult.Warnings = append(planResult.Warnings, "Dry run preparation failed: "+err.Error())
			// Populate a backend rejection for dry run error
			planResult.ValidationResult.IsValid = false
			planResult.ValidationResult.BlockingReasonCodes = append(planResult.ValidationResult.BlockingReasonCodes, reason)
			planResult.NextActions = []plan.LaunchPlanNextAction{plan.ActionFixValidation}
			c.JSON(http.StatusOK, planResult)
			return
		}

		planResult.DryRunPrepared = true
	}

	c.JSON(http.StatusOK, planResult)
}
