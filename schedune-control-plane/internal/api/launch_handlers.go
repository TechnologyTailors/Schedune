package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store/sqlite"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type StaticExecutorResolver struct {
	executors map[string]runtime.Executor
}

func (r *StaticExecutorResolver) Resolve(backend string) (runtime.Executor, error) {
	exec, ok := r.executors[backend]
	if !ok {
		return nil, fmt.Errorf("backend %s not registered", backend)
	}
	return exec, nil
}

type LaunchHandler struct {
	nodeStore *store.InMemoryStore
	execStore *sqlite.SQLiteStore
	orch      *domain.LaunchOrchestrator
}

func NewLaunchHandler(nodeStore *store.InMemoryStore, execStore *sqlite.SQLiteStore) *LaunchHandler {
	resolver := &StaticExecutorResolver{
		executors: map[string]runtime.Executor{
			"kvm_qemu":         &runtime.KvmExecutor{},
			"cloud_hypervisor": &runtime.CloudHypervisorExecutor{},
			"firecracker":      &runtime.FirecrackerExecutor{},
		},
	}
	orch := domain.NewLaunchOrchestrator(nodeStore, execStore, resolver)
	return &LaunchHandler{nodeStore: nodeStore, execStore: execStore, orch: orch}
}

// ValidateLaunch assesses whether a chosen node is physically capable of executing the requested spec.
func (h *LaunchHandler) ValidateLaunch(c *gin.Context) {
	var spec launch.LaunchSpec

	if err := c.ShouldBindJSON(&spec); err != nil {
		log.Warn().Err(err).Msg("LaunchSpec validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid LaunchSpec", "details": err.Error()})
		return
	}

	node, err := h.nodeStore.GetNode(spec.NodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target node not found"})
		return
	}

	result := domain.ValidateLaunch(spec, node)
	c.JSON(http.StatusOK, result)
}

// DryRunLaunch simulates execution on the node.
func (h *LaunchHandler) DryRunLaunch(c *gin.Context) {
	var spec launch.LaunchSpec

	if err := c.ShouldBindJSON(&spec); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid LaunchSpec", "details": err.Error()})
		return
	}

	// Override for dryrun
	spec.LaunchMode = "DryRun"

	node, err := h.nodeStore.GetNode(spec.NodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target node not found"})
		return
	}

	result := domain.ValidateLaunch(spec, node)
	c.JSON(http.StatusOK, result)
}

// ExecuteLaunch kicks off the actual orchestrator pipeline
func (h *LaunchHandler) ExecuteLaunch(c *gin.Context) {
	var spec launch.LaunchSpec
	if err := c.ShouldBindJSON(&spec); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid LaunchSpec", "details": err.Error()})
		return
	}
	spec.LaunchMode = "Execute"
	
	record := h.orch.StartLaunch(spec)
	if record.State == launch.StateFailed {
		c.JSON(http.StatusConflict, record)
		return
	}
	c.JSON(http.StatusAccepted, record)
}

// InspectLaunch returns the live status of the execution trace
func (h *LaunchHandler) InspectLaunch(c *gin.Context) {
	id := c.Param("id")
	rec, found, err := h.execStore.GetExecution(context.Background(), id)
	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

// TerminateLaunch kills a running execution
func (h *LaunchHandler) TerminateLaunch(c *gin.Context) {
	id := c.Param("id")
	rec, err := h.orch.TerminateLaunch(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "trace": rec.Trace})
		return
	}
	c.JSON(http.StatusOK, rec)
}
