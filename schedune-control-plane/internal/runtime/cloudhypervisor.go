package runtime

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// CloudHypervisorExecutor implements VM execution via Cloud Hypervisor
type CloudHypervisorExecutor struct{}

func (k *CloudHypervisorExecutor) Prepare(spec launch.LaunchSpec) (launch.PreparedLaunch, error) {
	binPath := "cloud-hypervisor"

	artifactPath := spec.ImageReference
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return launch.PreparedLaunch{}, fmt.Errorf("artifact missing at host path: %s", artifactPath)
	}

	args := []string{
		"--memory", fmt.Sprintf("size=%dM", spec.MemoryMB),
		"--cpus", fmt.Sprintf("boot=%d", spec.Vcpu),
		"--disk", fmt.Sprintf("path=%s", artifactPath),
	}

	return launch.PreparedLaunch{
		RuntimeBackend: "cloud_hypervisor",
		MemoryMB:       spec.MemoryMB,
		Vcpu:           spec.Vcpu,
		CloudHypervisor: &launch.PreparedCloudHypervisorLaunch{
			BinaryPath:   binPath,
			ArtifactPath: artifactPath,
			CommandArgs:  args,
		},
	}, nil
}

func (k *CloudHypervisorExecutor) Execute(prepared launch.PreparedLaunch) (int, error) {
	if prepared.CloudHypervisor == nil {
		return 0, fmt.Errorf("missing cloud_hypervisor prepared state")
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
