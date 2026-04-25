package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
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
	orch := domain.NewLaunchOrchestrator(nodeStore, execStore, resolver)

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
