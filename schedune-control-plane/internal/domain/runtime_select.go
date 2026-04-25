package domain

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
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
// It returns the selected backend name, the structured rejection evidence, and a legacy map of rejected backends to their rejection reason codes.
func SelectBackend(spec launch.LaunchSpec, node NodeRecord) (string, []launch.BackendRejectionEvidence, map[string]string) {
	rejected := make(map[string]string)
	var evidence []launch.BackendRejectionEvidence

	reject := func(backend, reasonCode string, capName *string, cap *NodeCapabilityRecord) {
		rejected[backend] = reasonCode
		ev := launch.BackendRejectionEvidence{
			Backend:    backend,
			ReasonCode: reasonCode,
		}
		if capName != nil {
			ev.CapabilityName = capName
		}
		if cap != nil {
			state := cap.State
			ev.CapabilityState = &state
			if cap.ReasonCode != "" {
				rc := cap.ReasonCode
				ev.CapabilityReasonCode = &rc
				rejected[backend] += " (" + cap.ReasonCode + ")"
			}
			stale := cap.IsStale
			ev.CapabilityStale = &stale
		}
		evidence = append(evidence, ev)
	}

	storage := normalizeStorage(spec)

	if spec.RuntimeClass == "MicroVM" {
		// Firecracker path
		capName := "kvm_vm_launch"
		kvmCap, kvmExists := node.Capabilities[capName]
		if !kvmExists || kvmCap.State != "Supported" || kvmCap.IsStale {
			var capPtr *NodeCapabilityRecord
			if kvmExists {
				capPtr = &kvmCap
			}
			reject("firecracker", schema.ReasonErrLaunchMissingCapabilityKvmQemu, &capName, capPtr)
			return "", evidence, rejected
		}

		capName = "firecracker_binary_present"
		binCap, binExists := node.Capabilities[capName]
		if !binExists || binCap.State != "Supported" || binCap.IsStale {
			var capPtr *NodeCapabilityRecord
			if binExists {
				capPtr = &binCap
			}
			reject("firecracker", schema.ReasonErrLaunchMissingCapabilityFcBinary, &capName, capPtr)
			return "", evidence, rejected
		}

		capName = "firecracker_tun_ready"
		tunCap, tunExists := node.Capabilities[capName]
		if !tunExists || tunCap.State != "Supported" || tunCap.IsStale {
			var capPtr *NodeCapabilityRecord
			if tunExists {
				capPtr = &tunCap
			}
			reject("firecracker", schema.ReasonErrLaunchMissingCapabilityFcTun, &capName, capPtr)
			return "", evidence, rejected
		}

		capName = "firecracker_cgroups_ready"
		cgCap, cgExists := node.Capabilities[capName]
		if !cgExists || cgCap.State != "Supported" || cgCap.IsStale {
			var capPtr *NodeCapabilityRecord
			if cgExists {
				capPtr = &cgCap
			}
			reject("firecracker", schema.ReasonErrLaunchMissingCapabilityFcCgroups, &capName, capPtr)
			return "", evidence, rejected
		}

		// Artifact validation
		if len(storage) == 0 {
			reject("firecracker", schema.ReasonErrLaunchInvalidFirecrackerArtifactModel, nil, nil)
			return "", evidence, rejected
		}
		for _, s := range storage {
			if s.Format == "qcow2" {
				reject("firecracker", schema.ReasonErrLaunchInvalidStorageFormat, nil, nil)
				return "", evidence, rejected
			}
		}

		return "firecracker", evidence, rejected
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
				reject(spec.RuntimeBackendPreference, schema.ReasonErrLaunchBackendNotSupported, nil, nil)
				return "", evidence, rejected
			}
		}

		for _, backend := range backends {
			if backend == "cloud_hypervisor" {
				capName := "kvm_vm_launch"
				kvmCap, kvmExists := node.Capabilities[capName]
				if !kvmExists || kvmCap.State != "Supported" || kvmCap.IsStale {
					var capPtr *NodeCapabilityRecord
					if kvmExists {
						capPtr = &kvmCap
					}
					reject("cloud_hypervisor", schema.ReasonErrLaunchMissingCapabilityKvmQemu, &capName, capPtr)
					continue
				}

				capName = "cloud_hypervisor_binary_present"
				binCap, binExists := node.Capabilities[capName]
				if !binExists || binCap.State != "Supported" || binCap.IsStale {
					var capPtr *NodeCapabilityRecord
					if binExists {
						capPtr = &binCap
					}
					reject("cloud_hypervisor", schema.ReasonErrLaunchMissingCapabilityChBinary, &capName, capPtr)
					continue
				}

				if len(storage) == 0 {
					reject("cloud_hypervisor", schema.ReasonErrLaunchMissingArtifact, nil, nil)
					continue
				}
				return "cloud_hypervisor", evidence, rejected
			}

			if backend == "kvm_qemu" {
				capName := "kvm_vm_launch"
				cap, exists := node.Capabilities[capName]
				if !exists || cap.State != "Supported" || cap.IsStale {
					var capPtr *NodeCapabilityRecord
					if exists {
						capPtr = &cap
					}
					reject("kvm_qemu", schema.ReasonErrLaunchMissingCapabilityKvmQemu, &capName, capPtr)
					continue
				}

				capName = "qemu_binary_present"
				binCap, binExists := node.Capabilities[capName]
				if !binExists || binCap.State != "Supported" || binCap.IsStale {
					var capPtr *NodeCapabilityRecord
					if binExists {
						capPtr = &binCap
					}
					reject("kvm_qemu", schema.ReasonErrLaunchMissingCapabilityQemuBinary, &capName, capPtr)
					continue
				}

				if len(storage) == 0 {
					reject("kvm_qemu", schema.ReasonErrLaunchMissingArtifact, nil, nil)
					continue
				}
				return "kvm_qemu", evidence, rejected
			}
		}
	}

	return "", evidence, rejected
}
