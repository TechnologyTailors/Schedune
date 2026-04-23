package api

import (
	"net/http"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type SchedulerHandler struct {
	store *store.InMemoryStore
}

func NewSchedulerHandler(s *store.InMemoryStore) *SchedulerHandler {
	return &SchedulerHandler{store: s}
}

// ExplainSchedule answers "If I submit this workload, where will it go and why?" without placing it.
func (h *SchedulerHandler) ExplainSchedule(c *gin.Context) {
	var intent workload.WorkloadIntent

	if err := c.ShouldBindJSON(&intent); err != nil {
		log.Warn().Err(err).Msg("Schema validation failed during schedule explain")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid WorkloadIntent", "details": err.Error()})
		return
	}

	// For v0, get all known nodes (In a real system, we'd pre-filter by tenant boundaries or zones)
	// We'll add a helper to store to just return all nodes for scoring.
	allNodes := h.store.ListAllNodes()

	decision := domain.ScheduleV0(intent, allNodes)

	c.JSON(http.StatusOK, decision)
}

// SelectNode actually executes the scheduler (Mocking placement for now)
func (h *SchedulerHandler) SelectNode(c *gin.Context) {
	var intent workload.WorkloadIntent

	if err := c.ShouldBindJSON(&intent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid WorkloadIntent", "details": err.Error()})
		return
	}

	allNodes := h.store.ListAllNodes()
	decision := domain.ScheduleV0(intent, allNodes)

	if decision.SelectedNode == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "No eligible nodes available for workload",
			"decision_trace": decision,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "scheduled",
		"node_id": *decision.SelectedNode,
		"workload_id": intent.WorkloadID,
	})
}
