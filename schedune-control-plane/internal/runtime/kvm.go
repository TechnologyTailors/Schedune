package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// Executor defines the strict abstraction for preparing and executing runtimes
type Executor interface {
	Prepare(spec launch.LaunchSpec) (launch.PreparedLaunch, error)
	Execute(prepared launch.PreparedLaunch) (int, error) // Returns PID or error
	Terminate(pid int) error
}

// KvmExecutor is the V0 implementation for KVM-backed VirtualMachine paths
type KvmExecutor struct{}

func (k *KvmExecutor) Prepare(spec launch.LaunchSpec) (launch.PreparedLaunch, error) {
	// For V0 MVP, resolve strictly to the qemu-system binary for the requested arch
	binPath := "qemu-system-" + spec.Architecture

	artifactPath, format := resolvePrimaryDisk(spec)

	if artifactPath == "" {
		return launch.PreparedLaunch{}, fmt.Errorf("artifact missing for execution")
	}

	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return launch.PreparedLaunch{}, fmt.Errorf("artifact missing at host path: %s", artifactPath)
	}

	controlSocket, err := GetControlSocketPath(spec.WorkloadID, "qemu")
	if err != nil {
		return launch.PreparedLaunch{}, fmt.Errorf("failed to resolve control socket: %w", err)
	}

	args := []string{
		"-m", fmt.Sprintf("%d", spec.MemoryMB),
		"-smp", fmt.Sprintf("%d", spec.Vcpu),
		"-drive", fmt.Sprintf("file=%s,format=%s", artifactPath, format),
		"-nographic",
		"-qmp", fmt.Sprintf("unix:%s,server,nowait", controlSocket),
	}

	return launch.PreparedLaunch{
		RuntimeBackend:  "kvm_qemu",
		MemoryMB:        spec.MemoryMB,
		Vcpu:            spec.Vcpu,
		StartupGraceSec: 5,
		KvmQemu: &launch.PreparedQemuLaunch{
			BinaryPath:        binPath,
			ArtifactPath:      artifactPath,
			CommandArgs:       args,
			ControlSocketPath: controlSocket,
		},
	}, nil
}

func (k *KvmExecutor) Execute(prepared launch.PreparedLaunch) (int, error) {
	if prepared.KvmQemu == nil {
		return 0, fmt.Errorf("missing kvm_qemu prepared state")
	}

	// Ensure the runtime directory exists before starting the binary
	if prepared.KvmQemu.ControlSocketPath != "" {
		if err := os.MkdirAll(filepath.Dir(prepared.KvmQemu.ControlSocketPath), 0755); err != nil {
			return 0, fmt.Errorf("failed to create runtime directory: %w", err)
		}
	}

	cmd := exec.Command(prepared.KvmQemu.BinaryPath, prepared.KvmQemu.CommandArgs...)
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("executable failed to start: %w", err)
	}
	return cmd.Process.Pid, nil
}

func (k *KvmExecutor) Terminate(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
