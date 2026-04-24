package recovery

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

type ReassociationResult struct {
	Matched    bool
	Ambiguous  bool
	Missing    bool
	MatchedPID *int
	ReasonCode string
	Message    string
}

type ReassociationEngine struct{}

func NewReassociationEngine() *ReassociationEngine {
	return &ReassociationEngine{}
}

func (e *ReassociationEngine) ReassociateExecution(rec launch.LaunchExecutionRecord, observed []runtime.ProcessIdentity) ReassociationResult {
	if rec.PID == nil {
		return ReassociationResult{Missing: true, ReasonCode: "ERR_RECOVERY_STALE_HANDLE", Message: "Execution has no PID to reassociate"}
	}

	matches := []int{}
	for _, obs := range observed {
		if obs.PID == *rec.PID {
			// For a fully fleshed out reassociation, we would check CommandFingerprint here.
			// V1 just matches PID
			matches = append(matches, obs.PID)
		}
	}

	if len(matches) == 1 {
		return ReassociationResult{Matched: true, MatchedPID: &matches[0], ReasonCode: "RECOVERY_CONFIRMED", Message: "Exact PID match"}
	} else if len(matches) > 1 {
		return ReassociationResult{Ambiguous: true, ReasonCode: "ERR_RECOVERY_REASSOCIATION_AMBIGUOUS", Message: "Multiple matches for PID"}
	}

	return ReassociationResult{Missing: true, ReasonCode: "ERR_RECOVERY_EXECUTION_MISSING", Message: "PID not found"}
}
