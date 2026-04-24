package lifecycle

import (
	"fmt"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime/inspect"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// Reconcile evaluates the current ExecutionRecord against reality and updates its State and Trace
func Reconcile(rec *launch.LaunchExecutionRecord, inspector inspect.Inspector) error {
	now := time.Now().Unix()

	// Terminal states don't need active polling
	if rec.State == launch.StateExited || rec.State == launch.StateFailed || rec.State == launch.StateTerminated {
		return nil
	}

	// We only actively reconcile processes that should exist (Starting, Running, Degraded, Terminating)
	if rec.State != launch.StateStarting && rec.State != launch.StateRunning && rec.State != launch.StateDegraded && rec.State != launch.StateTerminating && rec.State != launch.StateUnknown {
		return nil
	}

	if rec.PID == nil {
		return TransitionTo(rec, launch.StateFailed, "ERR_RECONCILE_PROCESS_MISSING", "Expected PID is missing from record")
	}

	obs, err := inspector.Inspect(*rec.PID)
	if err != nil {
		// If we can't even inspect it, we are blind.
		return TransitionTo(rec, launch.StateUnknown, "ERR_RECONCILE_STATUS_UNREADABLE", err.Error())
	}

	// Update Liveness explicitly
	if obs.ProcessExists {
		rec.RuntimeLiveness = "Alive"
	} else {
		rec.RuntimeLiveness = "Dead"
	}

	// State Machine Evaluation

	// Scenario 1: We are Terminating, and the process is finally gone
	if rec.State == launch.StateTerminating && !obs.ProcessExists {
		termTime := now
		rec.TerminatedAtSec = &termTime
		return TransitionTo(rec, launch.StateTerminated, "", "Process disappeared after termination requested")
	}

	// Scenario 2: The process exited unexpectedly
	if !obs.ProcessExists && rec.State != launch.StateTerminating {
		// If it was just Starting, it's a failure. If it was Running, it's Exited.
		if rec.State == launch.StateStarting {
			return TransitionTo(rec, launch.StateFailed, "ERR_EXEC_RUNTIME_EXITED_EARLY", "Process died before reaching readiness")
		}
		if obs.ExitCode != nil {
			rec.ExitCode = obs.ExitCode
		}
		return TransitionTo(rec, launch.StateExited, "ERR_EXEC_RUNTIME_CRASHED", fmt.Sprintf("Process exited unexpectedly"))
	}

	// Scenario 3: The process exists and we are waiting for Readiness
	if rec.State == launch.StateStarting && obs.ProcessExists {
		if obs.BackendSpecificReady {
			rec.RuntimeReadiness = "Ready"
			readyTime := now
			rec.ReadyAtSec = &readyTime
			return TransitionTo(rec, launch.StateRunning, "", "Runtime reached readiness criteria")
		}

		// Check for readiness timeout (e.g., 30 seconds)
		if rec.StartedAtSec != nil && (now-*rec.StartedAtSec > 30) {
			rec.RuntimeReadiness = "Failed"
			return TransitionTo(rec, launch.StateFailed, "ERR_READY_TIMEOUT", "Process is alive but failed readiness checks within timeout")
		}
	}

	// Scenario 4: It was Running, but backend reported an error (Degraded)
	if rec.State == launch.StateRunning && obs.BackendSpecificError != nil {
		rec.RuntimeReadiness = "Degraded"
		return TransitionTo(rec, launch.StateDegraded, "ERR_READY_PROBE_FAILED", *obs.BackendSpecificError)
	}

	// Scenario 5: It was Degraded, but backend is clean again
	if rec.State == launch.StateDegraded && obs.BackendSpecificError == nil && obs.BackendSpecificReady {
		rec.RuntimeReadiness = "Ready"
		return TransitionTo(rec, launch.StateRunning, "", "Runtime recovered from degraded state")
	}

	// Scenario 6: Recovery from Unknown
	if rec.State == launch.StateUnknown && obs.ProcessExists {
		return TransitionTo(rec, launch.StateRunning, "", "Telemetry restored, process is alive")
	}

	// Update timestamp
	rec.UpdatedAtSec = now
	return nil
}
