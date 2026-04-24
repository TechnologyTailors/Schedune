package domain

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// SelectBackend determines the appropriate runtime backend for a given launch spec and node.
// It returns the selected backend name, and a map of rejected backends to their rejection reason codes.
func SelectBackend(spec launch.LaunchSpec, node NodeRecord) (string, map[string]string) {
	rejected := make(map[string]string)

	if spec.RuntimeClass == "MicroVM" {
		// Firecracker path
		cap, exists := node.Capabilities["firecracker_launch"]
		if !exists || cap.State != "Supported" || cap.IsStale {
			rejected["firecracker"] = "CAP_FIRECRACKER_PREREQS_MISSING"
			return "", rejected
		}

		// Artifact validation
		if spec.KernelImagePath == "" || spec.RootfsPath == "" {
			rejected["firecracker"] = "ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL"
			return "", rejected
		}

		return "firecracker", rejected
	}

	if spec.RuntimeClass == "VirtualMachine" {
		backends := []string{"cloud_hypervisor", "kvm_qemu"}

		// Check explicitly requested backend
		if spec.RuntimeBackendPreference != "" {
			if spec.RuntimeBackendPreference == "cloud_hypervisor" || spec.RuntimeBackendPreference == "kvm_qemu" {
				backends = []string{spec.RuntimeBackendPreference}
				if spec.AllowBackendFallback {
					if spec.RuntimeBackendPreference == "cloud_hypervisor" {
						backends = []string{"cloud_hypervisor", "kvm_qemu"}
					} else {
						backends = []string{"kvm_qemu", "cloud_hypervisor"}
					}
				}
			} else {
				rejected[spec.RuntimeBackendPreference] = "ERR_LAUNCH_BACKEND_NOT_SUPPORTED"
				return "", rejected
			}
		}

		for _, backend := range backends {
			if backend == "cloud_hypervisor" {
				cap, exists := node.Capabilities["cloud_hypervisor_launch"]
				if !exists || cap.State != "Supported" || cap.IsStale {
					reason := "ERR_LAUNCH_MISSING_CAPABILITY_CLOUDHYPERVISOR"
					if exists && cap.ReasonCode != "" {
						reason += " (" + cap.ReasonCode + ")"
					}
					rejected["cloud_hypervisor"] = reason
					continue
				}
				if spec.ImageReference == "" {
					rejected["cloud_hypervisor"] = "ERR_LAUNCH_MISSING_ARTIFACT"
					continue
				}
				return "cloud_hypervisor", rejected
			}

			if backend == "kvm_qemu" {
				cap, exists := node.Capabilities["kvm_vm_launch"]
				if !exists || cap.State != "Supported" || cap.IsStale {
					reason := "ERR_LAUNCH_MISSING_CAPABILITY_KVM_QEMU"
					if exists && cap.ReasonCode != "" {
						reason += " (" + cap.ReasonCode + ")"
					}
					rejected["kvm_qemu"] = reason
					continue
				}
				if spec.ImageReference == "" {
					rejected["kvm_qemu"] = "ERR_LAUNCH_MISSING_ARTIFACT"
					continue
				}
				return "kvm_qemu", rejected
			}
		}
	}

	return "", rejected
}
