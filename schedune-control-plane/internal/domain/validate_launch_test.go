package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
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

func TestValidateLaunch_CloudHypervisorReady(t *testing.T) {
	env := readFixture(t, "cloudhypervisor_ready_arm.json")
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
		WorkloadID:     "wl-launch-ch",
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

	if result.SelectedBackend != "cloud_hypervisor" {
		t.Errorf("expected selected backend to be cloud_hypervisor, got %s", result.SelectedBackend)
	}

	if result.RecommendedRuntime != "cloud-hypervisor" {
		t.Errorf("expected 'cloud-hypervisor', got %s", result.RecommendedRuntime)
	}
}

func TestValidateLaunch_FirecrackerHostReadyArtifactInvalid(t *testing.T) {
	env := readFixture(t, "firecracker_host_ready.json")
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
		WorkloadID:     "wl-launch-fc",
		TenantID:       "tenant-1",
		NodeID:         node.ID,
		RuntimeClass:   "MicroVM",
		Architecture:   "x86_64",
		ImageReference: "oci://registry/img", // Invalid for FC which needs kernel/rootfs in V0
		Vcpu:           2,
		MemoryMB:       2048,
		LaunchMode:     "DryRun",
	}

	result := ValidateLaunch(spec, node)

	if result.IsValid {
		t.Errorf("expected launch to be invalid due to missing artifact model")
	}

	hasArtifactBlocker := false
	for _, tr := range result.ValidationTrace {
		if strings.Contains(tr, schema.ReasonErrLaunchInvalidFirecrackerArtifactModel) {
			hasArtifactBlocker = true
		}
	}

	if !hasArtifactBlocker {
		t.Errorf("expected ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL in trace, got %v", result.ValidationTrace)
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
		if code == schema.ReasonErrLaunchBackendNotSupported {
			hasKvmBlocker = true
		}
	}

	if !hasKvmBlocker {
		t.Errorf("expected ERR_LAUNCH_BACKEND_NOT_SUPPORTED blocker, got %v", result.BlockingReasonCodes)
	}

	if result.RejectedBackends["kvm_qemu"] != schema.ReasonErrLaunchMissingCapabilityKvmQemu+" ("+schema.ReasonCapKvmMissing+")" {
		t.Errorf("expected rejected backend kvm_qemu with CAP_KVM_MISSING, got %v", result.RejectedBackends)
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
		if code == schema.ReasonErrLaunchArchMismatch {
			hasArchBlocker = true
		}
	}

	if !hasArchBlocker {
		t.Errorf("expected ERR_LAUNCH_ARCH_MISMATCH blocker, got %v", result.BlockingReasonCodes)
	}

	if hint, ok := result.RemediationHints["architecture"]; !ok || !strings.Contains(hint, "matches the node") {
		t.Errorf("expected remediation hint for architecture, got %v", result.RemediationHints)
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
		if code == schema.ReasonErrLaunchBackendNotSupported {
			hasFcBlocker = true
		}
	}

	if !hasFcBlocker {
		t.Errorf("expected ERR_LAUNCH_BACKEND_NOT_SUPPORTED blocker, got %v", result.BlockingReasonCodes)
	}

	if result.RejectedBackends["firecracker"] != schema.ReasonErrLaunchMissingCapabilityFcBinary {
		t.Errorf("expected rejected backend firecracker, got %v", result.RejectedBackends)
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
		if code == schema.ReasonErrLaunchBackendNotSupported {
			hasKvmBlocker = true
		}
	}

	if !hasKvmBlocker {
		t.Errorf("expected ERR_LAUNCH_BACKEND_NOT_SUPPORTED blocker, got %v", result.BlockingReasonCodes)
	}

	if result.RejectedBackends["kvm_qemu"] != schema.ReasonErrLaunchMissingCapabilityKvmQemu+" ("+schema.ReasonCapKvmNotOpenablePerms+")" {
		t.Errorf("expected rejected backend kvm_qemu, got %v", result.RejectedBackends)
	}

	// Optional: verify that the reason trace exposes the exact reason from the agent
	traceStr := ""
	for _, tr := range result.ValidationTrace {
		traceStr += tr + " "
	}
	if !strings.Contains(traceStr, "CAP_KVM_NOT_OPENABLE_PERMS") {
		t.Errorf("expected trace to contain CAP_KVM_NOT_OPENABLE_PERMS, got %s", traceStr)
	}
}

