package lifecycle

import (
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime/inspect"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

type MockInspector struct {
	Obs inspect.RuntimeObservation
	Err error
}

func (m *MockInspector) Inspect(executionID string, pid *int, prepared launch.PreparedLaunch) (inspect.RuntimeObservation, error) {
	return m.Obs, m.Err
}

func TestReconcile_StartingToRunning(t *testing.T) {
	pid := 1234
	startedAt := time.Now().Unix() - 10

	rec := launch.LaunchExecutionRecord{
		State:        launch.StateStarting,
		PID:          &pid,
		StartedAtSec: &startedAt,
	}

	inspector := &MockInspector{
		Obs: inspect.RuntimeObservation{
			ProcessExists:       true,
			BackendReadySignal:  true,
			BackendSignalSource: "qmp_socket_dial_ok",
		},
	}

	err := Reconcile(&rec, inspector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.State != launch.StateRunning {
		t.Errorf("expected Running, got %s", rec.State)
	}
	if rec.RuntimeLiveness != "Alive" || rec.RuntimeReadiness != "Ready" {
		t.Errorf("expected Alive/Ready, got %s/%s", rec.RuntimeLiveness, rec.RuntimeReadiness)
	}
}

func TestReconcile_StartingToFailed_Timeout(t *testing.T) {
	pid := 1234
	startedAt := time.Now().Unix() - 40 // Past 15s timeout

	rec := launch.LaunchExecutionRecord{
		State:        launch.StateStarting,
		PID:          &pid,
		StartedAtSec: &startedAt,
	}

	inspector := &MockInspector{
		Obs: inspect.RuntimeObservation{
			ProcessExists:      true,
			BackendReadySignal: false, // Not ready yet
		},
	}

	err := Reconcile(&rec, inspector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.State != launch.StateFailed {
		t.Errorf("expected Failed, got %s", rec.State)
	}
	if rec.RuntimeReadiness != "Failed" {
		t.Errorf("expected readiness Failed, got %s", rec.RuntimeReadiness)
	}
}

func TestReconcile_RunningToExited(t *testing.T) {
	pid := 1234

	rec := launch.LaunchExecutionRecord{
		State: launch.StateRunning,
		PID:   &pid,
	}

	inspector := &MockInspector{
		Obs: inspect.RuntimeObservation{
			ProcessExists: false, // Process died
		},
	}

	err := Reconcile(&rec, inspector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.State != launch.StateExited {
		t.Errorf("expected Exited, got %s", rec.State)
	}
}

func TestReconcile_TerminatingToTerminated(t *testing.T) {
	pid := 1234

	rec := launch.LaunchExecutionRecord{
		State: launch.StateTerminating,
		PID:   &pid,
	}

	inspector := &MockInspector{
		Obs: inspect.RuntimeObservation{
			ProcessExists: false, // Process died as requested
		},
	}

	err := Reconcile(&rec, inspector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.State != launch.StateTerminated {
		t.Errorf("expected Terminated, got %s", rec.State)
	}
	if rec.RuntimeLiveness != "Dead" {
		t.Errorf("expected Dead, got %s", rec.RuntimeLiveness)
	}
	if rec.RuntimeReadiness != "NotReady" {
		t.Errorf("expected NotReady, got %s", rec.RuntimeReadiness)
	}
}
