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

	var prepared launch.PreparedLaunch
	if rec.PreparedState != nil {
		prepared = *rec.PreparedState
	}

	obs, err := inspector.Inspect(rec.ExecutionID, rec.PID, prepared)
	if err != nil {
		return TransitionTo(rec, launch.StateUnknown, "ERR_RECONCILE_STATUS_UNREADABLE", err.Error())
	}

	// Terminating state has special fast-path logic
	if rec.State == launch.StateTerminating && !obs.ProcessExists {
		termTime := now
		rec.TerminatedAtSec = &termTime
		rec.RuntimeLiveness = "Dead"
		return TransitionTo(rec, launch.StateTerminated, "", "Process disappeared after termination requested")
	}

	// Calculate a readiness deadline
	var deadline int64 = 0
	if rec.StartedAtSec != nil {
		deadline = *rec.StartedAtSec + 15 // Hardcoded 15 seconds for V1
	}

	// Call the generic EvaluateBackendReadiness (we'll implement this helper locally if domain isn't imported)
	// For now, let's assume it exists in `domain.EvaluateBackendReadiness`
	// Actually, Reconcile is IN the domain package (subpackage lifecycle). We need to import domain or redefine.
	// We'll just define the transition logic directly here based on obs to avoid circular deps.

	if obs.ProcessExists {
		rec.RuntimeLiveness = "Alive"
	} else {
		rec.RuntimeLiveness = "Dead"
	}

	// Scenario 2: The process exited unexpectedly
	if !obs.ProcessExists && rec.State != launch.StateTerminating {
		// If it was just Starting, it's a failure. If it was Running, it's Exited.
		if rec.State == launch.StateStarting {
			rec.RuntimeReadiness = "Failed"
			return TransitionTo(rec, launch.StateFailed, "ERR_EXEC_RUNTIME_EXITED_EARLY", "Process died before reaching readiness")
		}
		if obs.ExitCode != nil {
			rec.ExitCode = obs.ExitCode
		}
		return TransitionTo(rec, launch.StateExited, "ERR_EXEC_RUNTIME_CRASHED", fmt.Sprintf("Process exited unexpectedly"))
	}

	// Scenario 3: The process exists and we are waiting for Readiness
	if rec.State == launch.StateStarting && obs.ProcessExists {
		if obs.BackendReadySignal {
			rec.RuntimeReadiness = "Ready"
			readyTime := now
			rec.ReadyAtSec = &readyTime
			return TransitionTo(rec, launch.StateRunning, "", "Runtime reached readiness criteria via " + obs.BackendSignalSource)
		}

		// Check for readiness timeout
		if deadline > 0 && now > deadline {
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
	if rec.State == launch.StateDegraded && obs.BackendSpecificError == nil && obs.BackendReadySignal {
		rec.RuntimeReadiness = "Ready"
		return TransitionTo(rec, launch.StateRunning, "", "Runtime recovered from degraded state")
	}

	// Scenario 6: Recovery from Unknown
	if rec.State == launch.StateUnknown && obs.ProcessExists {
		if obs.BackendReadySignal {
			return TransitionTo(rec, launch.StateRunning, "", "Telemetry restored, process is alive and ready")
		} else {
			return TransitionTo(rec, launch.StateStarting, "", "Telemetry restored, process is alive but not ready")
		}
	}

	// Update timestamp
	rec.UpdatedAtSec = now
	return nil
}
