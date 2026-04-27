package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain/lifecycle"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime/inspect"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store/sqlite"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

type InspectorResolver interface {
	Resolve(backend string) inspect.Inspector
}

type StaticInspectorResolver struct{}

func (r *StaticInspectorResolver) Resolve(backend string) inspect.Inspector {
	switch backend {
	case "kvm_qemu":
		return &inspect.QemuInspector{}
	case "cloud_hypervisor":
		return &inspect.CloudHypervisorInspector{}
	default:
		return &inspect.ProcessInspector{}
	}
}

type LaunchHandler struct {
	nodeStore         *store.InMemoryStore
	execStore         *sqlite.SQLiteStore
	resolver          *StaticExecutorResolver
	inspectorResolver InspectorResolver
	orch              *domain.LaunchOrchestrator
}

func NewLaunchHandler(nodeStore *store.InMemoryStore, execStore *sqlite.SQLiteStore) *LaunchHandler {
	resolver := &StaticExecutorResolver{
		executors: map[string]runtime.Executor{
			"kvm_qemu":         &runtime.KvmExecutor{},
			"cloud_hypervisor": &runtime.CloudHypervisorExecutor{},
			"firecracker":      &runtime.FirecrackerExecutor{},
		},
	}
	orch := domain.NewLaunchOrchestrator(nodeStore, execStore, execStore, resolver)
	return &LaunchHandler{nodeStore: nodeStore, execStore: execStore, resolver: resolver, inspectorResolver: &StaticInspectorResolver{}, orch: orch}
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

	validationResult := domain.ValidateLaunch(spec, node)
	res := launch.LaunchDryRunResult{
		Validation: validationResult,
	}

	if !validationResult.IsValid {
		c.JSON(http.StatusOK, res)
		return
	}

	exec, err := h.resolver.Resolve(validationResult.SelectedBackend)
	if err != nil {
		errStr := err.Error()
		reason := schema.ReasonErrPreparationFailed
		res.PreparationError = &errStr
		res.PreparationReasonCode = &reason
		c.JSON(http.StatusOK, res)
		return
	}

	prep, err := exec.Prepare(spec)
	if err != nil {
		errStr := err.Error()
		reason := schema.ReasonErrPreparationFailed
		res.PreparationError = &errStr
		res.PreparationReasonCode = &reason
		c.JSON(http.StatusOK, res)
		return
	}

	res.PreparedLaunch = &prep
	c.JSON(http.StatusOK, res)
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

func materialSignalChanged(a, b *launch.ReadinessSignalSummary) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return a.ControlSocketExists != b.ControlSocketExists ||
		a.ControlSocketDialOK != b.ControlSocketDialOK ||
		a.BackendReadySignal != b.BackendReadySignal ||
		a.BackendSignalSource != b.BackendSignalSource ||
		a.StartupGraceElapsed != b.StartupGraceElapsed
}

