package domain

import (
	"fmt"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// ValidateLaunch assesses whether a chosen node is physically capable of executing the requested spec.
func ValidateLaunch(spec launch.LaunchSpec, node NodeRecord) launch.LaunchValidationResult {
	result := launch.LaunchValidationResult{
		IsValid:              true,
		BlockingReasonCodes:  []string{},
		Warnings:             []string{},
		RequiredHostFeatures: []string{},
		MissingHostFeatures:  []string{},
		ValidationTrace:      []string{"Started validation against node " + node.ID},
	}

	if spec.Architecture != node.Identity.Architecture {
		result.IsValid = false
		result.BlockingReasonCodes = append(result.BlockingReasonCodes, "ERR_LAUNCH_ARCH_MISMATCH")
		result.ValidationTrace = append(result.ValidationTrace, "Failed: Architecture mismatch. Node is " + node.Identity.Architecture)
	}

	if spec.RuntimeClass == "VirtualMachine" {
		result.RequiredHostFeatures = append(result.RequiredHostFeatures, "kvm_vm_launch")
	} else if spec.RuntimeClass == "MicroVM" {
		result.RequiredHostFeatures = append(result.RequiredHostFeatures, "firecracker_launch")
	}

	for _, req := range result.RequiredHostFeatures {
		cap, exists := node.Capabilities[req]
		if !exists || cap.State != "Supported" || cap.IsStale {
			result.IsValid = false
			result.MissingHostFeatures = append(result.MissingHostFeatures, req)
			result.BlockingReasonCodes = append(result.BlockingReasonCodes, "ERR_MISSING_RUNTIME_CAPABILITY_"+req)
			
			if !exists {
				result.ValidationTrace = append(result.ValidationTrace, "Failed: Capability " + req + " not discovered")
			} else if cap.IsStale {
				result.ValidationTrace = append(result.ValidationTrace, "Failed: Capability " + req + " is stale")
			} else {
				result.ValidationTrace = append(result.ValidationTrace, "Failed: Capability " + req + " is " + cap.State + " (" + cap.ReasonCode + ")")
			}
		} else {
			result.ValidationTrace = append(result.ValidationTrace, "Passed: Node has capability " + req)
		}
	}

	if !result.IsValid {
		result.ExplainabilityText = "Node cannot launch workload due to missing prerequisites or constraints."
	} else {
		result.ExplainabilityText = "Node is fully capable of executing this launch spec."
		if spec.RuntimeClass == "VirtualMachine" {
			result.RecommendedRuntime = "qemu-system-" + spec.Architecture
		} else {
			result.RecommendedRuntime = "firecracker"
		}
	}

	if spec.LaunchMode == "DryRun" && result.IsValid {
		result.ValidationTrace = append(result.ValidationTrace, fmt.Sprintf("DryRun: Successfully simulated allocation of %d vcpus.", spec.Vcpu))
	}

	return result
}
