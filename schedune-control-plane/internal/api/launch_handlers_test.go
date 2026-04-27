package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime/inspect"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store/sqlite"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/gin-gonic/gin"
)

type MockExecutor struct {
	ShouldFailPrepare bool
	PrepareCalled     bool
	ExecuteCalled     bool
	PreparedLaunch    launch.PreparedLaunch
}

func (m *MockExecutor) Prepare(spec launch.LaunchSpec) (launch.PreparedLaunch, error) {
	m.PrepareCalled = true
	if m.ShouldFailPrepare {
		return launch.PreparedLaunch{}, errors.New("mock prepare failure")
	}
	return m.PreparedLaunch, nil
}

func (m *MockExecutor) Execute(prepared launch.PreparedLaunch) (int, error) {
	m.ExecuteCalled = true
	return 1234, nil
}

func (m *MockExecutor) Terminate(pid int) error {
	return nil
}

func validBaseSpec(nodeID string) launch.LaunchSpec {
	return launch.LaunchSpec{
		SchemaVersion: "v1alpha1",
		WorkloadID:    "wl-test-" + nodeID,
		TenantID:      "tenant-test",
		NodeID:        nodeID,
		RuntimeClass:  "VirtualMachine",
		Architecture:  "x86_64",
		Vcpu:          2,
		MemoryMB:      1024,
		LaunchMode:    "DryRun", // This will be the default for the tests
		Storage: []launch.StorageAttachmentSpec{
			{HostPath: "/tmp/test.qcow2", Format: "qcow2"},
		},
	}
}

func setupDryRunTestRouter(executor *MockExecutor, node domain.NodeRecord) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	nodeStore := store.NewInMemoryStore()
	env := schema.SchedulerEnvelope{
		NodeID:       node.ID,
		TimestampSec: time.Now().Unix(),
	}
	_ = nodeStore.SaveNodeState(env, node)

	execStore, _ := sqlite.NewSQLiteStore(":memory:")
	resolver := &StaticExecutorResolver{
		executors: map[string]runtime.Executor{
			"kvm_qemu":         executor,
			"cloud_hypervisor": &runtime.CloudHypervisorExecutor{},
			"firecracker":      &runtime.FirecrackerExecutor{},
		},
	}
	orch := domain.NewLaunchOrchestrator(nodeStore, execStore, execStore, resolver)

	handler := &LaunchHandler{
		nodeStore: nodeStore,
		execStore: execStore,
		resolver:  resolver,
		orch:      orch,
	}

	router.POST("/launch/dry-run", handler.DryRunLaunch)

	return router
}

func TestDryRunLaunch_Success(t *testing.T) {
	mockExecutor := &MockExecutor{
		PreparedLaunch: launch.PreparedLaunch{
			RuntimeBackend: "kvm_qemu",
			MemoryMB:       1024,
			Vcpu:           2,
		},
	}
	node := domain.NodeRecord{
		ID: "test-node-ok",
		Identity: domain.NodeIdentity{
			Architecture: "x86_64",
		},
		Capabilities: map[string]domain.NodeCapabilityRecord{
			"kvm_vm_launch":       {State: "Supported"},
			"qemu_binary_present": {State: "Supported"},
		},
	}
	router := setupDryRunTestRouter(mockExecutor, node)

	spec := validBaseSpec("test-node-ok")
	body, _ := json.Marshal(spec)
	req, _ := http.NewRequest("POST", "/launch/dry-run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res launch.LaunchDryRunResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !res.Validation.IsValid {
		t.Errorf("expected validation to be valid, got false")
	}

	if !mockExecutor.PrepareCalled {
		t.Errorf("expected Prepare to be called, but it was not")
	}

	if mockExecutor.ExecuteCalled {
		t.Errorf("expected Execute NOT to be called, but it was")
	}

	if res.PreparedLaunch == nil {
		t.Fatal("expected prepared launch to be non-nil")
	}

	if res.PreparedLaunch.MemoryMB != 1024 {
		t.Errorf("expected prepared memory to be 1024, got %d", res.PreparedLaunch.MemoryMB)
	}
}