func TestValidateLaunch_MissingQemuBinary(t *testing.T) {
	env := readFixture(t, "missing_qemu_binary.json")
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
		WorkloadID:     "wl-launch-006",
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
		t.Errorf("expected launch to be invalid due to missing qemu binary")
	}

	hasQemuBlocker := false
	for _, code := range result.BlockingReasonCodes {
		if code == schema.ReasonErrLaunchBackendNotSupported {
			hasQemuBlocker = true
		}
	}

	if !hasQemuBlocker {
		t.Errorf("expected ERR_LAUNCH_BACKEND_NOT_SUPPORTED blocker, got %v", result.BlockingReasonCodes)
	}

	if result.RejectedBackends["kvm_qemu"] != schema.ReasonErrLaunchMissingCapabilityQemuBinary+" ("+schema.ReasonCapQemuBinaryMissing+")" {
		t.Errorf("expected rejected backend kvm_qemu with CAP_QEMU_BINARY_MISSING error, got %v", result.RejectedBackends)
	}

	foundEvidence := false
	for _, ev := range result.BackendRejectionEvidence {
		if ev.Backend == "kvm_qemu" && ev.ReasonCode == schema.ReasonErrLaunchMissingCapabilityQemuBinary {
			if ev.CapabilityName != nil && *ev.CapabilityName == "qemu_binary_present" {
				if ev.CapabilityReasonCode != nil && *ev.CapabilityReasonCode == "CAP_QEMU_BINARY_MISSING" {
					foundEvidence = true
				}
			}
		}
	}
	if !foundEvidence {
		t.Errorf("expected structured evidence for missing qemu binary, got %+v", result.BackendRejectionEvidence)
	}

	if hint, ok := result.RemediationHints["kvm_qemu_binary"]; !ok || !strings.Contains(hint, "Install qemu-system") {
		t.Errorf("expected remediation hint for missing qemu binary, got %v", result.RemediationHints)
	}
}

func TestValidateLaunch_TypedStorage(t *testing.T) {
	env := readFixture(t, "cloudhypervisor_ready_arm.json")
	now := time.Now().Unix()
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion: "v1alpha1",
		WorkloadID:    "wl-launch-typed",
		TenantID:      "tenant-1",
		NodeID:        node.ID,
		RuntimeClass:  "VirtualMachine",
		Architecture:  "aarch64",
		Vcpu:          4,
		MemoryMB:      8192,
		LaunchMode:    "DryRun",
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: "/tmp/vol1.qcow2", Format: "qcow2"},
		},
	}

	result := ValidateLaunch(spec, node)
	if !result.IsValid {
		t.Errorf("expected typed storage to validate successfully, got false: %v", result.BlockingReasonCodes)
	}
}

func TestValidateLaunch_TypedStoragePrecedence(t *testing.T) {
	env := readFixture(t, "cloudhypervisor_ready_arm.json")
	now := time.Now().Unix()
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion:  "v1alpha1",
		WorkloadID:     "wl-launch-prec",
		TenantID:       "tenant-1",
		NodeID:         node.ID,
		RuntimeClass:   "VirtualMachine",
		Architecture:   "aarch64",
		Vcpu:           4,
		MemoryMB:       8192,
		LaunchMode:     "DryRun",
		ImageReference: "/tmp/legacy.img", // Should trigger warning
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: "/tmp/typed.qcow2", Format: "qcow2"},
		},
	}

	result := ValidateLaunch(spec, node)
	if !result.IsValid {
		t.Errorf("expected validation to pass, got false")
	}

	hasWarning := false
	for _, w := range result.Warnings {
		if w == schema.ReasonWarnDeprecatedImageReference {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Errorf("expected warning for legacy image reference, got: %v", result.Warnings)
	}
}

