package recovery

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

type MatchResult struct {
	Matched     bool
	Ambiguous   bool
	ExecutionID string
	Reason      string
}

func MatchEnumeratedProcessToExecution(p runtime.EnumeratedProcess, execs []launch.LaunchExecutionRecord) MatchResult {
	var exactMatches []string
	var weakMatches []string

	for _, ex := range execs {
		// 1. Exact Execution Hint Match
		if p.ExecutionIDHint != "" && p.ExecutionIDHint == ex.ExecutionID {
			exactMatches = append(exactMatches, ex.ExecutionID)
			continue
		}

		// 2. Strong Fingerprint Match
		if ex.PreparedState != nil && ex.PreparedState.RuntimeBackend == p.Backend {
			if ex.PID != nil && *ex.PID == p.PID {
				// If we had the prepared command fingerprint we could match it here.
				// For V1, same backend + same PID is a weak/strong match.
				weakMatches = append(weakMatches, ex.ExecutionID)
			}
		}
	}

	if len(exactMatches) == 1 {
		return MatchResult{Matched: true, ExecutionID: exactMatches[0], Reason: "Exact execution marker match"}
	} else if len(exactMatches) > 1 {
		return MatchResult{Ambiguous: true, Reason: "Multiple exact marker matches"}
	}

	if len(weakMatches) == 1 {
		return MatchResult{Matched: true, ExecutionID: weakMatches[0], Reason: "Weak PID/Backend match"}
	} else if len(weakMatches) > 1 {
		return MatchResult{Ambiguous: true, Reason: "Multiple weak PID/Backend matches"}
	}

	return MatchResult{Matched: false, Reason: "No matching execution record"}
}

func ClassifyOrphan(p runtime.EnumeratedProcess, match MatchResult) domain.OrphanRecord {
	var classification domain.OrphanClassification
	var reasonCode string

	if match.Ambiguous {
		classification = domain.OrphanPossiblyManagedButUnlinked
		reasonCode = "ERR_ORPHAN_POSSIBLE_SCHEDUNE_PROCESS"
	} else if p.ExecutionIDHint != "" {
		// Had a hint, but didn't match cleanly -> Stale Artifact
		classification = domain.OrphanStaleExecutionArtifact
		reasonCode = "ERR_ORPHAN_STALE_ARTIFACT_STATE"
	} else {
		// No hint, no match -> Unmanaged
		classification = domain.OrphanUnmanagedBackendProcess
		reasonCode = "ERR_ORPHAN_UNMANAGED_BACKEND_PROCESS"
	}

	// UUID for orphan generated downstream or hash based
	orphanID := "orph-" + p.CommandFingerprint

	return domain.OrphanRecord{
		OrphanID:           orphanID,
		Backend:            p.Backend,
		PID:                p.PID,
		PPID:               p.PPID,
		Command:            p.Command,
		CommandArgs:        p.CommandArgs,
		CommandFingerprint: p.CommandFingerprint,
		ExecutionIDHint:    p.ExecutionIDHint,
		WorkloadIDHint:     p.WorkloadIDHint,
		Classification:     classification,
		Status:             domain.OrphanStatusActive,
		ReasonCode:         reasonCode,
		FirstSeenAtSec:     p.ObservedAtSec,
		LastSeenAtSec:      p.ObservedAtSec,
		Details:            p.Details,
	}
}