func (h *LaunchHandler) reconcileExecution(rec *launch.LaunchExecutionRecord) error {
	if rec.State == launch.StateExited || rec.State == launch.StateFailed || rec.State == launch.StateTerminated {
		return nil
	}

	backend := ""
	if rec.PreparedState != nil {
		backend = rec.PreparedState.RuntimeBackend
	}
	inspector := h.inspectorResolver.Resolve(backend)

	// Capture before state
	beforeState := rec.State
	beforeLiveness := rec.RuntimeLiveness
	beforeReadiness := rec.RuntimeReadiness
	var beforeExitCode *int
	if rec.ExitCode != nil {
		ec := *rec.ExitCode
		beforeExitCode = &ec
	}
	var beforeSignal *launch.ReadinessSignalSummary
	if rec.ReadinessSignal != nil {
		sig := *rec.ReadinessSignal
		beforeSignal = &sig
	}

	err := lifecycle.Reconcile(rec, inspector)
	if err != nil {
		log.Error().Err(err).Str("execution_id", rec.ExecutionID).Msg("Failed to reconcile execution")
		return err
	}

	// Determine if material change occurred
	changed := beforeState != rec.State ||
		beforeLiveness != rec.RuntimeLiveness ||
		beforeReadiness != rec.RuntimeReadiness ||
		(beforeExitCode == nil && rec.ExitCode != nil) ||
		(beforeExitCode != nil && rec.ExitCode != nil && *beforeExitCode != *rec.ExitCode) ||
		materialSignalChanged(beforeSignal, rec.ReadinessSignal)

	if changed {
		ev := launch.RuntimeEvent{
			EventID:      uuid.New().String(),
			ExecutionID:  rec.ExecutionID,
			EventType:    launch.EventTypeReconcileStateChanged,
			TimestampSec: time.Now().Unix(),
			ReasonCode:   "",
			PayloadJSON: launch.EventPayloadReconcile{
				ExecutionID:     rec.ExecutionID,
				NodeID:          rec.NodeID,
				BeforeState:     beforeState,
				AfterState:      rec.State,
				BeforeLiveness:  beforeLiveness,
				AfterLiveness:   rec.RuntimeLiveness,
				BeforeReadiness: beforeReadiness,
				AfterReadiness:  rec.RuntimeReadiness,
				ExitCode:        rec.ExitCode,
				Signal:          rec.ReadinessSignal,
			},
		}
		_ = h.execStore.AppendEvent(context.Background(), ev)
	}

	return h.execStore.SaveExecution(context.Background(), *rec)
}

// InspectLaunch returns the live status of the execution trace
func (h *LaunchHandler) InspectLaunch(c *gin.Context) {
	id := c.Param("id")
	rec, found, err := h.execStore.GetExecution(context.Background(), id)
	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	if err := h.reconcileExecution(&rec); err != nil {
		// Logged in reconcileExecution, continue to return current state
	}

	c.JSON(http.StatusOK, rec)
}

// ListLaunches returns a summary of all known executions
func (h *LaunchHandler) ListLaunches(c *gin.Context) {
	records, err := h.execStore.ListExecutions(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list executions"})
		return
	}

	summaries := []launch.ExecutionSummary{}
	for _, rec := range records {
		if err := h.reconcileExecution(&rec); err != nil {
			log.Warn().Err(err).Str("execution_id", rec.ExecutionID).Msg("Failed to reconcile during list")
		}
		summaries = append(summaries, launch.ExecutionSummary{
			ExecutionID:      rec.ExecutionID,
			WorkloadID:       rec.WorkloadID,
			NodeID:           rec.NodeID,
			State:            rec.State,
			RuntimeLiveness:  rec.RuntimeLiveness,
			RuntimeReadiness: rec.RuntimeReadiness,
			CreatedAtSec:     rec.CreatedAtSec,
		})
	}

	c.JSON(http.StatusOK, summaries)
}

