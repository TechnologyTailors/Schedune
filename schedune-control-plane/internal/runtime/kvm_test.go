package runtime

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"os"
	"strings"
	"testing"
)

func TestKvmExecutor_PrepareMissingImage(t *testing.T) {
	exec := &KvmExecutor{}

	spec := launch.LaunchSpec{
		Architecture: "aarch64",
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: "/tmp/non_existent_image_12345.qcow2", Format: "qcow2"},
		},
		Vcpu:     2,
		MemoryMB: 1024,
	}

	_, err := exec.Prepare(spec)
	if err == nil {
		t.Errorf("expected Prepare to fail with missing artifact")
	}
}

func TestKvmExecutor_PrepareValidImageLegacy(t *testing.T) {
	exec := &KvmExecutor{}

	f, err := os.CreateTemp("", "dummy_img_*.qcow2")
	if err != nil {
		t.Fatalf("could not create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	spec := launch.LaunchSpec{
		Architecture:   "aarch64",
		ImageReference: f.Name(),
		Vcpu:           2,
		MemoryMB:       1024,
	}

	prep, err := exec.Prepare(spec)
	if err != nil {
		t.Errorf("expected Prepare to succeed, got %v", err)
	}

	if prep.KvmQemu == nil {
		t.Fatalf("expected KvmQemu prepared state, got nil")
	}

	if prep.KvmQemu.BinaryPath != "qemu-system-aarch64" {
		t.Errorf("expected qemu-system-aarch64, got %s", prep.KvmQemu.BinaryPath)
	}

	foundDrive := false
	for _, arg := range prep.KvmQemu.CommandArgs {
		if strings.Contains(arg, "format=qcow2") {
			foundDrive = true
		}
	}
	if !foundDrive {
		t.Errorf("expected format=qcow2 in args, got %v", prep.KvmQemu.CommandArgs)
	}
}

func TestKvmExecutor_PrepareValidImageTyped(t *testing.T) {
	exec := &KvmExecutor{}

	f, err := os.CreateTemp("", "dummy_img_*.raw")
	if err != nil {
		t.Fatalf("could not create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	spec := launch.LaunchSpec{
		Architecture: "x86_64",
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: f.Name(), Format: "raw"},
		},
		Vcpu:     2,
		MemoryMB: 1024,
	}

	prep, err := exec.Prepare(spec)
	if err != nil {
		t.Errorf("expected Prepare to succeed, got %v", err)
	}

	if prep.KvmQemu == nil {
		t.Fatalf("expected KvmQemu prepared state, got nil")
	}

	foundDrive := false
	for _, arg := range prep.KvmQemu.CommandArgs {
		if strings.Contains(arg, "format=raw") {
			foundDrive = true
		}
	}
	if !foundDrive {
		t.Errorf("expected format=raw in args, got %v", prep.KvmQemu.CommandArgs)
	}
}

func TestCloudHypervisorExecutor_PrepareValidImageTyped(t *testing.T) {
	exec := &CloudHypervisorExecutor{}

	f, err := os.CreateTemp("", "dummy_img_*.raw")
	if err != nil {
		t.Fatalf("could not create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	spec := launch.LaunchSpec{
		Architecture: "x86_64",
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: f.Name(), Format: "raw"},
		},
		Vcpu:     2,
		MemoryMB: 1024,
	}

	prep, err := exec.Prepare(spec)
	if err != nil {
		t.Errorf("expected Prepare to succeed, got %v", err)
	}

	if prep.CloudHypervisor == nil {
		t.Fatalf("expected CloudHypervisor prepared state, got nil")
	}

	foundDrive := false
	for _, arg := range prep.CloudHypervisor.CommandArgs {
		if strings.Contains(arg, "path="+f.Name()) {
			foundDrive = true
		}
	}
	if !foundDrive {
		t.Errorf("expected path in args, got %v", prep.CloudHypervisor.CommandArgs)
	}
}

func TestFirecrackerExecutor_PrepareValidImageTyped(t *testing.T) {
	exec := &FirecrackerExecutor{}

	spec := launch.LaunchSpec{
		Architecture: "x86_64",
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: "/tmp/rootfs.ext4", Format: "ext4", MountPoint: "/"},
			{HostPath: "/tmp/vmlinux", Format: "raw", ReadOnly: true, MountPoint: "/boot/vmlinux"},
		},
		Vcpu:     2,
		MemoryMB: 1024,
	}

	prep, err := exec.Prepare(spec)
	if err != nil {
		t.Errorf("expected Prepare to succeed, got %v", err)
	}

	if prep.Firecracker == nil {
		t.Fatalf("expected Firecracker prepared state, got nil")
	}

	if prep.Firecracker.RootfsPath != "/tmp/rootfs.ext4" {
		t.Errorf("expected rootfs path to be set from typed storage")
	}
	if prep.Firecracker.KernelImagePath != "/tmp/vmlinux" {
		t.Errorf("expected kernel image path to be set from typed storage")
	}
}

func TestKvmExecutor_ExecuteSpawnFails(t *testing.T) {
	exec := &KvmExecutor{}

	prep := launch.PreparedLaunch{
		RuntimeBackend: "kvm_qemu",
		KvmQemu: &launch.PreparedQemuLaunch{
			BinaryPath:  "qemu-system-non-existent-binary-12345",
			CommandArgs: []string{"-m", "1024"},
		},
	}

	_, err := exec.Execute(prep)
	if err == nil {
		t.Errorf("expected Execute to fail due to missing binary")
	}
}