func TestDryRunLaunch_ValidationFails(t *testing.T) {
	mockExecutor := &MockExecutor{}
	node := domain.NodeRecord{
		ID: "test-node-invalid",
		Identity: domain.NodeIdentity{
			Architecture: "arm64", // Mismatch
		},
		Capabilities: map[string]domain.NodeCapabilityRecord{},
	}
	router := setupDryRunTestRouter(mockExecutor, node)

	spec := validBaseSpec("test-node-invalid")
	body, _ := json.Marshal(spec)
	req, _ := http.NewRequest("POST", "/launch/dry-run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res launch.LaunchDryRunResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if res.Validation.IsValid {
		t.Errorf("expected validation to be invalid, got true")
	}

	if mockExecutor.PrepareCalled {
		t.Errorf("expected Prepare not to be called, but it was")
	}

	if mockExecutor.ExecuteCalled {
		t.Errorf("expected Execute NOT to be called, but it was")
	}
}

func TestDryRunLaunch_PreparationFails(t *testing.T) {
	mockExecutor := &MockExecutor{
		ShouldFailPrepare: true,
	}
	node := domain.NodeRecord{
		ID: "test-node-prep-fail",
		Identity: domain.NodeIdentity{
			Architecture: "x86_64",
		},
		Capabilities: map[string]domain.NodeCapabilityRecord{
			"kvm_vm_launch":       {State: "Supported"},
			"qemu_binary_present": {State: "Supported"},
		},
	}
	router := setupDryRunTestRouter(mockExecutor, node)

	spec := validBaseSpec("test-node-prep-fail")
	body, _ := json.Marshal(spec)
	req, _ := http.NewRequest("POST", "/launch/dry-run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res launch.LaunchDryRunResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !res.Validation.IsValid {
		t.Errorf("expected validation to be valid, got false")
	}

	if !mockExecutor.PrepareCalled {
		t.Errorf("expected Prepare to be called, but it was not")
	}

	if mockExecutor.ExecuteCalled {
		t.Errorf("expected Execute NOT to be called, but it was")
	}

	if res.PreparedLaunch != nil {
		t.Error("expected prepared launch to be nil")
	}

	if res.PreparationError == nil || *res.PreparationError != "mock prepare failure" {
		t.Errorf("expected preparation error, got %v", res.PreparationError)
	}

	if res.PreparationReasonCode == nil || *res.PreparationReasonCode != schema.ReasonErrPreparationFailed {
		t.Errorf("expected preparation reason code, got %v", res.PreparationReasonCode)
	}
}

type MockInspector struct {
	Obs inspect.RuntimeObservation
	Err error
}

func (m *MockInspector) Inspect(executionID string, pid *int, prepared launch.PreparedLaunch) (inspect.RuntimeObservation, error) {
	return m.Obs, m.Err
}

type MockInspectorResolver struct {
	Inspector inspect.Inspector
}

func (m *MockInspectorResolver) Resolve(backend string) inspect.Inspector {
	return m.Inspector
}

func setupInspectTestRouter(t *testing.T) (*gin.Engine, *store.InMemoryStore, *sqlite.SQLiteStore, *MockInspector) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	nodeStore := store.NewInMemoryStore()
	execStore, _ := sqlite.NewSQLiteStore(":memory:")

	resolver := &StaticExecutorResolver{}
	orch := domain.NewLaunchOrchestrator(nodeStore, execStore, execStore, resolver)

	mockInspector := &MockInspector{}
	mockResolver := &MockInspectorResolver{Inspector: mockInspector}

	handler := &LaunchHandler{
		nodeStore:         nodeStore,
		execStore:         execStore,
		resolver:          resolver,
		inspectorResolver: mockResolver,
		orch:              orch,
	}

	router.GET("/launch/:id", handler.InspectLaunch)
	router.GET("/launch/:id/readiness", handler.InspectReadiness)

	return router, nodeStore, execStore, mockInspector
}

