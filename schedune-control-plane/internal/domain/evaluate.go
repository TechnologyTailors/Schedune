package domain

import (
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
	"time"
)

// EligibilityResult explains exactly why a node was accepted or rejected for a specific intent.
type EligibilityResult struct {
	NodeID             string   `json:"node_id"`
	IsEligible         bool     `json:"is_eligible"`
	
	// Hard Rejections (If any exist, IsEligible must be false)
	HardRejectionCodes []string `json:"hard_rejection_codes"`
	
	// Details for transparency
	FreshnessOK        bool     `json:"freshness_ok"`
	CompatibilityOK    bool     `json:"compatibility_ok"`
	HealthOK           bool     `json:"health_ok"`
	
	// Soft matching data for the scoring engine
	MatchedFeatures    []string `json:"matched_features"`
	Warnings           []string `json:"warnings"`
}

// Evaluate tests a NodeRecord against a WorkloadIntent and returns a pure EligibilityResult.
func Evaluate(intent workload.WorkloadIntent, node NodeRecord) EligibilityResult {
	result := EligibilityResult{
		NodeID:             node.ID,
		IsEligible:         true,
		HardRejectionCodes: []string{},
		MatchedFeatures:    []string{},
		Warnings:           []string{},
	}

	now := time.Now().Unix()

	// 1. Health Evaluation
	result.HealthOK = node.Health.State == "Healthy" || node.Health.State == "Warning"
	if !result.HealthOK {
		result.IsEligible = false
		result.HardRejectionCodes = append(result.HardRejectionCodes, "REJECT_NODE_UNHEALTHY")
	}

	// 2. Freshness Evaluation
	nodeAgeSec := now - node.Freshness.LastCollectionTime.Unix()
	result.FreshnessOK = nodeAgeSec <= intent.MaxTelemetryAgeSec && !node.Freshness.IsStale
	if !result.FreshnessOK {
		result.IsEligible = false
		result.HardRejectionCodes = append(result.HardRejectionCodes, "REJECT_TELEMETRY_STALE")
	}

	// 3. Architecture Evaluation
	if intent.RequiredArchitecture != "any" && intent.RequiredArchitecture != node.Identity.Architecture {
		result.IsEligible = false
		result.HardRejectionCodes = append(result.HardRejectionCodes, "REJECT_ARCHITECTURE_MISMATCH")
	}

	// 4. Compatibility Class Evaluation
	result.CompatibilityOK = false
	for _, reqClass := range intent.RequiredCompatibilityClasses {
		if reqClass == node.Compatibility.Class {
			result.CompatibilityOK = true
			break
		}
	}
	if !result.CompatibilityOK {
		result.IsEligible = false
		result.HardRejectionCodes = append(result.HardRejectionCodes, "REJECT_COMPATIBILITY_CLASS_MISMATCH")
	}

	// 5. Hard Feature Requirements (KVM, TPM)
	if intent.RequiresKVM {
		cap, exists := node.Capabilities["kvm_vm_launch"]
		if !exists || cap.State != "Supported" {
			result.IsEligible = false
			result.HardRejectionCodes = append(result.HardRejectionCodes, "REJECT_MISSING_KVM")
		}
	}

	if intent.RequiresTPM {
		cap, exists := node.Capabilities["hardware_tpm"]
		if !exists || cap.State != "Supported" {
			result.IsEligible = false
			result.HardRejectionCodes = append(result.HardRejectionCodes, "REJECT_MISSING_TPM")
		}
	}

	// 6. Forbidden Constraints
	for _, forbidden := range intent.ForbiddenConstraints {
		for code := range node.Constraints {
			if forbidden == code {
				result.IsEligible = false
				result.HardRejectionCodes = append(result.HardRejectionCodes, "REJECT_FORBIDDEN_CONSTRAINT_"+code)
			}
		}
	}

	// 7. Soft Preferences (Gathering data for the Scorer)
	for _, pref := range intent.PreferredFeatures {
		if cap, exists := node.Capabilities[pref]; exists && cap.State == "Supported" {
			result.MatchedFeatures = append(result.MatchedFeatures, pref)
		}
	}

	if len(node.Health.ActiveAlarms) > 0 {
		result.Warnings = append(result.Warnings, "Node has active operational alarms")
	}

	return result
}
