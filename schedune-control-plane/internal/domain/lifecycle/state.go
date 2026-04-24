package lifecycle

import (
	"fmt"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// AllowedTransitions defines the strict state machine rules
var AllowedTransitions = map[launch.LaunchState][]launch.LaunchState{
	launch.StatePending:    {launch.StatePreparing, launch.StateTerminated},
	launch.StatePreparing:  {launch.StateValidated, launch.StateFailed, launch.StateTerminating},
	launch.StateValidated:  {launch.StateLaunching, launch.StateFailed, launch.StateTerminating},
	launch.StateLaunching:  {launch.StateStarting, launch.StateFailed, launch.StateTerminating},
	launch.StateStarting:   {launch.StateRunning, launch.StateExited, launch.StateFailed, launch.StateTerminating},
	launch.StateRunning:    {launch.StateDegraded, launch.StateExited, launch.StateTerminating},
	launch.StateDegraded:   {launch.StateFailed, launch.StateTerminating},
	// Terminal states cannot transition to active states
	launch.StateExited:     {},
	launch.StateFailed:     {},
	launch.StateTerminated: {},
	launch.StateTerminating: {launch.StateTerminated},
}

// Ensure "Unknown" can be transitioned to/from globally according to rules
func IsTransitionAllowed(current, next launch.LaunchState) bool {
	// Any active state can become Unknown if we lose telemetry
	if next == launch.StateUnknown {
		return current != launch.StateTerminated && current != launch.StateFailed && current != launch.StateExited
	}

	// Unknown can recover to specific states based on observation
	if current == launch.StateUnknown {
		return next == launch.StateRunning || next == launch.StateExited || next == launch.StateFailed || next == launch.StateTerminating
	}

	allowed, ok := AllowedTransitions[current]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == next {
			return true
		}
	}
	return false
}

// AppendTrace safely adds an append-only trace step
func AppendTrace(rec *launch.LaunchExecutionRecord, stage, status, code, msg string) {
	rec.Trace = append(rec.Trace, launch.ExecutionTraceStep{
		Stage:        stage,
		Status:       status,
		ReasonCode:   code,
		Message:      msg,
		TimestampSec: time.Now().Unix(),
	})
}

// TransitionTo attempts to move the execution to a new state safely.
func TransitionTo(rec *launch.LaunchExecutionRecord, next launch.LaunchState, reasonCode string, message string) error {
	if !IsTransitionAllowed(rec.State, next) {
		return fmt.Errorf("ERR_STATE_INVALID_TRANSITION: cannot transition from %s to %s", rec.State, next)
	}

	rec.State = next
	AppendTrace(rec, "StateTransition", "Success", reasonCode, message)
	return nil
}