func TestValidateLaunch_FirecrackerQcow2Rejected(t *testing.T) {
	env := readFixture(t, "firecracker_host_ready.json")
	now := time.Now().Unix()
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion: "v1alpha1",
		WorkloadID:    "wl-launch-fc-qcow2",
		TenantID:      "tenant-1",
		NodeID:        node.ID,
		RuntimeClass:  "MicroVM",
		Architecture:  "x86_64",
		Vcpu:          2,
		MemoryMB:      1024,
		LaunchMode:    "DryRun",
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: "/tmp/vol.qcow2", Format: "qcow2"},
		},
	}

	result := ValidateLaunch(spec, node)
	if result.IsValid {
		t.Errorf("expected firecracker to reject qcow2, but it validated")
	}

	if reason, ok := result.RejectedBackends["firecracker"]; !ok || !strings.Contains(reason, schema.ReasonErrLaunchInvalidStorageFormat) {
		t.Errorf("expected firecracker rejection with ERR_LAUNCH_INVALID_STORAGE_FORMAT, got %v", result.RejectedBackends)
	}
}

func TestValidateLaunch_MissingArtifact(t *testing.T) {
	env := readFixture(t, "healthy_arm_production.json")
	now := time.Now().Unix()
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion: "v1alpha1",
		WorkloadID:    "wl-launch-missing-artifact",
		TenantID:      "tenant-1",
		NodeID:        node.ID,
		RuntimeClass:  "VirtualMachine",
		Architecture:  "aarch64",
		Vcpu:          2,
		MemoryMB:      1024,
		LaunchMode:    "DryRun",
		// No ImageReference, no Storage
	}

	result := ValidateLaunch(spec, node)
	if result.IsValid {
		t.Errorf("expected failure due to missing artifact, but it validated")
	}

	if reason, ok := result.RejectedBackends["kvm_qemu"]; !ok || !strings.Contains(reason, schema.ReasonErrLaunchMissingArtifact) {
		t.Errorf("expected rejection ERR_LAUNCH_MISSING_ARTIFACT for kvm_qemu, got %v", result.RejectedBackends)
	}
}

func TestValidateLaunch_SecurityContextRequiresSeccomp(t *testing.T) {
	env := readFixture(t, "missing_seccomp.json")
	now := time.Now().Unix()
	env.TimestampSec = now
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion: "v1alpha1",
		WorkloadID:    "wl-launch-seccomp",
		TenantID:      "tenant-1",
		NodeID:        node.ID,
		RuntimeClass:  "VirtualMachine",
		Architecture:  "x86_64",
		Vcpu:          2,
		MemoryMB:      1024,
		LaunchMode:    "DryRun",
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: "/tmp/vol.qcow2", Format: "qcow2"},
		},
		Security: &launch.SecurityContextSpec{
			SeccompProfile: "runtime/default",
		},
	}

	result := ValidateLaunch(spec, node)
	if result.IsValid {
		t.Errorf("expected failure due to missing seccomp capability, but it validated")
	}

	hasSeccompBlocker := false
	for _, code := range result.BlockingReasonCodes {
		if code == schema.ReasonErrLaunchMissingCapabilitySeccomp {
			hasSeccompBlocker = true
		}
	}

	if !hasSeccompBlocker {
		t.Errorf("expected ERR_LAUNCH_MISSING_CAPABILITY_SECCOMP blocker, got %v", result.BlockingReasonCodes)
	}

	if hint, ok := result.RemediationHints["kernel_seccomp"]; !ok || !strings.Contains(hint, "CONFIG_SECCOMP") {
		t.Errorf("expected remediation hint for missing seccomp, got %v", result.RemediationHints)
	}
}

