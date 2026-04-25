package domain

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

func normalizeStorage(spec launch.LaunchSpec) []launch.StorageAttachmentSpec {
	if len(spec.Storage) > 0 {
		return spec.Storage
	}
	if spec.RuntimeClass == "MicroVM" {
		if spec.KernelImagePath != "" && spec.RootfsPath != "" {
			return []launch.StorageAttachmentSpec{
				{HostPath: spec.RootfsPath, Format: "ext4", ReadOnly: false, MountPoint: "/"},
				{HostPath: spec.KernelImagePath, Format: "raw", ReadOnly: true, MountPoint: "/boot/vmlinux"},
			}
		}
	} else if spec.ImageReference != "" {
		return []launch.StorageAttachmentSpec{
			{HostPath: spec.ImageReference, Format: "qcow2", ReadOnly: false},
		}
	}
	return nil
}

func normalizeNetworks(spec launch.LaunchSpec) []launch.NetworkAttachmentSpec {
	if len(spec.Networks) > 0 {
		return spec.Networks
	}
	var nets []launch.NetworkAttachmentSpec
	for _, net := range spec.NetworkAttachments {
		nets = append(nets, launch.NetworkAttachmentSpec{
			Type:       "tap",
			HostDevice: net,
		})
	}
	return nets
}

// SelectBackend determines the appropriate runtime backend for a given launch spec and node.
// It returns the selected backend name, and a map of rejected backends to their rejection reason codes.
func SelectBackend(spec launch.LaunchSpec, node NodeRecord) (string, map[string]string) {
	rejected := make(map[string]string)
	storage := normalizeStorage(spec)

	if spec.RuntimeClass == "MicroVM" {
		// Firecracker path
		kvmCap, kvmExists := node.Capabilities["kvm_vm_launch"]
		if !kvmExists || kvmCap.State != "Supported" || kvmCap.IsStale {
			reason := "ERR_LAUNCH_MISSING_CAPABILITY_KVM_QEMU"
			if kvmExists && kvmCap.ReasonCode != "" {
				reason += " (" + kvmCap.ReasonCode + ")"
			}
			rejected["firecracker"] = reason
			return "", rejected
		}

		binCap, binExists := node.Capabilities["firecracker_binary_present"]
		if !binExists || binCap.State != "Supported" || binCap.IsStale {
			reason := "ERR_LAUNCH_MISSING_CAPABILITY_FC_BINARY"
			if binExists && binCap.ReasonCode != "" {
				reason += " (" + binCap.ReasonCode + ")"
			}
			rejected["firecracker"] = reason
			return "", rejected
		}

		tunCap, tunExists := node.Capabilities["firecracker_tun_ready"]
		if !tunExists || tunCap.State != "Supported" || tunCap.IsStale {
			reason := "ERR_LAUNCH_MISSING_CAPABILITY_FC_TUN"
			if tunExists && tunCap.ReasonCode != "" {
				reason += " (" + tunCap.ReasonCode + ")"
			}
			rejected["firecracker"] = reason
			return "", rejected
		}

		cgCap, cgExists := node.Capabilities["firecracker_cgroups_ready"]
		if !cgExists || cgCap.State != "Supported" || cgCap.IsStale {
			reason := "ERR_LAUNCH_MISSING_CAPABILITY_FC_CGROUPS"
			if cgExists && cgCap.ReasonCode != "" {
				reason += " (" + cgCap.ReasonCode + ")"
			}
			rejected["firecracker"] = reason
			return "", rejected
		}

		// Artifact validation
		if len(storage) == 0 {
			rejected["firecracker"] = "ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL"
			return "", rejected
		}
		for _, s := range storage {
			if s.Format == "qcow2" {
				rejected["firecracker"] = "ERR_LAUNCH_INVALID_STORAGE_FORMAT"
				return "", rejected
			}
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
				kvmCap, kvmExists := node.Capabilities["kvm_vm_launch"]
				if !kvmExists || kvmCap.State != "Supported" || kvmCap.IsStale {
					reason := "ERR_LAUNCH_MISSING_CAPABILITY_KVM_QEMU"
					if kvmExists && kvmCap.ReasonCode != "" {
						reason += " (" + kvmCap.ReasonCode + ")"
					}
					rejected["cloud_hypervisor"] = reason
					continue
				}

				binCap, binExists := node.Capabilities["cloud_hypervisor_binary_present"]
				if !binExists || binCap.State != "Supported" || binCap.IsStale {
					reason := "ERR_LAUNCH_MISSING_CAPABILITY_CH_BINARY"
					if binExists && binCap.ReasonCode != "" {
						reason += " (" + binCap.ReasonCode + ")"
					}
					rejected["cloud_hypervisor"] = reason
					continue
				}

				if len(storage) == 0 {
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
				binCap, binExists := node.Capabilities["qemu_binary_present"]
				if !binExists || binCap.State != "Supported" || binCap.IsStale {
					reason := "ERR_LAUNCH_MISSING_CAPABILITY_QEMU_BINARY"
					if binExists && binCap.ReasonCode != "" {
						reason += " (" + binCap.ReasonCode + ")"
					}
					rejected["kvm_qemu"] = reason
					continue
				}
				if len(storage) == 0 {
					rejected["kvm_qemu"] = "ERR_LAUNCH_MISSING_ARTIFACT"
					continue
				}
				return "kvm_qemu", rejected
			}
		}
	}

	return "", rejected
}
