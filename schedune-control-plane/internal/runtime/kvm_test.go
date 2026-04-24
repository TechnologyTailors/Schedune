package runtime

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"os"
	"testing"
)

func TestKvmExecutor_PrepareMissingImage(t *testing.T) {
	exec := &KvmExecutor{}

	spec := launch.LaunchSpec{
		Architecture:   "aarch64",
		ImageReference: "/tmp/non_existent_image_12345.qcow2",
		Vcpu:           2,
		MemoryMB:       1024,
	}

	_, err := exec.Prepare(spec)
	if err == nil {
		t.Errorf("expected Prepare to fail with missing artifact")
	}
}

func TestKvmExecutor_PrepareValidImage(t *testing.T) {
	exec := &KvmExecutor{}

	// create a dummy file
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

	if len(prep.KvmQemu.CommandArgs) < 6 {
		t.Errorf("expected populated command args, got %v", prep.KvmQemu.CommandArgs)
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
