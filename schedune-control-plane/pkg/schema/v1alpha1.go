package schema

// SchedulerEnvelope is the V1Alpha1 contract emitted by the Schedune Node Agent.
type SchedulerEnvelope struct {
	SchemaVersion     string                      `json:"schema_version" binding:"required,eq=v1alpha1"`
	AgentVersion      string                      `json:"agent_version" binding:"required"`
	CollectionID      string                      `json:"collection_id" binding:"required,uuid"`
	TimestampSec      int64                       `json:"timestamp_sec" binding:"required"`
	NodeID            string                      `json:"node_id" binding:"required"`
	Compatibility     CompatibilityClassification `json:"compatibility" binding:"required"`
	Facts             NodeFacts                   `json:"facts" binding:"required"`
	Capabilities      []NodeCapability            `json:"capabilities" binding:"required"`
	Constraints       []NodeConstraint            `json:"constraints"`
	Health            NodeHealth                  `json:"health" binding:"required"`
	CollectorStatuses []CollectorStatus           `json:"collector_statuses"`
}

type CompatibilityClassification struct {
	Class       string   `json:"class" binding:"required,oneof=ArmProduction X86HoldingPool Unsupported Degraded"`
	ReasonCodes []string `json:"reason_codes"`
}

type NodeFacts struct {
	CPU    CpuFacts    `json:"cpu" binding:"required"`
	Memory MemoryFacts `json:"memory" binding:"required"`
	OS     OsFacts     `json:"os" binding:"required"`
}

type CpuFacts struct {
	Architecture string `json:"architecture" binding:"required"`
	Cores        int    `json:"cores" binding:"required,gt=0"`
	VendorID     string `json:"vendor_id"`
}

type MemoryFacts struct {
	TotalMB int64 `json:"total_mb" binding:"required,gt=0"`
}

type OsFacts struct {
	Hostname      string `json:"hostname" binding:"required"`
	Name          string `json:"name" binding:"required"`
	KernelVersion string `json:"kernel_version"`
}

type NodeCapability struct {
	Feature         string      `json:"feature" binding:"required"`
	State           string      `json:"state" binding:"required,oneof=Supported Unsupported Unknown Unavailable"`
	Provenance      interface{} `json:"provenance" binding:"required"`
	ReasonCode      *string     `json:"reason_code"`
	ObservedAtSec   int64       `json:"observed_at_sec" binding:"required"`
	StaleAfterSec   *int64      `json:"stale_after_sec"`
}

type NodeConstraint struct {
	ConstraintType string  `json:"constraint_type" binding:"required"`
	Code           string  `json:"code" binding:"required"`
	Description    string  `json:"description" binding:"required"`
	ObservedValue  *string `json:"observed_value"`
	ExpectedValue  *string `json:"expected_value"`
}

type NodeHealth struct {
	State        string        `json:"state" binding:"required,oneof=Healthy Warning Degraded Unschedulable Quarantined Unknown"`
	ActiveAlarms []ActiveAlarm `json:"active_alarms"`
}

type ActiveAlarm struct {
	Source          string `json:"source" binding:"required"`
	Severity        string `json:"severity" binding:"required,oneof=Info Warning Error Critical"`
	Code            string `json:"code" binding:"required"`
	Description     string `json:"description" binding:"required"`
	RemediationHint *string `json:"remediation_hint"`
	TimestampSec    int64  `json:"timestamp_sec" binding:"required"`
}

type CollectorStatus struct {
	CollectorName string  `json:"collector_name" binding:"required"`
	Success       bool    `json:"success"`
	DurationMs    int64   `json:"duration_ms"`
	ErrorMessage  *string `json:"error_message"`
}
