package runtime

import (
	"fmt"
	"os"
	"os/exec"

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
	
	// Basic artifact validation to prove the trace pipeline
	artifactPath := spec.ImageReference
	
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return launch.PreparedLaunch{}, fmt.Errorf("artifact missing at host path: %s", artifactPath)
	}

	args := []string{
		"-m", fmt.Sprintf("%d", spec.MemoryMB),
		"-smp", fmt.Sprintf("%d", spec.Vcpu),
		"-drive", fmt.Sprintf("file=%s,format=qcow2", artifactPath),
		"-nographic",
	}

	return launch.PreparedLaunch{
		RuntimeBackend: "kvm",
		BinaryPath:     binPath,
		ArtifactPath:   artifactPath,
		CommandArgs:    args,
		MemoryMB:       spec.MemoryMB,
		Vcpu:           spec.Vcpu,
	}, nil
}

func (k *KvmExecutor) Execute(prepared launch.PreparedLaunch) (int, error) {
	cmd := exec.Command(prepared.BinaryPath, prepared.CommandArgs...)
	// We run it asynchronously for V0 to prove the "Running" state
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("executable failed to start: %v", err)
	}
	return cmd.Process.Pid, nil
}

func (k *KvmExecutor) Terminate(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	// Simple kill for MVP
	return proc.Kill()
}
