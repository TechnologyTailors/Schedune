package domain

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime/inspect"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

type ReadinessDecision struct {
	Liveness   string
	Readiness  string
	ReasonCode string
	Message    string
}

func EvaluateQemuReadiness(state launch.LaunchState, obs inspect.RuntimeObservation, readinessDeadlineSec int64) ReadinessDecision {
	if !obs.ProcessExists {
		if obs.ExitCode != nil {
			return ReadinessDecision{
				Liveness:   "Dead",
				Readiness:  "Failed",
				ReasonCode: "ERR_EXEC_RUNTIME_EXITED_EARLY",
				Message:    "qemu process exited before readiness",
			}
		}
		return ReadinessDecision{
			Liveness:   "Unknown",
			Readiness:  "Unknown",
			ReasonCode: "ERR_RECONCILE_PROCESS_MISSING",
			Message:    "qemu process missing during readiness evaluation",
		}
	}

	if obs.BackendReadySignal {
		return ReadinessDecision{
			Liveness:  "Alive",
			Readiness: "Ready",
			Message:   "qemu readiness signal observed via " + obs.BackendSignalSource,
		}
	}

	if !obs.StartupGraceElapsed {
		return ReadinessDecision{
			Liveness:  "Alive",
			Readiness: "InProgress",
			Message:   "qemu startup grace window still active",
		}
	}

	if readinessDeadlineSec > 0 && obs.ObservedAtSec > readinessDeadlineSec {
		return ReadinessDecision{
			Liveness:   "Alive",
			Readiness:  "Failed", // State will become Failed
			ReasonCode: "ERR_READY_QEMU_SOCKET_TIMEOUT",
			Message:    "qemu readiness socket did not appear before timeout",
		}
	}

	return ReadinessDecision{
		Liveness:  "Alive",
		Readiness: "InProgress",
		Message:   "qemu process alive but readiness signal not yet observed",
	}
}

func EvaluateCloudHypervisorReadiness(state launch.LaunchState, obs inspect.RuntimeObservation, readinessDeadlineSec int64) ReadinessDecision {
	if !obs.ProcessExists {
		if obs.ExitCode != nil {
			return ReadinessDecision{
				Liveness:   "Dead",
				Readiness:  "Failed",
				ReasonCode: "ERR_EXEC_RUNTIME_EXITED_EARLY",
				Message:    "cloud hypervisor process exited before readiness",
			}
		}
		return ReadinessDecision{
			Liveness:   "Unknown",
			Readiness:  "Unknown",
			ReasonCode: "ERR_RECONCILE_PROCESS_MISSING",
			Message:    "cloud hypervisor process missing during readiness evaluation",
		}
	}

	if obs.BackendReadySignal {
		return ReadinessDecision{
			Liveness:  "Alive",
			Readiness: "Ready",
			Message:   "cloud hypervisor readiness signal observed via " + obs.BackendSignalSource,
		}
	}

	if !obs.StartupGraceElapsed {
		return ReadinessDecision{
			Liveness:  "Alive",
			Readiness: "InProgress",
			Message:   "cloud hypervisor startup grace window still active",
		}
	}

	if readinessDeadlineSec > 0 && obs.ObservedAtSec > readinessDeadlineSec {
		return ReadinessDecision{
			Liveness:   "Alive",
			Readiness:  "Failed",
			ReasonCode: "ERR_READY_CLOUDHYPERVISOR_SOCKET_TIMEOUT",
			Message:    "cloud hypervisor API socket did not appear before timeout",
		}
	}

	return ReadinessDecision{
		Liveness:  "Alive",
		Readiness: "InProgress",
		Message:   "cloud hypervisor process alive but readiness signal not yet observed",
	}
}

func EvaluateBackendReadiness(backend string, state launch.LaunchState, obs inspect.RuntimeObservation, readinessDeadlineSec int64) ReadinessDecision {
	switch backend {
	case "kvm_qemu":
		return EvaluateQemuReadiness(state, obs, readinessDeadlineSec)
	case "cloud_hypervisor":
		return EvaluateCloudHypervisorReadiness(state, obs, readinessDeadlineSec)
	default:
		return ReadinessDecision{
			Liveness:   "Unknown",
			Readiness:  "Unknown",
			ReasonCode: "ERR_READY_BACKEND_UNSUPPORTED",
			Message:    "no readiness evaluator available for backend " + backend,
		}
	}
}
