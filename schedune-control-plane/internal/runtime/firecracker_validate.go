package runtime

import (
	"fmt"
	"path/filepath"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// FirecrackerExecutor validates and dry-runs Firecracker execution
type FirecrackerExecutor struct{}

func (k *FirecrackerExecutor) Prepare(spec launch.LaunchSpec) (launch.PreparedLaunch, error) {
	binPath := "firecracker"

	kernel, rootfs := resolveFirecrackerDisks(spec)

	if kernel == "" || rootfs == "" {
		return launch.PreparedLaunch{}, fmt.Errorf("missing kernel or rootfs path for firecracker artifact model")
	}

	controlSocket, err := GetControlSocketPath(spec.WorkloadID, "firecracker")
	if err != nil {
		return launch.PreparedLaunch{}, fmt.Errorf("failed to resolve control socket: %w", err)
	}

	runtimeDir := filepath.Dir(controlSocket)

	args := []string{
		"--api-sock", controlSocket,
		// In a real execution, we'd write a config JSON and pass it, or call the API.
		// For V0 dry-run, we just mock the arguments.
		"--config-file", filepath.Join(runtimeDir, "fc-config.json"),
	}

	return launch.PreparedLaunch{
		RuntimeBackend:  "firecracker",
		MemoryMB:        spec.MemoryMB,
		Vcpu:            spec.Vcpu,
		StartupGraceSec: 2,
		Firecracker: &launch.PreparedFirecrackerLaunch{
			BinaryPath:        binPath,
			KernelImagePath:   kernel,
			RootfsPath:        rootfs,
			CommandArgs:       args,
			ControlSocketPath: controlSocket,
		},
	}, nil
}

func (k *FirecrackerExecutor) Execute(prepared launch.PreparedLaunch) (int, error) {
	return 0, fmt.Errorf("firecracker actual execution not implemented in Data Plane V0 - validate/dry-run only")
}

func (k *FirecrackerExecutor) Terminate(pid int) error {
	return fmt.Errorf("firecracker termination not implemented")
}