// buildReadinessResponse creates a LaunchReadinessResponse from a record
func buildReadinessResponse(rec launch.LaunchExecutionRecord) launch.LaunchReadinessResponse {
	backend := ""
	if rec.PreparedState != nil {
		backend = rec.PreparedState.RuntimeBackend
	}

	sigView := launch.ReadinessSignalView{}
	if rec.ReadinessSignal != nil {
		sigView = launch.ReadinessSignalView{
			ControlSocketPath:   rec.ReadinessSignal.ControlSocketPath,
			ControlSocketExists: rec.ReadinessSignal.ControlSocketExists,
			ControlSocketDialOK: rec.ReadinessSignal.ControlSocketDialOK,
			BackendReadySignal:  rec.ReadinessSignal.BackendReadySignal,
			BackendSignalSource: rec.ReadinessSignal.BackendSignalSource,
		}
	}

	var graceSec int64 = 0
	if rec.PreparedState != nil {
		graceSec = rec.PreparedState.StartupGraceSec
	}

	var deadlineSec int64 = 0
	if rec.StartedAtSec != nil {
		deadlineSec = *rec.StartedAtSec + 15
	}

	var lastObsSec int64 = 0
	if rec.ReadinessSignal != nil {
		lastObsSec = rec.ReadinessSignal.LastObservedAtSec
	}

	timingView := launch.ReadinessTimingView{
		StartupGraceSec:      graceSec,
		StartupGraceElapsed:  rec.ReadinessSignal != nil && rec.ReadinessSignal.StartupGraceElapsed,
		ReadinessTimeoutSec:  15,
		ReadinessDeadlineSec: deadlineSec,
		LastObservedAtSec:    lastObsSec,
	}

	msg := ""
	if rec.State == launch.StateStarting {
		msg = "process alive but readiness signal not yet observed"
	} else if rec.RuntimeReadiness == "Ready" {
		msg = "readiness signal observed"
	} else if rec.RuntimeReadiness == "Failed" {
		msg = "readiness failed or timed out"
	} else if rec.RuntimeReadiness == "Unknown" {
		msg = "process missing during readiness evaluation"
	}

	reasonCode := ""
	if rec.FailureReasonCode != nil {
		reasonCode = *rec.FailureReasonCode
	}

	return launch.LaunchReadinessResponse{
		ExecutionID:      rec.ExecutionID,
		WorkloadID:       rec.WorkloadID,
		NodeID:           rec.NodeID,
		Backend:          backend,
		State:            rec.State,
		RuntimeLiveness:  rec.RuntimeLiveness,
		RuntimeReadiness: rec.RuntimeReadiness,
		ReasonCode:       reasonCode,
		Message:          msg,
		Signal:           sigView,
		Timing:           timingView,
	}
}

// ObserveLaunch provides a comprehensive, read-only view of a specific execution
func (h *LaunchHandler) ObserveLaunch(c *gin.Context) {
	id := c.Param("id")
	rec, found, err := h.execStore.GetExecution(context.Background(), id)
	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	if err := h.reconcileExecution(&rec); err != nil {
		log.Warn().Err(err).Str("execution_id", rec.ExecutionID).Msg("Failed to reconcile during observation")
	}

	events, err := h.execStore.ListEvents(context.Background(), id)
	if err != nil {
		log.Warn().Err(err).Str("execution_id", id).Msg("Failed to list events for observation")
		events = []launch.RuntimeEvent{}
	}

	// Limit to recent 20 events for summary
	if len(events) > 20 {
		events = events[len(events)-20:]
	}

	observation := launch.ExecutionObservation{
		Summary: launch.ExecutionSummary{
			ExecutionID:      rec.ExecutionID,
			WorkloadID:       rec.WorkloadID,
			NodeID:           rec.NodeID,
			State:            rec.State,
			RuntimeLiveness:  rec.RuntimeLiveness,
			RuntimeReadiness: rec.RuntimeReadiness,
			CreatedAtSec:     rec.CreatedAtSec,
		},
		PreparedWith: rec.PreparedState,
		Readiness:    buildReadinessResponse(rec),
		Trace:        rec.Trace,
		RecentEvents: events,
	}

	c.JSON(http.StatusOK, observation)
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

// InspectReadiness returns a focused readiness view
func (h *LaunchHandler) InspectReadiness(c *gin.Context) {
	id := c.Param("id")
	rec, found, err := h.execStore.GetExecution(context.Background(), id)
	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	if err := h.reconcileExecution(&rec); err != nil {
		// Logged in reconcileExecution, continue to return current state
	}

	res := buildReadinessResponse(rec)
	c.JSON(http.StatusOK, res)
}

// InspectTrace returns the append-only lifecycle history
func (h *LaunchHandler) InspectTrace(c *gin.Context) {
	id := c.Param("id")
	rec, found, err := h.execStore.GetExecution(context.Background(), id)
	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}
	c.JSON(http.StatusOK, rec.Trace)
}

// InspectEvents returns operational events related to readiness
func (h *LaunchHandler) InspectEvents(c *gin.Context) {
	id := c.Param("id")
	events, err := h.execStore.ListEvents(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}
