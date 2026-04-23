package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
)

func readFixture(t *testing.T, filename string) schema.SchedulerEnvelope {
	path := filepath.Join("..", "..", "..", "testdata", "fixtures", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", filename, err)
	}

	var env schema.SchedulerEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("failed to unmarshal fixture %s: %v", filename, err)
	}

	return env
}

func TestProjectEnvelope_HealthyArmProduction(t *testing.T) {
	env := readFixture(t, "healthy_arm_production.json")
	record := ProjectEnvelope(env)

	if record.ID != "arm-prod-01" {
		t.Errorf("expected ID 'arm-prod-01', got %s", record.ID)
	}
	if record.Identity.Architecture != "aarch64" {
		t.Errorf("expected architecture 'aarch64', got %s", record.Identity.Architecture)
	}
	if record.Compatibility.Class != "ArmProduction" {
		t.Errorf("expected class 'ArmProduction', got %s", record.Compatibility.Class)
	}

	cap, exists := record.Capabilities["kvm_vm_launch"]
	if !exists {
		t.Fatalf("expected capability 'kvm_vm_launch' to exist")
	}
	if cap.State != "Supported" {
		t.Errorf("expected 'Supported', got %s", cap.State)
	}
	if cap.ReasonCode != "KVM_OPENABLE" {
		t.Errorf("expected 'KVM_OPENABLE', got %s", cap.ReasonCode)
	}
}

func TestProjectEnvelope_MissingKvmX86(t *testing.T) {
	env := readFixture(t, "missing_kvm_x86.json")
	record := ProjectEnvelope(env)

	if record.Compatibility.Class != "Unsupported" {
		t.Errorf("expected class 'Unsupported', got %s", record.Compatibility.Class)
	}
	
	if len(record.Constraints) == 0 {
		t.Fatalf("expected constraints to be projected")
	}
	
	constraint, exists := record.Constraints["CONSTRAINT_NO_KVM"]
	if !exists {
		t.Fatalf("expected 'CONSTRAINT_NO_KVM' constraint")
	}
	if constraint.ObservedValue != "missing" {
		t.Errorf("expected observed 'missing', got %s", constraint.ObservedValue)
	}
}

func TestProjectEnvelope_StaleTelemetry(t *testing.T) {
	env := readFixture(t, "stale_telemetry.json")
	record := ProjectEnvelope(env)

	if !record.Freshness.IsStale {
		t.Errorf("expected node to be stale due to collector failure")
	}

	if len(record.Freshness.StaleCollectors) == 0 {
		t.Errorf("expected stale collectors to be listed")
	} else if record.Freshness.StaleCollectors[0] != "MockCollector" && record.Freshness.StaleCollectors[0] != "GlobalEnvelopeStale" {
		t.Errorf("unexpected stale collector: %s", record.Freshness.StaleCollectors[0])
	}
}

func TestProjectEnvelope_HealthyX86KvmOpenable(t *testing.T) {
	env := readFixture(t, "healthy_x86_kvm_openable.json")
	record := ProjectEnvelope(env)

	if record.Compatibility.Class != "X86HoldingPool" {
		t.Errorf("expected class 'X86HoldingPool', got %s", record.Compatibility.Class)
	}
}

func TestProjectEnvelope_KvmExistsNotOpenable(t *testing.T) {
	env := readFixture(t, "kvm_exists_not_openable.json")
	record := ProjectEnvelope(env)

	if record.Compatibility.Class != "Degraded" {
		t.Errorf("expected class 'Degraded', got %s", record.Compatibility.Class)
	}

	cap := record.Capabilities["kvm_vm_launch"]
	if cap.State != "Unavailable" || cap.ReasonCode != "KVM_NOT_OPENABLE_PERMS" {
		t.Errorf("expected Unavailable/KVM_NOT_OPENABLE_PERMS, got %s/%s", cap.State, cap.ReasonCode)
	}
}

func TestProjectEnvelope_HealthyUnsupportedCompatibility(t *testing.T) {
	env := readFixture(t, "healthy_unsupported_compatibility.json")
	record := ProjectEnvelope(env)

	if record.Health.State != "Healthy" {
		t.Errorf("expected HealthState 'Healthy', got %s", record.Health.State)
	}
	
	if record.Compatibility.Class != "Unsupported" {
		t.Errorf("expected CompatibilityClass 'Unsupported', got %s", record.Compatibility.Class)
	}
}
