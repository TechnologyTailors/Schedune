package domain

// EligibilityEngine answers specific workload orchestration questions based on the NodeRecord.
// It DOES NOT place workloads. It only answers questions about the node's eligibility.
type EligibilityEngine struct {
	node NodeRecord
}

func NewEligibilityEngine(node NodeRecord) *EligibilityEngine {
	return &EligibilityEngine{node: node}
}

func (e *EligibilityEngine) CanRunKVMVMs() bool {
	cap, exists := e.node.Capabilities["kvm_virtualization"]
	return exists && cap.State == "Supported" && !cap.IsStale
}

func (e *EligibilityEngine) CanRunFirecrackerMicroVMs() bool {
	// Firecracker requires KVM support
	return e.CanRunKVMVMs()
}

func (e *EligibilityEngine) CanJoinArmProductionPool() bool {
	return e.node.Compatibility.Class == "ArmProduction" && e.node.Health.State == "Healthy" && !e.node.Freshness.IsStale
}

func (e *EligibilityEngine) IsOnlyEligibleForHoldingPool() bool {
	return e.node.Compatibility.Class == "X86HoldingPool" && e.node.Health.State == "Healthy"
}

func (e *EligibilityEngine) IsFreshEnoughToTrust() bool {
	return !e.node.Freshness.IsStale
}

func (e *EligibilityEngine) IsHealthyButPolicyIneligible() bool {
	// e.g. Node hardware is fine, OS is fine, but it lacks KVM or has the wrong arch for production workloads
	return e.node.Health.State == "Healthy" && e.node.Compatibility.Class == "Unsupported"
}