func TestInspectReadiness_ReconcilesToReady(t *testing.T) {
	router, _, execStore, mockInspector := setupInspectTestRouter(t)

	// Insert a Starting execution
	pid := 1234
	now := time.Now().Unix()
	rec := launch.LaunchExecutionRecord{
		ExecutionID:  "test-exec-1",
		NodeID:       "test-node",
		State:        launch.StateStarting,
		PID:          &pid,
		StartedAtSec: &now,
	}
	execStore.SaveExecution(context.Background(), rec)

	// Mock observation: Process is alive and ready
	mockInspector.Obs = inspect.RuntimeObservation{
		ProcessExists:       true,
		BackendReadySignal:  true,
		BackendSignalSource: "mock-backend",
		ObservedAtSec:       now,
	}

	// Request readiness
	req, _ := http.NewRequest("GET", "/launch/test-exec-1/readiness", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res launch.LaunchExecutionRecord
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if res.State != launch.StateRunning {
		t.Errorf("expected state to be Running, got %s", res.State)
	}
	if res.RuntimeReadiness != "Ready" {
		t.Errorf("expected readiness to be Ready, got %s", res.RuntimeReadiness)
	}

	// Verify it was persisted
	persisted, _, err := execStore.GetExecution(context.Background(), "test-exec-1")
	if err != nil {
		t.Fatalf("failed to get execution: %v", err)
	}
	if persisted.State != launch.StateRunning {
		t.Errorf("expected persisted state to be Running, got %s", persisted.State)
	}

	events, _ := execStore.ListEvents(context.Background(), "test-exec-1")
	hasReconcileEvent := false
	for _, ev := range events {
		if ev.EventType == launch.EventTypeReconcileStateChanged {
			hasReconcileEvent = true
			break
		}
	}
	if !hasReconcileEvent {
		t.Errorf("expected EventTypeReconcileStateChanged to be emitted")
	}
}

func TestInspectReadiness_MaterialSignalChange(t *testing.T) {
	router, _, execStore, mockInspector := setupInspectTestRouter(t)

	pid := 1234
	now := time.Now().Unix()

	// Create a record that is already running and ready, but with no signal data
	rec := launch.LaunchExecutionRecord{
		ExecutionID:      "test-exec-signal",
		NodeID:           "test-node",
		State:            launch.StateRunning,
		RuntimeLiveness:  "Alive",
		RuntimeReadiness: "Ready",
		PID:              &pid,
		StartedAtSec:     &now,
		ReadinessSignal: &launch.ReadinessSignalSummary{
			ControlSocketExists: true,
			ControlSocketDialOK: true,
		},
	}
	execStore.SaveExecution(context.Background(), rec)

	// Mock observation: Process is alive and ready, but signal changed (BackendReadySignal is now true)
	mockInspector.Obs = inspect.RuntimeObservation{
		ProcessExists:       true,
		BackendReadySignal:  true,
		BackendSignalSource: "mock-backend",
		ObservedAtSec:       now,
	}

	req, _ := http.NewRequest("GET", "/launch/test-exec-signal/readiness", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	events, _ := execStore.ListEvents(context.Background(), "test-exec-signal")
	hasReconcileEvent := false
	for _, ev := range events {
		if ev.EventType == launch.EventTypeReconcileStateChanged {
			hasReconcileEvent = true

			// verify payload has signal evidence
			payloadBytes, _ := json.Marshal(ev.PayloadJSON)
			var payload launch.EventPayloadReconcile
			json.Unmarshal(payloadBytes, &payload)

			if payload.Signal == nil {
				t.Errorf("expected signal in payload, got nil")
			} else if !payload.Signal.BackendReadySignal {
				t.Errorf("expected signal BackendReadySignal=true in payload")
			}
			break
		}
	}
	if !hasReconcileEvent {
		t.Errorf("expected EventTypeReconcileStateChanged to be emitted for material signal change")
	}
}
