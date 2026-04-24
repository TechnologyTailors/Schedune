package recovery

import (
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

func TestReassociateExecution_ExactMatch(t *testing.T) {
	engine := NewReassociationEngine()
	pid := 1234
	rec := launch.LaunchExecutionRecord{PID: &pid}

	observed := []runtime.ProcessIdentity{
		{PID: 1234, Backend: "kvm"},
		{PID: 5678, Backend: "kvm"},
	}

	res := engine.ReassociateExecution(rec, observed)

	if !res.Matched {
		t.Errorf("expected matched to be true")
	}
	if res.MatchedPID == nil || *res.MatchedPID != 1234 {
		t.Errorf("expected MatchedPID 1234, got %v", res.MatchedPID)
	}
}

func TestReassociateExecution_Ambiguous(t *testing.T) {
	engine := NewReassociationEngine()
	pid := 1234
	rec := launch.LaunchExecutionRecord{PID: &pid}

	observed := []runtime.ProcessIdentity{
		{PID: 1234, Backend: "kvm"},
		{PID: 1234, Backend: "cloud_hypervisor"}, // PID reuse scenario
	}

	res := engine.ReassociateExecution(rec, observed)

	if !res.Ambiguous {
		t.Errorf("expected ambiguous to be true")
	}
	if res.ReasonCode != "ERR_RECOVERY_REASSOCIATION_AMBIGUOUS" {
		t.Errorf("expected ERR_RECOVERY_REASSOCIATION_AMBIGUOUS, got %s", res.ReasonCode)
	}
}

func TestReassociateExecution_Missing(t *testing.T) {
	engine := NewReassociationEngine()
	pid := 1234
	rec := launch.LaunchExecutionRecord{PID: &pid}

	observed := []runtime.ProcessIdentity{
		{PID: 5678, Backend: "kvm"},
	}

	res := engine.ReassociateExecution(rec, observed)

	if !res.Missing {
		t.Errorf("expected missing to be true")
	}
}