func TestValidateLaunch_SecurityContextRequiresNamespaces(t *testing.T) {
	env := readFixture(t, "missing_seccomp.json") // We can mock missing namespaces locally for test
	now := time.Now().Unix()
	env.TimestampSec = now
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
		if env.Capabilities[i].Feature == "kernel_namespaces_supported" {
			env.Capabilities[i].State = "Unsupported"
			rc := "CAP_NAMESPACES_MISSING"
			env.Capabilities[i].ReasonCode = &rc
		}
	}
	node := ProjectEnvelope(env)

	spec := launch.LaunchSpec{
		SchemaVersion: "v1alpha1",
		WorkloadID:    "wl-launch-namespaces",
		TenantID:      "tenant-1",
		NodeID:        node.ID,
		RuntimeClass:  "VirtualMachine",
		Architecture:  "x86_64",
		Vcpu:          2,
		MemoryMB:      1024,
		LaunchMode:    "DryRun",
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: "/tmp/vol.qcow2", Format: "qcow2"},
		},
		Security: &launch.SecurityContextSpec{
			DropCapabilities: []string{"CAP_SYS_ADMIN"},
		},
	}

	result := ValidateLaunch(spec, node)
	if !result.IsValid {
		t.Errorf("expected validation to pass even when namespaces capability is missing, because dropping capabilities does not require full namespace support, got false: %v", result.BlockingReasonCodes)
	}
}

func TestValidateLaunch_ReasonCodeRegistryHygiene(t *testing.T) {
	// Pick a few representative fixtures that test different validation failure modes
	fixtures := []string{
		"cloudhypervisor_binary_missing.json",
		"firecracker_partial_fail.json",
		"missing_kvm_x86.json",
		"missing_qemu_binary.json",
		"healthy_unsupported_compatibility.json",
		"healthy_x86_kvm_openable.json",
		"stale_telemetry.json",
	}

	for _, fname := range fixtures {
		env := readFixture(t, fname)
		node := ProjectEnvelope(env)

		spec := launch.LaunchSpec{
			SchemaVersion: "v1alpha1",
			WorkloadID:    "wl-test",
			TenantID:      "tenant-1",
			NodeID:        node.ID,
			RuntimeClass:  "VirtualMachine",
			Architecture:  "x86_64",
			LaunchMode:    "DryRun",
			Vcpu:          2,
			MemoryMB:      1024,
		}

		res := ValidateLaunch(spec, node)

		for _, code := range res.BlockingReasonCodes {
			if !schema.IsKnownReasonCode(code) {
				t.Errorf("fixture %s produced unregistered blocking reason code: %q", fname, code)
			}
		}

		for _, code := range res.Warnings {
			if !schema.IsKnownReasonCode(code) {
				t.Errorf("fixture %s produced unregistered warning reason code: %q", fname, code)
			}
		}

		for _, ev := range res.BackendRejectionEvidence {
			if !schema.IsKnownReasonCode(ev.ReasonCode) {
				t.Errorf("fixture %s produced unregistered structured reason code: %q", fname, ev.ReasonCode)
			}
			if ev.CapabilityReasonCode != nil && *ev.CapabilityReasonCode != "" {
				if !schema.IsKnownReasonCode(*ev.CapabilityReasonCode) {
					t.Errorf("fixture %s produced unregistered structured capability reason code: %q", fname, *ev.CapabilityReasonCode)
				}
			}
		}

		for _, reason := range res.RejectedBackends {
			if reason == "" {
				t.Errorf("fixture %s produced an empty rejected backend reason", fname)
			}
			words := strings.Split(reason, " ")
			for _, w := range words {
				w = strings.Trim(w, "()[]")
				if strings.HasPrefix(w, "ERR_") || strings.HasPrefix(w, "CAP_") {
					if !schema.IsKnownReasonCode(w) {
						t.Errorf("fixture %s produced unregistered backend rejection code in message %q: %q", fname, reason, w)
					}
				}
			}
		}
	}
}
