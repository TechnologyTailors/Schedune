package api

import (
	"net/http"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type IntakeHandler struct {
	store *store.InMemoryStore
}

func NewIntakeHandler(s *store.InMemoryStore) *IntakeHandler {
	return &IntakeHandler{store: s}
}

// Ingest handles the incoming SchedulerEnvelope from the Schedune Agent
func (h *IntakeHandler) Ingest(c *gin.Context) {
	var env schema.SchedulerEnvelope

	if err := c.ShouldBindJSON(&env); err != nil {
		log.Warn().Err(err).Msg("Schema validation failed during intake")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Schema validation failed", "details": err.Error()})
		return
	}

	record := domain.ProjectEnvelope(env)

	if err := h.store.SaveNodeState(env, record); err != nil {
		log.Error().Err(err).Str("node_id", record.ID).Msg("Failed to save node state")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal storage error"})
		return
	}

	log.Info().
		Str("node_id", record.ID).
		Str("class", record.Compatibility.Class).
		Str("health", record.Health.State).
		Bool("is_stale", record.Freshness.IsStale).
		Msg("Successfully ingested node state")

	c.JSON(http.StatusOK, gin.H{"status": "accepted", "collection_id": env.CollectionID})
}

// ExplainNodeDecision returns the full operational transparency for a node
func (h *IntakeHandler) ExplainNodeDecision(c *gin.Context) {
	nodeID := c.Param("id")
	
	record, err := h.store.GetNode(nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	engine := domain.NewEligibilityEngine(record)

	// Build the detailed operational explanation response
	response := gin.H{
		"identity": gin.H{
			"node_id":      record.ID,
			"hostname":     record.Identity.Hostname,
			"architecture": record.Identity.Architecture,
		},
		"status": gin.H{
			"overall_health": record.Health.State,
			"is_stale":       record.Freshness.IsStale,
		},
		"reasons": record.Compatibility.ReasonCodes,
		"eligibility": gin.H{
			"can_run_kvm":                      engine.CanRunKVMVMs(),
			"can_run_firecracker":              engine.CanRunFirecrackerMicroVMs(),
			"can_join_arm_production_pool":     engine.CanJoinArmProductionPool(),
			"is_only_eligible_for_holding":     engine.IsOnlyEligibleForHoldingPool(),
			"is_healthy_but_policy_ineligible": engine.IsHealthyButPolicyIneligible(),
		},
		"blocking_constraints": record.Constraints,
		"stale_collectors":     record.Freshness.StaleCollectors,
		"remediation_relevant_alarms": record.Health.ActiveAlarms,
	}

	c.JSON(http.StatusOK, response)
}
