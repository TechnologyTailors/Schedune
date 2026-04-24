package domain

import (
	"fmt"
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
	}

	// 1. Layer 1: Generic launch checks
	if spec.Architecture != node.Identity.Architecture {
		result.IsValid = false
		result.BlockingReasonCodes = append(result.BlockingReasonCodes, "ERR_LAUNCH_ARCH_MISMATCH")
		result.ValidationTrace = append(result.ValidationTrace, "Failed: Architecture mismatch. Node is " + node.Identity.Architecture)
	}

	if !result.IsValid {
		result.ExplainabilityText = "Node cannot launch workload due to generic preflight blockers."
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
		return result
	}

	// 3. Layer 4: Setup context for preparation phase validation
	result.ValidationTrace = append(result.ValidationTrace, "Passed: Selected backend " + selectedBackend)
	
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

	return result
}
