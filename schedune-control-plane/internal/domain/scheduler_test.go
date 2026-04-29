package domain

import (
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
)

func TestScheduleV0_Determinism(t *testing.T) {
	intent := workload.WorkloadIntent{
		WorkloadID:                   "test-wl",
		RuntimeClass:                 "VirtualMachine",
		RequiredArchitecture:         "x86_64",
		RequiredCompatibilityClasses: []string{"kvm_standard"},
		MaxTelemetryAgeSec:           300,
	}

	freshTime := time.Now()

	// Both nodes are eligible and will have the same score.
	// Node B is lexicographically after Node A.
	nodeB := NodeRecord{
		ID:            "node-b",
		Identity:      NodeIdentity{Architecture: "x86_64"},
		Compatibility: NodeCompatibilityRecord{Class: "kvm_standard"},
		Health:        NodeHealthSummary{State: "Healthy", ActiveAlarms: []schema.ActiveAlarm{}}, // +20 score
		Freshness:     NodeFreshnessRecord{LastCollectionTime: freshTime},                        // +30 score
	}

	nodeA := NodeRecord{
		ID:            "node-a",
		Identity:      NodeIdentity{Architecture: "x86_64"},
		Compatibility: NodeCompatibilityRecord{Class: "kvm_standard"},
		Health:        NodeHealthSummary{State: "Healthy", ActiveAlarms: []schema.ActiveAlarm{}}, // +20 score
		Freshness:     NodeFreshnessRecord{LastCollectionTime: freshTime},                        // +30 score
	}

	// Order shouldn't matter, it should always pick node-a.
	candidateNodes1 := []NodeRecord{nodeB, nodeA}
	decision1 := ScheduleV0(intent, candidateNodes1)
	if decision1.SelectedNode == nil {
		t.Fatalf("expected selected node, got nil")
	}
	if *decision1.SelectedNode != "node-a" {
		t.Errorf("expected node-a, got %s", *decision1.SelectedNode)
	}

	candidateNodes2 := []NodeRecord{nodeA, nodeB}
	decision2 := ScheduleV0(intent, candidateNodes2)
	if decision2.SelectedNode == nil {
		t.Fatalf("expected selected node, got nil")
	}
	if *decision2.SelectedNode != "node-a" {
		t.Errorf("expected node-a, got %s", *decision2.SelectedNode)
	}
}
