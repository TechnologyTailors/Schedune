package domain

import (
	"context"
	"errors"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"testing"
	"time"
)

type MockStore struct {
	node NodeRecord
	exec map[string]launch.LaunchExecutionRecord
}

func (m *MockStore) GetNode(id string) (NodeRecord, error) {
	return m.node, nil
}
func (m *MockStore) SaveExecution(ctx context.Context, rec launch.LaunchExecutionRecord) error {
	m.exec[rec.ExecutionID] = rec
	return nil
}
func (m *MockStore) GetExecution(ctx context.Context, id string) (launch.LaunchExecutionRecord, bool, error) {
	rec, ok := m.exec[id]
	return rec, ok, nil
}

type MockExecutor struct {
	ShouldFailPrepare bool
	ShouldFailExecute bool
}

func (m *MockExecutor) Resolve(backend string) (runtime.Executor, error) {
	return m, nil
}

func (m *MockExecutor) Prepare(spec launch.LaunchSpec) (launch.PreparedLaunch, error) {
	if m.ShouldFailPrepare {
		return launch.PreparedLaunch{}, errors.New("failed prepare")
	}
	return launch.PreparedLaunch{}, nil
}

func (m *MockExecutor) Execute(prep launch.PreparedLaunch) (int, error) {
	if m.ShouldFailExecute {
		return 0, errors.New("failed execute")
	}
	return 1234, nil
}

func (m *MockExecutor) Terminate(pid int) error {
	return nil
}

func TestLaunchOrchestrator_Success(t *testing.T) {
	env := readFixture(t, "healthy_arm_production.json")
	now := time.Now().Unix()
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	store := &MockStore{
		node: node,
		exec: make(map[string]launch.LaunchExecutionRecord),
	}
	exec := &MockExecutor{}

	orch := NewLaunchOrchestrator(store, store, exec)

	spec := launch.LaunchSpec{
		SchemaVersion:  "v1alpha1",
		WorkloadID:     "wl-test",
		TenantID:       "tenant-test",
		NodeID:         node.ID,
		RuntimeClass:   "VirtualMachine",
		Architecture:   "aarch64",
		ImageReference: "dummy",
		Vcpu:           2,
		MemoryMB:       1024,
		LaunchMode:     "Execute",
	}

	rec := orch.StartLaunch(spec)

	if rec.State != launch.StateStarting {
		t.Errorf("expected state %s, got %s", launch.StateStarting, rec.State)
	}

	if rec.PID == nil || *rec.PID != 1234 {
		t.Errorf("expected PID 1234, got %v", rec.PID)
	}

	// Verify trace
	hasSpawnSuccess := false
	for _, tr := range rec.Trace {
		if tr.Stage == "StateTransition" && tr.ReasonCode == "" && tr.Message == "Process successfully spawned (PID 1234)" {
			hasSpawnSuccess = true
		}
	}
	if !hasSpawnSuccess {
		t.Errorf("expected trace to have RuntimeSpawn Success, got %v", rec.Trace)
	}
}

func TestLaunchOrchestrator_ValidationFails(t *testing.T) {
	env := readFixture(t, "missing_kvm_x86.json") // Node without KVM
	now := time.Now().Unix()
	for i := range env.Capabilities {
		env.Capabilities[i].ObservedAtSec = now
		staleAfter := now + 300
		env.Capabilities[i].StaleAfterSec = &staleAfter
	}
	node := ProjectEnvelope(env)

	store := &MockStore{
		node: node,
		exec: make(map[string]launch.LaunchExecutionRecord),
	}
	exec := &MockExecutor{}

	orch := NewLaunchOrchestrator(store, store, exec)

	spec := launch.LaunchSpec{
		SchemaVersion:  "v1alpha1",
		WorkloadID:     "wl-test",
		TenantID:       "tenant-test",
		NodeID:         node.ID,
		RuntimeClass:   "VirtualMachine",
		Architecture:   "x86_64", // matching arch to force KVM validation fail
		ImageReference: "dummy",
		Vcpu:           2,
		MemoryMB:       1024,
		LaunchMode:     "Execute",
	}

	rec := orch.StartLaunch(spec)

	if rec.State != launch.StateFailed {
		t.Errorf("expected state %s, got %s", launch.StateFailed, rec.State)
	}

	hasValFailed := false
	for _, tr := range rec.Trace {
		if tr.Stage == "StateTransition" && tr.ReasonCode == "ERR_VALIDATION_FAILED" {
			hasValFailed = true
		}
	}
	if !hasValFailed {
		t.Errorf("expected trace to have HostPreflight Failed, got %v", rec.Trace)
	}
}
