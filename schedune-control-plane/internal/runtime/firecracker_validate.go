package runtime

import (
	"fmt"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// FirecrackerExecutor validates and dry-runs Firecracker execution
type FirecrackerExecutor struct{}

func (k *FirecrackerExecutor) Prepare(spec launch.LaunchSpec) (launch.PreparedLaunch, error) {
	binPath := "firecracker"
	
	if spec.KernelImagePath == "" || spec.RootfsPath == "" {
		return launch.PreparedLaunch{}, fmt.Errorf("missing kernel or rootfs path for firecracker artifact model")
	}

	args := []string{
		"--api-sock", "/tmp/firecracker.socket",
		// In a real execution, we'd write a config JSON and pass it, or call the API.
		// For V0 dry-run, we just mock the arguments.
		"--config-file", "/tmp/fc-config.json",
	}

	return launch.PreparedLaunch{
		RuntimeBackend: "firecracker",
		MemoryMB:       spec.MemoryMB,
		Vcpu:           spec.Vcpu,
		Firecracker: &launch.PreparedFirecrackerLaunch{
			BinaryPath:      binPath,
			KernelImagePath: spec.KernelImagePath,
			RootfsPath:      spec.RootfsPath,
			CommandArgs:     args,
		},
	}, nil
}

func (k *FirecrackerExecutor) Execute(prepared launch.PreparedLaunch) (int, error) {
	return 0, fmt.Errorf("firecracker actual execution not implemented in Data Plane V0 - validate/dry-run only")
}

func (k *FirecrackerExecutor) Terminate(pid int) error {
	return fmt.Errorf("firecracker termination not implemented")
}
