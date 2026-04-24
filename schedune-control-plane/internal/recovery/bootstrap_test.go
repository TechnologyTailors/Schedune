package recovery

import (
	"context"
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime/inspect"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

type MockExecStore struct {
	Execs []launch.LaunchExecutionRecord
	Saved []launch.LaunchExecutionRecord
}

func (m *MockExecStore) SaveExecution(ctx context.Context, rec launch.LaunchExecutionRecord) error {
	m.Saved = append(m.Saved, rec)
	return nil
}

func (m *MockExecStore) GetExecution(ctx context.Context, executionID string) (launch.LaunchExecutionRecord, bool, error) {
	return launch.LaunchExecutionRecord{}, false, nil
}

func (m *MockExecStore) ListActiveExecutions(ctx context.Context) ([]launch.LaunchExecutionRecord, error) {
	return m.Execs, nil
}

func (m *MockExecStore) ListRecoverableExecutions(ctx context.Context) ([]launch.LaunchExecutionRecord, error) {
	return m.Execs, nil
}

type MockEventStore struct {
	Events []launch.RuntimeEvent
}

func (m *MockEventStore) AppendEvent(ctx context.Context, ev launch.RuntimeEvent) error {
	m.Events = append(m.Events, ev)
	return nil
}

func (m *MockEventStore) ListEvents(ctx context.Context, executionID string) ([]launch.RuntimeEvent, error) {
	return nil, nil
}

type MockInspector struct {
	Obs inspect.RuntimeObservation
	Err error
}

func (m *MockInspector) Inspect(executionID string, pid *int, prepared launch.PreparedLaunch) (inspect.RuntimeObservation, error) {
	return m.Obs, m.Err
}

func TestBootstrap_RunningExecutionConfirmed(t *testing.T) {
	pid := 1234
	execStore := &MockExecStore{
		Execs: []launch.LaunchExecutionRecord{
			{ExecutionID: "exec-1", State: launch.StateRunning, PID: &pid},
		},
	}
	eventStore := &MockEventStore{}
	inspector := &MockInspector{
		Obs: inspect.RuntimeObservation{ProcessExists: true},
	}

	bootstrapper := NewRecoveryBootstrapper(execStore, eventStore, inspector)
	err := bootstrapper.Bootstrap(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(execStore.Saved) != 1 {
		t.Fatalf("expected 1 saved execution, got %d", len(execStore.Saved))
	}

	rec := execStore.Saved[0]
	if rec.State != launch.StateRunning {
		t.Errorf("expected state Running, got %s", rec.State)
	}
	if rec.RuntimeLiveness != "Alive" {
		t.Errorf("expected liveness Alive, got %s", rec.RuntimeLiveness)
	}

	if len(eventStore.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.Events))
	}
	if eventStore.Events[0].EventType != "ExecutionRehydrated" {
		t.Errorf("expected ExecutionRehydrated event, got %s", eventStore.Events[0].EventType)
	}
}

func TestBootstrap_RunningExecutionMissingProcess(t *testing.T) {
	pid := 1234
	execStore := &MockExecStore{
		Execs: []launch.LaunchExecutionRecord{
			{ExecutionID: "exec-2", State: launch.StateRunning, PID: &pid},
		},
	}
	eventStore := &MockEventStore{}
	inspector := &MockInspector{
		Obs: inspect.RuntimeObservation{ProcessExists: false}, // Process died during crash
	}

	bootstrapper := NewRecoveryBootstrapper(execStore, eventStore, inspector)
	err := bootstrapper.Bootstrap(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := execStore.Saved[0]
	if rec.State != launch.StateUnknown {
		t.Errorf("expected state Unknown, got %s", rec.State)
	}

	if eventStore.Events[0].ReasonCode != "ERR_RECOVERY_EXECUTION_MISSING" {
		t.Errorf("expected ERR_RECOVERY_EXECUTION_MISSING, got %s", eventStore.Events[0].ReasonCode)
	}
}

func TestBootstrap_TerminatingExecutionMissing(t *testing.T) {
	pid := 1234
	execStore := &MockExecStore{
		Execs: []launch.LaunchExecutionRecord{
			{ExecutionID: "exec-3", State: launch.StateTerminating, PID: &pid},
		},
	}
	eventStore := &MockEventStore{}
	inspector := &MockInspector{
		Obs: inspect.RuntimeObservation{ProcessExists: false}, // Process finally exited
	}

	bootstrapper := NewRecoveryBootstrapper(execStore, eventStore, inspector)
	err := bootstrapper.Bootstrap(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := execStore.Saved[0]
	if rec.State != launch.StateTerminated {
		t.Errorf("expected state Terminated, got %s", rec.State)
	}
}

func TestBootstrap_StaleHandleMidFlight(t *testing.T) {
	execStore := &MockExecStore{
		Execs: []launch.LaunchExecutionRecord{
			{ExecutionID: "exec-4", State: launch.StateLaunching, PID: nil}, // Crash before PID assigned
		},
	}
	eventStore := &MockEventStore{}
	inspector := &MockInspector{}

	bootstrapper := NewRecoveryBootstrapper(execStore, eventStore, inspector)
	err := bootstrapper.Bootstrap(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := execStore.Saved[0]
	if rec.State != launch.StateUnknown {
		t.Errorf("expected state Unknown, got %s", rec.State)
	}
	if eventStore.Events[0].ReasonCode != "ERR_RECOVERY_STALE_HANDLE" {
		t.Errorf("expected ERR_RECOVERY_STALE_HANDLE, got %s", eventStore.Events[0].ReasonCode)
	}
}
