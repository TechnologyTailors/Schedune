package domain

type OrphanClassification string

const (
	OrphanUnmanagedBackendProcess    OrphanClassification = "UnmanagedBackendProcess"
	OrphanPossiblyManagedButUnlinked OrphanClassification = "PossiblyManagedButUnlinked"
	OrphanStaleExecutionArtifact     OrphanClassification = "StaleExecutionArtifact"
	OrphanUnknownRuntimeProcess      OrphanClassification = "UnknownRuntimeProcess"
)

type OrphanRecord struct {
	OrphanID           string               `json:"orphan_id"`
	Backend            string               `json:"backend"`
	PID                int                  `json:"pid"`
	CommandFingerprint string               `json:"command_fingerprint"`
	Classification     OrphanClassification `json:"classification"`
	ReasonCode         string               `json:"reason_code"`
	ObservedAtSec      int64                `json:"observed_at_sec"`
}
