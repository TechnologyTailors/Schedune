package domain

import (
	"fmt"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain/lifecycle"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/google/uuid"
)

// StoreInterface avoids circular dependencies with the concrete store package
type StoreInterface interface {
	GetNode(id string) (NodeRecord, error)
	SaveExecution(rec launch.LaunchExecutionRecord)
	GetExecution(id string) (launch.LaunchExecutionRecord, error)
}

// ExecutorResolver defines how the orchestrator finds the right backend logic
type ExecutorResolver interface {
	Resolve(backend string) (runtime.Executor, error)
}

// LaunchOrchestrator manages the deterministic pipeline of Validate -> Prepare -> Execute
type LaunchOrchestrator struct {
	store    StoreInterface
	resolver ExecutorResolver
}

func NewLaunchOrchestrator(store StoreInterface, resolver ExecutorResolver) *LaunchOrchestrator {
	return &LaunchOrchestrator{store: store, resolver: resolver}
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
	o.store.SaveExecution(rec)

	node, err := o.store.GetNode(spec.NodeID)
	if err != nil {
		lifecycle.TransitionTo(&rec, launch.StatePreparing, "", "Initializing preparation")
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_NODE_NOT_FOUND", "Target node not found in store")
		o.store.SaveExecution(rec)
		return rec
	}

	// 1. Validation Stage
	valRes := ValidateLaunch(spec, node)
	if !valRes.IsValid {
		lifecycle.TransitionTo(&rec, launch.StatePreparing, "", "Initializing preparation")
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_VALIDATION_FAILED", fmt.Sprintf("Node missing prerequisites: %v", valRes.BlockingReasonCodes))
		o.store.SaveExecution(rec)
		return rec
	}
	
	lifecycle.TransitionTo(&rec, launch.StatePreparing, "", "Host capabilities validated successfully")
	o.store.SaveExecution(rec)

	// 2. Preparation Stage
	lifecycle.AppendTrace(&rec, "ArtifactResolution", "Pending", "", "Preparing artifact and command execution")
	
	exec, err := o.resolver.Resolve(valRes.SelectedBackend)
	if err != nil {
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_PREPARATION_FAILED", fmt.Sprintf("Failed to resolve executor for backend %s: %v", valRes.SelectedBackend, err))
		o.store.SaveExecution(rec)
		return rec
	}
	
	prep, err := exec.Prepare(spec)
	if err != nil {
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_PREPARATION_FAILED", err.Error())
		o.store.SaveExecution(rec)
		return rec
	}
	rec.PreparedState = &prep
	lifecycle.AppendTrace(&rec, "ArtifactResolution", "Success", "", "Artifacts resolved and runtime contract generated")

	lifecycle.TransitionTo(&rec, launch.StateValidated, "", "Ready for launch")
	o.store.SaveExecution(rec)

	// 3. Execution Stage
	lifecycle.TransitionTo(&rec, launch.StateLaunching, "", "Spawning runtime process via executor")
	o.store.SaveExecution(rec)
	
	if spec.LaunchMode == "DryRun" || spec.LaunchMode == "Validate" {
		lifecycle.TransitionTo(&rec, launch.StateTerminated, "", "DryRun/Validate successful. Aborting actual spawn.")
		o.store.SaveExecution(rec)
		return rec
	}

	pid, err := exec.Execute(prep)
	if err != nil {
		lifecycle.TransitionTo(&rec, launch.StateFailed, "ERR_EXEC_RUNTIME_SPAWN_FAILED", err.Error())
		o.store.SaveExecution(rec)
		return rec
	}
	
	rec.PID = &pid
	startedTime := time.Now().Unix()
	rec.StartedAtSec = &startedTime
	
	lifecycle.TransitionTo(&rec, launch.StateStarting, "", fmt.Sprintf("Process successfully spawned (PID %d)", pid))
	o.store.SaveExecution(rec)

	return rec
}

func (o *LaunchOrchestrator) TerminateLaunch(executionID string) (launch.LaunchExecutionRecord, error) {
	rec, err := o.store.GetExecution(executionID)
	if err != nil {
		return rec, err
	}
	
	// Idempotency: if already terminated or failed, do nothing
	if rec.State == launch.StateTerminated || rec.State == launch.StateFailed || rec.State == launch.StateExited {
		return rec, nil
	}

	lifecycle.TransitionTo(&rec, launch.StateTerminating, "", "Termination requested by API")
	o.store.SaveExecution(rec)
	
	if rec.PID != nil && rec.PreparedState != nil {
		exec, err := o.resolver.Resolve(rec.PreparedState.RuntimeBackend)
		if err == nil {
			err = exec.Terminate(*rec.PID)
		}
		if err != nil {
			// Do not jump to Failed immediately; let reconcile loop handle stubborn processes
			lifecycle.AppendTrace(&rec, "Termination", "Failed", "ERR_TERM_SIGNAL_FAILED", err.Error())
			o.store.SaveExecution(rec)
			return rec, err
		}
	}
	
	termTime := time.Now().Unix()
	rec.TerminatedAtSec = &termTime
	lifecycle.TransitionTo(&rec, launch.StateTerminated, "", "Process terminated successfully")
	o.store.SaveExecution(rec)
	return rec, nil
}
