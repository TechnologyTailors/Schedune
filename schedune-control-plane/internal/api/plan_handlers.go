package api

import (
	"net/http"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/plan"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type PlanHandler struct {
	nodeStore *store.InMemoryStore
}

func NewPlanHandler(nodeStore *store.InMemoryStore) *PlanHandler {
	return &PlanHandler{
		nodeStore: nodeStore,
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
		planResult.PreparationResult = &plan.LaunchPreparationResult{
			SchemaVersion:       "v1alpha1",
			Status:              plan.PreparationStatusPendingNodeAgent,
			NodeID:              selectedNodeID,
			Backend:             validationResult.SelectedBackend,
			IsPreparable:        false,
			BlockingReasonCodes: []string{schema.ReasonErrPreparationRequiresNodeAgent},
			Warnings:            []string{"Node-scoped preparation requires an active node agent path"},
		}

		planResult.DryRunPrepared = false // Keep false until distributed prep exists
		planResult.NextActions = []plan.LaunchPlanNextAction{plan.ActionPrepareOnNode}
	}

	c.JSON(http.StatusOK, planResult)
}
