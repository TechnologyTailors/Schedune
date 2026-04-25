package domain

import (
	"fmt"
	"strings"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// ValidateLaunch assesses whether a chosen node is physically capable of executing the requested spec.
func ValidateLaunch(spec launch.LaunchSpec, node NodeRecord) launch.LaunchValidationResult {
	result := launch.LaunchValidationResult{
		IsValid:              true,
		RejectedBackends:     make(map[string]string),
		BlockingReasonCodes:  []string{},
		Warnings:             []string{},
		RequiredHostFeatures: []string{},
		MissingHostFeatures:  []string{},
		ValidationTrace:      []string{"Started validation against node " + node.ID},
		RemediationHints:     make(map[string]string),
	}

	// 1. Layer 1: Generic launch checks
	if spec.Architecture != node.Identity.Architecture {
		result.IsValid = false
		result.BlockingReasonCodes = append(result.BlockingReasonCodes, schema.ReasonErrLaunchArchMismatch)
		result.ValidationTrace = append(result.ValidationTrace, "Failed: Architecture mismatch. Node is "+node.Identity.Architecture)
	}

	if spec.ImageReference != "" || spec.KernelImagePath != "" {
		result.Warnings = append(result.Warnings, schema.ReasonWarnDeprecatedImageReference)
		if len(spec.Storage) > 0 {
			result.ValidationTrace = append(result.ValidationTrace, "Warning: Both typed Storage and legacy ImageReference provided. Typed Storage takes precedence.")
		} else {
			result.ValidationTrace = append(result.ValidationTrace, "Warning: Using deprecated legacy artifact fields. Please migrate to typed Storage array.")
		}
	}

	if len(spec.NetworkAttachments) > 0 {
		result.Warnings = append(result.Warnings, schema.ReasonWarnDeprecatedNetworkAttachments)
		result.ValidationTrace = append(result.ValidationTrace, "Warning: Using deprecated NetworkAttachments field. Please migrate to typed Networks array.")
	}

	// Security Context Validation
	if spec.Security != nil {
		if spec.Security.SeccompProfile != "" {
			cap, exists := node.Capabilities["kernel_seccomp_supported"]
			if !exists || cap.State != schema.CapabilityStateSupported || cap.IsStale {
				result.IsValid = false
				result.BlockingReasonCodes = append(result.BlockingReasonCodes, schema.ReasonErrLaunchMissingCapabilitySeccomp)
				result.ValidationTrace = append(result.ValidationTrace, "Failed: Security Context requires seccomp profile, but kernel seccomp is not supported or missing.")
			} else {
				result.ValidationTrace = append(result.ValidationTrace, "Passed: Node supports kernel seccomp for requested profile.")
			}
		}
	}

	if !result.IsValid {
		result.ExplainabilityText = "Node cannot launch workload due to generic preflight blockers."
		result.RemediationHints = generateRemediationHints(result)
		return result
	}

	// 2. Layer 2 & 3: Backend-specific capability and artifact checks via Runtime Selector
	selectedBackend, evidence, rejectedBackends := SelectBackend(spec, node)
	result.RejectedBackends = rejectedBackends
	result.BackendRejectionEvidence = evidence
	result.SelectedBackend = selectedBackend

	if selectedBackend == "" {
		result.IsValid = false
		result.BlockingReasonCodes = append(result.BlockingReasonCodes, schema.ReasonErrLaunchBackendNotSupported)
		result.ValidationTrace = append(result.ValidationTrace, "Failed: No supported runtime backend found for spec.")
		for backend, reason := range rejectedBackends {
			result.ValidationTrace = append(result.ValidationTrace, fmt.Sprintf("Backend %s rejected: %s", backend, reason))
		}
		result.ExplainabilityText = "Node cannot launch workload due to missing backend prerequisites or constraints."
		result.RemediationHints = generateRemediationHints(result)
		return result
	}

	// 3. Layer 4: Setup context for preparation phase validation
	result.ValidationTrace = append(result.ValidationTrace, "Passed: Selected backend "+selectedBackend)

	result.ExplainabilityText = "Node is fully capable of executing this launch spec."

	if selectedBackend == schema.BackendKvmQemu {
		result.RecommendedRuntime = "qemu-system-" + spec.Architecture
	} else if selectedBackend == schema.BackendCloudHypervisor {
		result.RecommendedRuntime = "cloud-hypervisor"
	} else if selectedBackend == schema.BackendFirecracker {
		result.RecommendedRuntime = "firecracker"
	}

	if spec.LaunchMode == "DryRun" && result.IsValid {
		result.ValidationTrace = append(result.ValidationTrace, fmt.Sprintf("DryRun: Successfully simulated allocation of %d vcpus.", spec.Vcpu))
	}

	result.RemediationHints = generateRemediationHints(result)
	return result
}

func generateRemediationHints(result launch.LaunchValidationResult) map[string]string {
	hints := make(map[string]string)

	for _, code := range result.BlockingReasonCodes {
		if code == schema.ReasonErrLaunchArchMismatch {
			hints["architecture"] = "Ensure the requested architecture matches the node architecture."
		}
		if code == schema.ReasonErrLaunchBackendNotSupported {
			hints["backend"] = "Check the RejectedBackends map for specific missing capabilities."
		}
		if code == schema.ReasonErrLaunchMissingCapabilitySeccomp {
			hints["kernel_seccomp"] = "Ensure the host kernel is compiled with CONFIG_SECCOMP and actions_avail is readable."
		}
	}

	for backend, reason := range result.RejectedBackends {
		if backend == schema.BackendKvmQemu && strings.Contains(reason, schema.ReasonCapQemuBinaryMissing) {
			hints["kvm_qemu_binary"] = "Install qemu-system-x86_64 or qemu-system-aarch64 on the host."
		}
		if backend == schema.BackendKvmQemu && strings.Contains(reason, schema.ReasonCapKvmMissing) {
			hints["kvm_qemu_kvm"] = "Enable KVM in BIOS or load kvm kernel modules."
		}
		if backend == schema.BackendKvmQemu && strings.Contains(reason, schema.ReasonCapKvmNotOpenablePerms) {
			hints["kvm_qemu_perms"] = "Ensure the Schedune agent has rw permissions to /dev/kvm."
		}
		if backend == schema.BackendCloudHypervisor && strings.Contains(reason, schema.ReasonCapCloudHypervisorBinaryMissing) {
			hints["cloud_hypervisor_binary"] = "Install cloud-hypervisor binary on the host."
		}
		if backend == schema.BackendCloudHypervisor && strings.Contains(reason, schema.ReasonCapKvmMissing) {
			hints["cloud_hypervisor_kvm"] = "Enable KVM in BIOS or load kvm kernel modules."
		}
		if backend == schema.BackendFirecracker && strings.Contains(reason, schema.ReasonCapFirecrackerBinaryMissing) {
			hints["firecracker_binary"] = "Install firecracker binary on the host."
		}
		if backend == schema.BackendFirecracker && strings.Contains(reason, schema.ReasonCapFirecrackerTunMissing) {
			hints["firecracker_tun"] = "Ensure the tun kernel module is loaded (/dev/net/tun)."
		}
		if backend == schema.BackendFirecracker && strings.Contains(reason, schema.ReasonCapFirecrackerCgroupsMissing) {
			hints["firecracker_cgroups"] = "Ensure cgroups v2 are mounted on the host."
		}
		if backend == schema.BackendFirecracker && strings.Contains(reason, schema.ReasonCapKvmMissing) {
			hints["firecracker_kvm"] = "Enable KVM in BIOS or load kvm kernel modules."
		}
		if strings.Contains(reason, schema.ReasonErrLaunchMissingArtifact) {
			hints["artifact_missing"] = "Ensure ImageReference is provided for the workload."
		}
		if strings.Contains(reason, schema.ReasonErrLaunchInvalidFirecrackerArtifactModel) {
			hints["firecracker_artifact"] = "Ensure KernelImagePath and RootfsPath are provided for MicroVMs."
		}
	}

	return hints
}
