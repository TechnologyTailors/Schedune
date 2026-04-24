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
	o.execStore.SaveExecution(context.Background(), rec)

	node, err := o.nodeStore.GetNode(spec.NodeID)
	if err != nil {
		lifecycle.TransitionTo(&rec, launch.StatePreparing, "", "Initializing preparation")
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_NODE_NOT_FOUND", "Target node not found in store")
		o.execStore.SaveExecution(context.Background(), rec)
		return rec
	}

	// 1. Validation Stage
	valRes := ValidateLaunch(spec, node)
	if !valRes.IsValid {
		lifecycle.TransitionTo(&rec, launch.StatePreparing, "", "Initializing preparation")
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_VALIDATION_FAILED", fmt.Sprintf("Node missing prerequisites: %v", valRes.BlockingReasonCodes))
		o.execStore.SaveExecution(context.Background(), rec)
		return rec
	}
	
	lifecycle.TransitionTo(&rec, launch.StatePreparing, "", "Host capabilities validated successfully")
	o.execStore.SaveExecution(context.Background(), rec)

	// 2. Preparation Stage
	lifecycle.AppendTrace(&rec, "ArtifactResolution", "Pending", "", "Preparing artifact and command execution")
	
	exec, err := o.resolver.Resolve(valRes.SelectedBackend)
	if err != nil {
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_PREPARATION_FAILED", fmt.Sprintf("Failed to resolve executor for backend %s: %v", valRes.SelectedBackend, err))
		o.execStore.SaveExecution(context.Background(), rec)
		return rec
	}
	
	prep, err := exec.Prepare(spec)
	if err != nil {
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_PREPARATION_FAILED", err.Error())
		o.execStore.SaveExecution(context.Background(), rec)
		return rec
	}
	rec.PreparedState = &prep
	lifecycle.AppendTrace(&rec, "ArtifactResolution", "Success", "", "Artifacts resolved and runtime contract generated")

	lifecycle.TransitionTo(&rec, launch.StateValidated, "", "Ready for launch")
	o.execStore.SaveExecution(context.Background(), rec)

	// 3. Execution Stage
	lifecycle.TransitionTo(&rec, launch.StateLaunching, "", "Spawning runtime process via executor")
	o.execStore.SaveExecution(context.Background(), rec)
	
	if spec.LaunchMode == "DryRun" || spec.LaunchMode == "Validate" {
		lifecycle.TransitionTo(&rec, launch.StateTerminated, "", "DryRun/Validate successful. Aborting actual spawn.")
		o.execStore.SaveExecution(context.Background(), rec)
		return rec
	}

	pid, err := exec.Execute(prep)
	if err != nil {
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_EXEC_RUNTIME_SPAWN_FAILED", err.Error())
		o.execStore.SaveExecution(context.Background(), rec)
		return rec
	}
	
	rec.PID = &pid
	startedTime := time.Now().Unix()
	rec.StartedAtSec = &startedTime
	
	lifecycle.TransitionTo(&rec, launch.StateStarting, "", fmt.Sprintf("Process successfully spawned (PID %d)", pid))
	o.execStore.SaveExecution(context.Background(), rec)

	return rec
}

func (o *LaunchOrchestrator) TerminateLaunch(executionID string) (launch.LaunchExecutionRecord, error) {
	rec, found, err := o.execStore.GetExecution(context.Background(), executionID)
	if err != nil || !found {
		return rec, fmt.Errorf("execution not found or err: %v", err)
	}
	
	// Idempotency: if already terminated or failed, do nothing
	if rec.State == launch.StateTerminated || rec.State == launch.StateFailed || rec.State == launch.StateExited {
		return rec, nil
	}

	lifecycle.TransitionTo(&rec, launch.StateTerminating, "", "Termination requested by API")
	o.execStore.SaveExecution(context.Background(), rec)
	
	if rec.PID != nil && rec.PreparedState != nil {
		exec, err := o.resolver.Resolve(rec.PreparedState.RuntimeBackend)
		if err == nil {
			err = exec.Terminate(*rec.PID)
		}
		if err != nil {
			// Do not jump to Failed immediately; let reconcile loop handle stubborn processes
			lifecycle.AppendTrace(&rec, "Termination", "Failed", "ERR_TERM_SIGNAL_FAILED", err.Error())
			o.execStore.SaveExecution(context.Background(), rec)
			return rec, err
		}
	}
	
	termTime := time.Now().Unix()
	rec.TerminatedAtSec = &termTime
	lifecycle.TransitionTo(&rec, launch.StateTerminated, "", "Process terminated successfully")
	o.execStore.SaveExecution(context.Background(), rec)
	return rec, nil
}
