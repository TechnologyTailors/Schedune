package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// CloudHypervisorExecutor implements VM execution via Cloud Hypervisor
type CloudHypervisorExecutor struct{}

func (k *CloudHypervisorExecutor) Prepare(spec launch.LaunchSpec) (launch.PreparedLaunch, error) {
	binPath := "cloud-hypervisor"

	artifactPath, _ := resolvePrimaryDisk(spec)

	if artifactPath == "" {
		return launch.PreparedLaunch{}, fmt.Errorf("artifact missing for execution")
	}

	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return launch.PreparedLaunch{}, fmt.Errorf("artifact missing at host path: %s", artifactPath)
	}

	controlSocket, err := GetControlSocketPath(spec.WorkloadID, "cloudhypervisor")
	if err != nil {
		return launch.PreparedLaunch{}, fmt.Errorf("failed to resolve control socket: %w", err)
	}

	args := []string{
		"--memory", fmt.Sprintf("size=%dM", spec.MemoryMB),
		"--cpus", fmt.Sprintf("boot=%d", spec.Vcpu),
		"--disk", fmt.Sprintf("path=%s", artifactPath),
		"--api-socket", controlSocket,
	}

	return launch.PreparedLaunch{
		RuntimeBackend:  "cloud_hypervisor",
		MemoryMB:        spec.MemoryMB,
		Vcpu:            spec.Vcpu,
		StartupGraceSec: 3,
		CloudHypervisor: &launch.PreparedCloudHypervisorLaunch{
			BinaryPath:        binPath,
			ArtifactPath:      artifactPath,
			CommandArgs:       args,
			ControlSocketPath: controlSocket,
		},
	}, nil
}

func (k *CloudHypervisorExecutor) Execute(prepared launch.PreparedLaunch) (int, error) {
	if prepared.CloudHypervisor == nil {
		return 0, fmt.Errorf("missing cloud_hypervisor prepared state")
	}

	if prepared.CloudHypervisor.ControlSocketPath != "" {
		if err := os.MkdirAll(filepath.Dir(prepared.CloudHypervisor.ControlSocketPath), 0755); err != nil {
			return 0, fmt.Errorf("failed to create runtime directory: %w", err)
		}
	}

	cmd := exec.Command(prepared.CloudHypervisor.BinaryPath, prepared.CloudHypervisor.CommandArgs...)
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("executable failed to start: %w", err)
	}
	return cmd.Process.Pid, nil
}

func (k *CloudHypervisorExecutor) Terminate(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
