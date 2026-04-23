package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

func TestValidateLaunch_HealthyArmProduction(t *testing.T) {
	env := readFixture(t, "healthy_arm_production.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion:  "v1alpha1",
		WorkloadID:     "wl-launch-001",
		TenantID:       "tenant-1",
		NodeID:         node.ID,
		RuntimeClass:   "VirtualMachine",
		Architecture:   "aarch64",
		ImageReference: "oci://registry/img",
		Vcpu:           4,
		MemoryMB:       8192,
		LaunchMode:     "DryRun",
	}

	result := ValidateLaunch(spec, node)

	if !result.IsValid {
		t.Errorf("expected launch to be valid, got false: %v", result.BlockingReasonCodes)
	}

	if result.RecommendedRuntime != "qemu-system-aarch64" {
		t.Errorf("expected 'qemu-system-aarch64', got %s", result.RecommendedRuntime)
	}
}

func TestValidateLaunch_MissingKvmX86(t *testing.T) {
	env := readFixture(t, "missing_kvm_x86.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion:  "v1alpha1",
		WorkloadID:     "wl-launch-002",
		TenantID:       "tenant-1",
		NodeID:         node.ID,
		RuntimeClass:   "VirtualMachine",
		Architecture:   "x86_64", // Match arch to focus on KVM blocker
		ImageReference: "oci://registry/img",
		Vcpu:           2,
		MemoryMB:       4096,
		LaunchMode:     "DryRun",
	}

	result := ValidateLaunch(spec, node)

	if result.IsValid {
		t.Errorf("expected launch to be invalid")
	}

	hasKvmBlocker := false
	for _, code := range result.BlockingReasonCodes {
		if code == "ERR_MISSING_RUNTIME_CAPABILITY_kvm_vm_launch" {
			hasKvmBlocker = true
		}
	}

	if !hasKvmBlocker {
		t.Errorf("expected ERR_MISSING_RUNTIME_CAPABILITY_kvm_vm_launch blocker, got %v", result.BlockingReasonCodes)
	}
}

func TestValidateLaunch_ArchitectureMismatch(t *testing.T) {
	env := readFixture(t, "missing_kvm_x86.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion:  "v1alpha1",
		WorkloadID:     "wl-launch-003",
		TenantID:       "tenant-1",
		NodeID:         node.ID,
		RuntimeClass:   "Container",
		Architecture:   "aarch64", // Spec asks for aarch64, Node is x86_64
		ImageReference: "oci://registry/img",
		Vcpu:           2,
		MemoryMB:       4096,
		LaunchMode:     "DryRun",
	}

	result := ValidateLaunch(spec, node)

	if result.IsValid {
		t.Errorf("expected launch to be invalid")
	}

	hasArchBlocker := false
	for _, code := range result.BlockingReasonCodes {
		if code == "ERR_LAUNCH_ARCH_MISMATCH" {
			hasArchBlocker = true
		}
	}

	if !hasArchBlocker {
		t.Errorf("expected ERR_LAUNCH_ARCH_MISMATCH blocker, got %v", result.BlockingReasonCodes)
	}
}

func TestValidateLaunch_FirecrackerPartialFail(t *testing.T) {
	env := readFixture(t, "firecracker_partial_fail.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion:  "v1alpha1",
		WorkloadID:     "wl-launch-004",
		TenantID:       "tenant-1",
		NodeID:         node.ID,
		RuntimeClass:   "MicroVM",
		Architecture:   "aarch64",
		ImageReference: "oci://registry/img",
		Vcpu:           2,
		MemoryMB:       2048,
		LaunchMode:     "DryRun",
	}

	result := ValidateLaunch(spec, node)

	if result.IsValid {
		t.Errorf("expected launch to be invalid due to missing firecracker_launch capability")
	}

	hasFcBlocker := false
	for _, code := range result.BlockingReasonCodes {
		if code == "ERR_MISSING_RUNTIME_CAPABILITY_firecracker_launch" {
			hasFcBlocker = true
		}
	}

	if !hasFcBlocker {
		t.Errorf("expected ERR_MISSING_RUNTIME_CAPABILITY_firecracker_launch blocker, got %v", result.BlockingReasonCodes)
	}
}

func TestValidateLaunch_KvmExistsNotOpenable(t *testing.T) {
	env := readFixture(t, "kvm_exists_not_openable.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion:  "v1alpha1",
		WorkloadID:     "wl-launch-005",
		TenantID:       "tenant-1",
		NodeID:         node.ID,
		RuntimeClass:   "VirtualMachine",
		Architecture:   "x86_64",
		ImageReference: "oci://registry/img",
		Vcpu:           2,
		MemoryMB:       2048,
		LaunchMode:     "DryRun",
	}

	result := ValidateLaunch(spec, node)

	if result.IsValid {
		t.Errorf("expected launch to be invalid due to unopenable kvm")
	}

	hasKvmBlocker := false
	for _, code := range result.BlockingReasonCodes {
		if code == "ERR_MISSING_RUNTIME_CAPABILITY_kvm_vm_launch" {
			hasKvmBlocker = true
		}
	}

	if !hasKvmBlocker {
		t.Errorf("expected ERR_MISSING_RUNTIME_CAPABILITY_kvm_vm_launch blocker, got %v", result.BlockingReasonCodes)
	}
	
	// Optional: verify that the reason trace exposes the exact reason from the agent
	traceStr := ""
	for _, tr := range result.ValidationTrace {
		traceStr += tr + " "
	}
	if !strings.Contains(traceStr, "KVM_NOT_OPENABLE_PERMS") {
		t.Errorf("expected trace to contain KVM_NOT_OPENABLE_PERMS, got %s", traceStr)
	}
}
