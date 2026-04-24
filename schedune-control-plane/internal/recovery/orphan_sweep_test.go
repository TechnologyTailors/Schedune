package recovery

import (
	"context"
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

type MockEnumerator struct {
	Procs []runtime.EnumeratedProcess
}

func (m *MockEnumerator) Enumerate() ([]runtime.EnumeratedProcess, error) {
	return m.Procs, nil
}

type MockOrphanStore struct {
	Orphans map[string]domain.OrphanRecord
}

func (m *MockOrphanStore) UpsertOrphan(ctx context.Context, rec domain.OrphanRecord) error {
	m.Orphans[rec.OrphanID] = rec
	return nil
}

func (m *MockOrphanStore) GetOrphan(ctx context.Context, orphanID string) (domain.OrphanRecord, bool, error) {
	rec, ok := m.Orphans[orphanID]
	return rec, ok, nil
}

func (m *MockOrphanStore) ListOrphans(ctx context.Context, filter domain.OrphanFilter) ([]domain.OrphanRecord, error) {
	return nil, nil
}

func (m *MockOrphanStore) MarkMissingAsDisappeared(ctx context.Context, observedOrphanIDs []string, nowSec int64) error {
	return nil
}

func TestSweepOnce_CreatesOrphans(t *testing.T) {
	enumerator := &MockEnumerator{
		Procs: []runtime.EnumeratedProcess{
			{PID: 100, Backend: "cloud_hypervisor", CommandFingerprint: "f1"},
		},
	}
	execStore := &MockExecStore{} // Empty execs
	orphanStore := &MockOrphanStore{Orphans: make(map[string]domain.OrphanRecord)}
	eventStore := &MockEventStore{}

	service := NewOrphanSweepService(enumerator, execStore, orphanStore, eventStore, "test-node")
	err := service.SweepOnce(context.Background())
	if err != nil {
		t.Fatalf("sweep failed: %v", err)
	}

	if len(orphanStore.Orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphanStore.Orphans))
	}
	
	// Should generate an event
	if len(eventStore.Events) != 1 || eventStore.Events[0].EventType != "OrphanDetected" {
		t.Errorf("expected OrphanDetected event")
	}
}

func TestSweepOnce_IgnoresMatched(t *testing.T) {
	enumerator := &MockEnumerator{
		Procs: []runtime.EnumeratedProcess{
			{PID: 1234, Backend: "kvm_qemu", ExecutionIDHint: "exec-1", CommandFingerprint: "f2"},
		},
	}
	execStore := &MockExecStore{
		Execs: []launch.LaunchExecutionRecord{
			{ExecutionID: "exec-1", PID: func(i int) *int { return &i }(1234), PreparedState: &launch.PreparedLaunch{RuntimeBackend: "kvm_qemu"}},
		},
	}
	orphanStore := &MockOrphanStore{Orphans: make(map[string]domain.OrphanRecord)}
	eventStore := &MockEventStore{}

	service := NewOrphanSweepService(enumerator, execStore, orphanStore, eventStore, "test-node")
	err := service.SweepOnce(context.Background())
	if err != nil {
		t.Fatalf("sweep failed: %v", err)
	}

	if len(orphanStore.Orphans) != 0 {
		t.Errorf("expected 0 orphans for matched process, got %d", len(orphanStore.Orphans))
	}
}
