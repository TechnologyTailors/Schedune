package domain

type OrphanClassification string

const (
	OrphanUnmanagedBackendProcess    OrphanClassification = "UnmanagedBackendProcess"
	OrphanPossiblyManagedButUnlinked OrphanClassification = "PossiblyManagedButUnlinked"
	OrphanStaleExecutionArtifact     OrphanClassification = "StaleExecutionArtifact"
	OrphanUnknownRuntimeProcess      OrphanClassification = "UnknownRuntimeProcess"
)

type OrphanStatus string

const (
	OrphanStatusActive       OrphanStatus = "Active"
	OrphanStatusDisappeared  OrphanStatus = "Disappeared"
	OrphanStatusReclassified OrphanStatus = "Reclassified"
)

type OrphanRecord struct {
	OrphanID           string               `json:"orphan_id"`
	Backend            string               `json:"backend"`
	PID                int                  `json:"pid"`
	PPID               *int                 `json:"ppid,omitempty"`
	Command            string               `json:"command,omitempty"`
	CommandArgs        []string             `json:"command_args,omitempty"`
	CommandFingerprint string               `json:"command_fingerprint"`
	ExecutionIDHint    string               `json:"execution_id_hint,omitempty"`
	WorkloadIDHint     string               `json:"workload_id_hint,omitempty"`
	Classification     OrphanClassification `json:"classification"`
	Status             OrphanStatus         `json:"status"`
	ReasonCode         string               `json:"reason_code"`
	FirstSeenAtSec     int64                `json:"first_seen_at_sec"`
	LastSeenAtSec      int64                `json:"last_seen_at_sec"`
	NodeID             string               `json:"node_id,omitempty"`
	RecoveryEpoch      string               `json:"recovery_epoch,omitempty"`
	Details            map[string]interface{} `json:"details,omitempty"`
}

type OrphanFilter struct {
	Backend        string
	Status         string
	Classification string
}
