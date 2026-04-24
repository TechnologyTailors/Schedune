package domain

import (
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
)

func TestEvaluate_HealthyArmProduction(t *testing.T) {
	env := readFixture(t, "healthy_arm_production.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	intent := workload.WorkloadIntent{
		SchemaVersion:                "v1alpha1",
		WorkloadID:                   "wl-arm-001",
		TenantID:                     "tenant-1",
		RuntimeClass:                 "VirtualMachine",
		RequiredArchitecture:         "aarch64",
		MaxTelemetryAgeSec:           60,
		RequiresKVM:                  true,
		RequiresTPM:                  true,
		RequiredCompatibilityClasses: []string{"ArmProduction"},
	}

	result := Evaluate(intent, node)

	if !result.IsEligible {
		t.Errorf("expected node to be eligible, but got rejections: %v", result.HardRejectionCodes)
	}
}

func TestEvaluate_ArchitectureMismatch(t *testing.T) {
	env := readFixture(t, "missing_kvm_x86.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	node := ProjectEnvelope(env)

	intent := workload.WorkloadIntent{
		SchemaVersion:                "v1alpha1",
		WorkloadID:                   "wl-arm-002",
		TenantID:                     "tenant-1",
		RuntimeClass:                 "VirtualMachine",
		RequiredArchitecture:         "aarch64", // Node is x86_64
		MaxTelemetryAgeSec:           60,
		RequiredCompatibilityClasses: []string{"ArmProduction"},
	}

	result := Evaluate(intent, node)

	if result.IsEligible {
		t.Errorf("expected node to be ineligible")
	}

	hasArchReject := false
	for _, code := range result.HardRejectionCodes {
		if code == "REJECT_ARCHITECTURE_MISMATCH" {
			hasArchReject = true
		}
	}

	if !hasArchReject {
		t.Errorf("expected REJECT_ARCHITECTURE_MISMATCH")
	}
}

func TestEvaluate_MissingKVM(t *testing.T) {
	env := readFixture(t, "missing_kvm_x86.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	node := ProjectEnvelope(env)

	intent := workload.WorkloadIntent{
		SchemaVersion:                "v1alpha1",
		WorkloadID:                   "wl-x86-001",
		TenantID:                     "tenant-1",
		RuntimeClass:                 "VirtualMachine",
		RequiredArchitecture:         "x86_64",
		MaxTelemetryAgeSec:           60,
		RequiresKVM:                  true, // Node is missing KVM
		RequiredCompatibilityClasses: []string{"Unsupported"},
	}

	result := Evaluate(intent, node)

	if result.IsEligible {
		t.Errorf("expected node to be ineligible")
	}

	hasKvmReject := false
	for _, code := range result.HardRejectionCodes {
		if code == "REJECT_MISSING_KVM" {
			hasKvmReject = true
		}
	}

	if !hasKvmReject {
		t.Errorf("expected REJECT_MISSING_KVM, got %v", result.HardRejectionCodes)
	}
}

func TestEvaluate_StaleTelemetry(t *testing.T) {
	env := readFixture(t, "stale_telemetry.json")
	node := ProjectEnvelope(env)

	intent := workload.WorkloadIntent{
		SchemaVersion:                "v1alpha1",
		WorkloadID:                   "wl-stale-001",
		TenantID:                     "tenant-1",
		RuntimeClass:                 "Container",
		RequiredArchitecture:         "any",
		MaxTelemetryAgeSec:           60,
		RequiredCompatibilityClasses: []string{"ArmProduction"},
	}

	result := Evaluate(intent, node)

	if result.IsEligible {
		t.Errorf("expected node to be ineligible due to stale telemetry")
	}

	hasStaleReject := false
	for _, code := range result.HardRejectionCodes {
		if code == "REJECT_TELEMETRY_STALE" {
			hasStaleReject = true
		}
	}

	if !hasStaleReject {
		t.Errorf("expected REJECT_TELEMETRY_STALE, got %v", result.HardRejectionCodes)
	}
}

func TestEvaluate_FirecrackerPartialFail(t *testing.T) {
	env := readFixture(t, "firecracker_partial_fail.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	intent := workload.WorkloadIntent{
		SchemaVersion:                "v1alpha1",
		WorkloadID:                   "wl-fc-001",
		TenantID:                     "tenant-1",
		RuntimeClass:                 "MicroVM",
		RequiredArchitecture:         "aarch64",
		MaxTelemetryAgeSec:           60,
		RequiresKVM:                  true, // It has KVM, but Firecracker launch validator will fail it
		RequiredCompatibilityClasses: []string{"ArmProduction"},
	}

	result := Evaluate(intent, node)

	// In the evaluator, RequiresKVM passes, but does Evaluate know it's a MicroVM requiring firecracker_launch?
	// Currently Evaluate only checks RequiresKVM and RequiresTPM explicitly.
	// The runtime logic is in ValidateLaunch.
	// So it might be "eligible" for scheduling, but fail launch validation.
	if !result.IsEligible {
		t.Errorf("expected node to be eligible for scheduling (LaunchValidation will catch the FC failure)")
	}
}
