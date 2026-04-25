package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain/lifecycle"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/google/uuid"
)

// NodeStore handles finding the node truth
type NodeStore interface {
	GetNode(id string) (NodeRecord, error)
}

// ExecutionStore handles state persistence
type ExecutionStore interface {
	SaveExecution(ctx context.Context, rec launch.LaunchExecutionRecord) error
	GetExecution(ctx context.Context, id string) (launch.LaunchExecutionRecord, bool, error)
}

// ExecutorResolver defines how the orchestrator finds the right backend logic
type ExecutorResolver interface {
	Resolve(backend string) (runtime.Executor, error)
}

// LaunchOrchestrator manages the deterministic pipeline of Validate -> Prepare -> Execute
type LaunchOrchestrator struct {
	nodeStore NodeStore
	execStore ExecutionStore
	resolver  ExecutorResolver
}

func NewLaunchOrchestrator(nodeStore NodeStore, execStore ExecutionStore, resolver ExecutorResolver) *LaunchOrchestrator {
	return &LaunchOrchestrator{nodeStore: nodeStore, execStore: execStore, resolver: resolver}
}

// StartLaunch executes the pipeline synchronously to prove execution boundaries
func (o *LaunchOrchestrator) StartLaunch(spec launch.LaunchSpec) launch.LaunchExecutionRecord {
	rec := o.initializeRecord(spec)
	o.save(&rec)

	if !o.validateAndRecord(spec, &rec) {
		return rec
	}

	exec, ok := o.prepareAndRecord(spec, &rec)
	if !ok {
		return rec
	}

	o.spawnAndRecord(spec, &rec, exec)
	return rec
}

func (o *LaunchOrchestrator) initializeRecord(spec launch.LaunchSpec) launch.LaunchExecutionRecord {
	now := time.Now().Unix()
	rec := launch.LaunchExecutionRecord{
		ExecutionID:      uuid.New().String(),
		WorkloadID:       spec.WorkloadID,
		NodeID:           spec.NodeID,
		Spec:             spec,
		State:            launch.StatePending,
		RuntimeLiveness:  "Unknown",
		RuntimeReadiness: "NotStarted",
		CreatedAtSec:     now,
		UpdatedAtSec:     now,
		Trace:            []launch.ExecutionTraceStep{},
	}
	lifecycle.AppendTrace(&rec, "Init", "Success", "", "Execution initialized")
	return rec
}

func (o *LaunchOrchestrator) validateAndRecord(spec launch.LaunchSpec, rec *launch.LaunchExecutionRecord) bool {
	node, err := o.nodeStore.GetNode(spec.NodeID)
	if err != nil {
		lifecycle.TransitionTo(rec, launch.StatePreparing, "", "Initializing preparation")
		lifecycle.TransitionTo(rec, launch.StateFailed, "ERR_NODE_NOT_FOUND", "Target node not found in store")
		o.save(rec)
		return false
	}

	valRes := ValidateLaunch(spec, node)
	if !valRes.IsValid {
		lifecycle.TransitionTo(rec, launch.StatePreparing, "", "Initializing preparation")
		lifecycle.TransitionTo(rec, launch.StateFailed, "ERR_VALIDATION_FAILED", fmt.Sprintf("Node missing prerequisites: %v", valRes.BlockingReasonCodes))
		o.save(rec)
		return false
	}

	lifecycle.TransitionTo(rec, launch.StatePreparing, "", "Host capabilities validated successfully")
	o.save(rec)
	return true
}

func (o *LaunchOrchestrator) prepareAndRecord(spec launch.LaunchSpec, rec *launch.LaunchExecutionRecord) (runtime.Executor, bool) {
	lifecycle.AppendTrace(rec, "ArtifactResolution", "Pending", "", "Preparing artifact and command execution")

	// We can run ValidateLaunch again or assume the backend is already in ValidationTrace,
	// but it's cleaner to query the backend from the node + spec using the selector.
	node, _ := o.nodeStore.GetNode(spec.NodeID)
	selectedBackend, _, _ := SelectBackend(spec, node)

	exec, err := o.resolver.Resolve(selectedBackend)
	if err != nil {
		lifecycle.TransitionTo(rec, launch.StateFailed, "ERR_PREPARATION_FAILED", fmt.Sprintf("Failed to resolve executor for backend %s: %v", selectedBackend, err))
		o.save(rec)
		return nil, false
	}

	prep, err := exec.Prepare(spec)
	if err != nil {
		lifecycle.TransitionTo(rec, launch.StateFailed, "ERR_PREPARATION_FAILED", err.Error())
		o.save(rec)
		return nil, false
	}
	rec.PreparedState = &prep
	lifecycle.AppendTrace(rec, "ArtifactResolution", "Success", "", "Artifacts resolved and runtime contract generated")

	lifecycle.TransitionTo(rec, launch.StateValidated, "", "Ready for launch")
	o.save(rec)
	return exec, true
}

func (o *LaunchOrchestrator) spawnAndRecord(spec launch.LaunchSpec, rec *launch.LaunchExecutionRecord, exec runtime.Executor) {
	lifecycle.TransitionTo(rec, launch.StateLaunching, "", "Spawning runtime process via executor")
	o.save(rec)

	if spec.LaunchMode == "DryRun" || spec.LaunchMode == "Validate" {
		lifecycle.TransitionTo(rec, launch.StateTerminated, "", "DryRun/Validate successful. Aborting actual spawn.")
		o.save(rec)
		return
	}

	pid, err := exec.Execute(*rec.PreparedState)
	if err != nil {
		lifecycle.TransitionTo(rec, launch.StateFailed, "ERR_EXEC_RUNTIME_SPAWN_FAILED", err.Error())
		o.save(rec)
		return
	}

	rec.PID = &pid
	startedTime := time.Now().Unix()
	rec.StartedAtSec = &startedTime

	lifecycle.TransitionTo(rec, launch.StateStarting, "", fmt.Sprintf("Process successfully spawned (PID %d)", pid))
	o.save(rec)
}

func (o *LaunchOrchestrator) save(rec *launch.LaunchExecutionRecord) {
	o.execStore.SaveExecution(context.Background(), *rec)
}

func (o *LaunchOrchestrator) TerminateLaunch(executionID string) (launch.LaunchExecutionRecord, error) {
	rec, found, err := o.execStore.GetExecution(context.Background(), executionID)
	if err != nil || !found {
		return rec, fmt.Errorf("execution not found or err: %w", err)
	}

	// Idempotency: if already terminated or failed, do nothing
	if rec.State == launch.StateTerminated || rec.State == launch.StateFailed || rec.State == launch.StateExited {
		return rec, nil
	}

	lifecycle.TransitionTo(&rec, launch.StateTerminating, "", "Termination requested by API")
	o.save(&rec)

	if rec.PID != nil && rec.PreparedState != nil {
		exec, err := o.resolver.Resolve(rec.PreparedState.RuntimeBackend)
		if err == nil {
			err = exec.Terminate(*rec.PID)
		}
		if err != nil {
			// Do not jump to Failed immediately; let reconcile loop handle stubborn processes
			lifecycle.AppendTrace(&rec, "Termination", "Failed", "ERR_TERM_SIGNAL_FAILED", err.Error())
			o.save(&rec)
			return rec, err
		}
	}

	termTime := time.Now().Unix()
	rec.TerminatedAtSec = &termTime
	lifecycle.TransitionTo(&rec, launch.StateTerminated, "", "Process terminated successfully")
	o.save(&rec)
	return rec, nil
}
