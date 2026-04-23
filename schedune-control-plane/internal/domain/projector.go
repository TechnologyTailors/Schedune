package domain

import (
	"time"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
)

// NodeRecord is the canonical, queryable state of a node within the control plane.
type NodeRecord struct {
	ID                 string
	Identity           NodeIdentity
	Health             NodeHealthSummary
	Compatibility      NodeCompatibilityRecord
	Capabilities       map[string]NodeCapabilityRecord
	Constraints        map[string]NodeConstraintRecord
	Freshness          NodeFreshnessRecord
}

type NodeIdentity struct {
	Hostname     string
	Architecture string
	TotalCores   int
	TotalMemoryMB int64
}

type NodeHealthSummary struct {
	State        string
	ActiveAlarms []schema.ActiveAlarm
}

type NodeCompatibilityRecord struct {
	Class       string
	ReasonCodes []string
}

type NodeCapabilityRecord struct {
	State      string
	Provenance string
	ReasonCode string
	IsStale    bool
}

type NodeConstraintRecord struct {
	ConstraintType string
	Description    string
	ObservedValue  string
	ExpectedValue  string
}

type NodeFreshnessRecord struct {
	LastCollectionID  string
	LastCollectionTime time.Time
	IsStale           bool
	StaleCollectors   []string
}

// ProjectEnvelope maps the raw wire schema into the canonical domain model.
func ProjectEnvelope(env schema.SchedulerEnvelope) NodeRecord {
	now := time.Now().Unix()
	
	record := NodeRecord{
		ID: env.NodeID,
		Identity: NodeIdentity{
			Hostname:     env.Facts.OS.Hostname,
			Architecture: env.Facts.CPU.Architecture,
			TotalCores:   env.Facts.CPU.Cores,
			TotalMemoryMB: env.Facts.Memory.TotalMB,
		},
		Health: NodeHealthSummary{
			State:        env.Health.State,
			ActiveAlarms: env.Health.ActiveAlarms,
		},
		Compatibility: NodeCompatibilityRecord{
			Class:       env.Compatibility.Class,
			ReasonCodes: env.Compatibility.ReasonCodes,
		},
		Capabilities: make(map[string]NodeCapabilityRecord),
		Constraints:  make(map[string]NodeConstraintRecord),
		Freshness: NodeFreshnessRecord{
			LastCollectionID:  env.CollectionID,
			LastCollectionTime: time.Unix(env.TimestampSec, 0),
			IsStale:           false, // derived below
			StaleCollectors:   []string{},
		},
	}

	// 1. Process Collector Statuses for Freshness
	// A collector that took too long or failed means that specific telemetry subset is stale/untrusted
	for _, status := range env.CollectorStatuses {
		if !status.Success {
			record.Freshness.StaleCollectors = append(record.Freshness.StaleCollectors, status.CollectorName)
			record.Freshness.IsStale = true
		}
	}

	// 2. Process Capabilities and their individual staleness
	for _, cap := range env.Capabilities {
		isStale := false
		if cap.StaleAfterSec != nil && *cap.StaleAfterSec < now {
			isStale = true
			record.Freshness.IsStale = true
			record.Freshness.StaleCollectors = append(record.Freshness.StaleCollectors, cap.Feature+"_probe")
		}
		
		reason := ""
		if cap.ReasonCode != nil {
			reason = *cap.ReasonCode
		}
		
		// Unpack the provenance interface (this is simplified for the Go backend V1)
		provStr := "Observed"
		if provMap, ok := cap.Provenance.(map[string]interface{}); ok {
			for k, v := range provMap {
				provStr = k + ": " + v.(string)
			}
		} else if pStr, ok := cap.Provenance.(string); ok {
			provStr = pStr
		}

		record.Capabilities[cap.Feature] = NodeCapabilityRecord{
			State:      cap.State,
			Provenance: provStr,
			ReasonCode: reason,
			IsStale:    isStale,
		}
	}

	// 3. Process Constraints
	for _, constraint := range env.Constraints {
		obs := ""
		if constraint.ObservedValue != nil {
			obs = *constraint.ObservedValue
		}
		exp := ""
		if constraint.ExpectedValue != nil {
			exp = *constraint.ExpectedValue
		}
		record.Constraints[constraint.Code] = NodeConstraintRecord{
			ConstraintType: constraint.ConstraintType,
			Description:    constraint.Description,
			ObservedValue:  obs,
			ExpectedValue:  exp,
		}
	}

	// Safety Check: Envelope itself is too old
	if now - env.TimestampSec > 300 {
		record.Freshness.IsStale = true
		record.Freshness.StaleCollectors = append(record.Freshness.StaleCollectors, "GlobalEnvelopeStale")
	}

	return record
}
