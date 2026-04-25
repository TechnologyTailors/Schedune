package domain

import (
	"fmt"
	"strings"

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
		result.BlockingReasonCodes = append(result.BlockingReasonCodes, "ERR_LAUNCH_ARCH_MISMATCH")
		result.ValidationTrace = append(result.ValidationTrace, "Failed: Architecture mismatch. Node is "+node.Identity.Architecture)
	}

	if spec.ImageReference != "" || spec.KernelImagePath != "" {
		result.Warnings = append(result.Warnings, "WARN_DEPRECATED_IMAGE_REFERENCE")
		if len(spec.Storage) > 0 {
			result.ValidationTrace = append(result.ValidationTrace, "Warning: Both typed Storage and legacy ImageReference provided. Typed Storage takes precedence.")
		} else {
			result.ValidationTrace = append(result.ValidationTrace, "Warning: Using deprecated legacy artifact fields. Please migrate to typed Storage array.")
		}
	}

	if len(spec.NetworkAttachments) > 0 {
		result.Warnings = append(result.Warnings, "WARN_DEPRECATED_NETWORK_ATTACHMENTS")
		result.ValidationTrace = append(result.ValidationTrace, "Warning: Using deprecated NetworkAttachments field. Please migrate to typed Networks array.")
	}

	if !result.IsValid {
		result.ExplainabilityText = "Node cannot launch workload due to generic preflight blockers."
		result.RemediationHints = generateRemediationHints(result)
		return result
	}

	// 2. Layer 2 & 3: Backend-specific capability and artifact checks via Runtime Selector
	selectedBackend, rejectedBackends := SelectBackend(spec, node)
	result.RejectedBackends = rejectedBackends
	result.SelectedBackend = selectedBackend

	if selectedBackend == "" {
		result.IsValid = false
		result.BlockingReasonCodes = append(result.BlockingReasonCodes, "ERR_LAUNCH_BACKEND_NOT_SUPPORTED")
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

	if selectedBackend == "kvm_qemu" {
		result.RecommendedRuntime = "qemu-system-" + spec.Architecture
	} else if selectedBackend == "cloud_hypervisor" {
		result.RecommendedRuntime = "cloud-hypervisor"
	} else if selectedBackend == "firecracker" {
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
		if code == "ERR_LAUNCH_ARCH_MISMATCH" {
			hints["architecture"] = "Ensure the requested architecture matches the node architecture."
		}
		if code == "ERR_LAUNCH_BACKEND_NOT_SUPPORTED" {
			hints["backend"] = "Check the RejectedBackends map for specific missing capabilities."
		}
	}

	for backend, reason := range result.RejectedBackends {
		if backend == "kvm_qemu" && strings.Contains(reason, "CAP_QEMU_BINARY_MISSING") {
			hints["kvm_qemu_binary"] = "Install qemu-system-x86_64 or qemu-system-aarch64 on the host."
		}
		if backend == "kvm_qemu" && strings.Contains(reason, "CAP_KVM_MISSING") {
			hints["kvm_qemu_kvm"] = "Enable KVM in BIOS or load kvm kernel modules."
		}
		if backend == "kvm_qemu" && strings.Contains(reason, "CAP_KVM_NOT_OPENABLE_PERMS") {
			hints["kvm_qemu_perms"] = "Ensure the Schedune agent has rw permissions to /dev/kvm."
		}
		if backend == "cloud_hypervisor" && strings.Contains(reason, "CAP_CLOUDHYPERVISOR_BINARY_MISSING") {
			hints["cloud_hypervisor_binary"] = "Install cloud-hypervisor binary on the host."
		}
		if backend == "firecracker" && strings.Contains(reason, "CAP_FIRECRACKER_PREREQS_MISSING") {
			hints["firecracker_prereqs"] = "Ensure firecracker binary, /dev/net/tun, and cgroups v2 are available."
		}
		if strings.Contains(reason, "ERR_LAUNCH_MISSING_ARTIFACT") {
			hints["artifact_missing"] = "Ensure ImageReference is provided for the workload."
		}
		if strings.Contains(reason, "ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL") {
			hints["firecracker_artifact"] = "Ensure KernelImagePath and RootfsPath are provided for MicroVMs."
		}
	}

	return hints
}
