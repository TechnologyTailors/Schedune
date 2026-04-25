package domain

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"testing"
	"time"
)

func TestSelectBackend_MicroVM(t *testing.T) {
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
		RuntimeClass:    "MicroVM",
		KernelImagePath: "/tmp/vmlinux",
		RootfsPath:      "/tmp/rootfs.ext4",
	}

	backend, _, rejected := SelectBackend(spec, node)
	if backend != "firecracker" {
		t.Errorf("expected firecracker, got %s. rejections: %v", backend, rejected)
	}

	// Missing artifacts
	spec.KernelImagePath = ""
	backend, _, rejected = SelectBackend(spec, node)
	if backend != "" {
		t.Errorf("expected rejection due to missing artifact, got %s", backend)
	}
	if reason := rejected["firecracker"]; reason != "ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL" {
		t.Errorf("expected ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL, got %s", reason)
	}
}

func TestSelectBackend_VirtualMachine(t *testing.T) {
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
		RuntimeClass:   "VirtualMachine",
		ImageReference: "/tmp/image.qcow2",
	}

	// Should prefer cloud_hypervisor
	backend, _, rejected := SelectBackend(spec, node)
	if backend != "cloud_hypervisor" {
		t.Errorf("expected cloud_hypervisor, got %s. rejections: %v", backend, rejected)
	}
}

func TestSelectBackend_Fallback(t *testing.T) {
	env := readFixture(t, "healthy_arm_production.json") // KVM yes, CH no
	now := time.Now().Unix()
	env.TimestampSec = now
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)
	spec := launch.LaunchSpec{
		RuntimeClass:   "VirtualMachine",
		ImageReference: "/tmp/image.qcow2",
	}

	backend, _, rejected := SelectBackend(spec, node)
	if backend != "kvm_qemu" {
		t.Errorf("expected kvm_qemu fallback, got %s. rejections: %v", backend, rejected)
	}
}
