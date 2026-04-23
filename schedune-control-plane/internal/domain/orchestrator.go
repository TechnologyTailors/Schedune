package domain

import (
	"fmt"
	"time"

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

// LaunchOrchestrator manages the deterministic pipeline of Validate -> Prepare -> Execute
type LaunchOrchestrator struct {
	store StoreInterface
	exec  runtime.Executor
}

func NewLaunchOrchestrator(store StoreInterface, exec runtime.Executor) *LaunchOrchestrator {
	return &LaunchOrchestrator{store: store, exec: exec}
}

// StartLaunch executes the pipeline synchronously to prove execution boundaries
func (o *LaunchOrchestrator) StartLaunch(spec launch.LaunchSpec) launch.LaunchExecutionRecord {
	rec := launch.LaunchExecutionRecord{
		ExecutionID: uuid.New().String(),
		WorkloadID:  spec.WorkloadID,
		NodeID:      spec.NodeID,
		Spec:        spec,
		State:       launch.StatePending,
		Trace:       []launch.ExecutionTraceStep{},
	}
	
	addTrace := func(stage, status, code, msg string) {
		rec.Trace = append(rec.Trace, launch.ExecutionTraceStep{
			Stage:        stage,
			Status:       status,
			ReasonCode:   code,
			Message:      msg,
			TimestampSec: time.Now().Unix(),
		})
		o.store.SaveExecution(rec)
	}

	addTrace("Init", "Success", "", "Execution initialized")

	node, err := o.store.GetNode(spec.NodeID)
	if err != nil {
		rec.State = launch.StateFailed
		addTrace("HostPreflight", "Failed", "ERR_NODE_NOT_FOUND", "Target node not found in store")
		return rec
	}

	// 1. Validation Stage
	valRes := ValidateLaunch(spec, node)
	if !valRes.IsValid {
		rec.State = launch.StateFailed
		addTrace("HostPreflight", "Failed", "ERR_VALIDATION_FAILED", fmt.Sprintf("Node missing prerequisites: %v", valRes.BlockingReasonCodes))
		return rec
	}
	rec.State = launch.StateValidated
	addTrace("HostPreflight", "Success", "", "Host capabilities validated successfully")

	// 2. Preparation Stage
	rec.State = launch.StatePreparing
	addTrace("ArtifactResolution", "Pending", "", "Preparing artifact and command execution")
	
	prep, err := o.exec.Prepare(spec)
	if err != nil {
		rec.State = launch.StateFailed
		addTrace("ArtifactResolution", "Failed", "ERR_PREPARATION_FAILED", err.Error())
		return rec
	}
	rec.PreparedState = &prep
	addTrace("ArtifactResolution", "Success", "", "Artifacts resolved and runtime contract generated")

	// 3. Execution Stage
	rec.State = launch.StateLaunching
	addTrace("RuntimeSpawn", "Pending", "", "Spawning runtime process via executor")
	
	pid, err := o.exec.Execute(prep)
	if err != nil {
		rec.State = launch.StateFailed
		addTrace("RuntimeSpawn", "Failed", "ERR_SPAWN_FAILED", err.Error())
		return rec
	}
	
	rec.PID = &pid
	rec.State = launch.StateRunning
	addTrace("RuntimeSpawn", "Success", "", fmt.Sprintf("Process successfully spawned (PID %d)", pid))

	return rec
}

func (o *LaunchOrchestrator) TerminateLaunch(executionID string) (launch.LaunchExecutionRecord, error) {
	rec, err := o.store.GetExecution(executionID)
	if err != nil {
		return rec, err
	}
	
	if rec.State != launch.StateRunning && rec.State != launch.StateLaunching {
		return rec, fmt.Errorf("execution is not in a running state")
	}
	
	if rec.PID != nil {
		err = o.exec.Terminate(*rec.PID)
		if err != nil {
			rec.Trace = append(rec.Trace, launch.ExecutionTraceStep{
				Stage: "Termination", Status: "Failed", ReasonCode: "ERR_TERMINATION_FAILED", Message: err.Error(), TimestampSec: time.Now().Unix(),
			})
			o.store.SaveExecution(rec)
			return rec, err
		}
	}
	
	rec.State = launch.StateTerminated
	rec.Trace = append(rec.Trace, launch.ExecutionTraceStep{
		Stage: "Termination", Status: "Success", ReasonCode: "", Message: "Process terminated successfully", TimestampSec: time.Now().Unix(),
	})
	o.store.SaveExecution(rec)
	return rec, nil
}
