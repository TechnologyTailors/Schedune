package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/plan"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
	"github.com/gin-gonic/gin"
)

func setupPlanTestRouter(nodeStore *store.InMemoryStore, mockExec *MockExecutor) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewPlanHandler(nodeStore)
	handler.resolver = &StaticExecutorResolver{
		executors: map[string]runtime.Executor{
			"kvm_qemu": mockExec,
		},
	}
	router.POST("/plan/launch", handler.PlanLaunch)
	return router
}

func basePlanRequest(mode string, nodeID string) plan.LaunchPlanRequest {
	return plan.LaunchPlanRequest{
		SchemaVersion: "v1alpha1",
		Mode:          mode,
		TargetNodeID:  nodeID,
		WorkloadIntent: workload.WorkloadIntent{
			SchemaVersion:                "v1alpha1",
			WorkloadID:                   "wl-1",
			TenantID:                     "tenant-1",
			RuntimeClass:                 "VirtualMachine",
			RequiredArchitecture:         "x86_64",
			MaxTelemetryAgeSec:           300,
			RequiredCompatibilityClasses: []string{"kvm_standard"},
		},
		LaunchTemplate: launch.LaunchSpec{
			SchemaVersion: "v1alpha1",
			WorkloadID:    "wl-1",
			TenantID:      "tenant-1",
			NodeID:        "placeholder", // Hydrated by plan
			RuntimeClass:  "VirtualMachine",
			Architecture:  "x86_64",
			Vcpu:          2,
			MemoryMB:      1024,
			LaunchMode:    "Validate", // Hydrated by plan
			Storage: []launch.StorageAttachmentSpec{
				{HostPath: "/tmp/test.qcow2", Format: "qcow2"},
			},
		},
	}
}

func TestPlanLaunch_NoEligibleNode(t *testing.T) {
	nodeStore := store.NewInMemoryStore()
	mockExec := &MockExecutor{}
	router := setupPlanTestRouter(nodeStore, mockExec)

	reqData := basePlanRequest("validate", "")
	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest("POST", "/plan/launch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res plan.LaunchPlanResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if res.Status != plan.PlanStatusRejected {
		t.Errorf("expected Rejected status, got %s", res.Status)
	}
	if res.PlannedLaunchSpec != nil {
		t.Error("expected planned_launch_spec to be nil")
	}

	// Test empty-array JSON encoding
	if res.RejectedNodes == nil {
		t.Error("expected RejectedNodes to be non-nil empty array")
	}

	// Ensure JSON specifically encoded as `[]` instead of `null`
	if !bytes.Contains(w.Body.Bytes(), []byte(`"rejected_nodes":[]`)) {
		t.Errorf("expected rejected_nodes to be serialized as empty array [], got: %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"warnings":[`)) {
		t.Errorf("expected warnings to be serialized as array, got: %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"next_actions":[`)) {
		t.Errorf("expected next_actions to be serialized as array, got: %s", w.Body.String())
	}
}

func TestPlanLaunch_ValidationFailure(t *testing.T) {
	nodeStore := store.NewInMemoryStore()
	// Node without required qemu binary capability
	node := domain.NodeRecord{
		ID:            "node-1",
		Identity:      domain.NodeIdentity{Architecture: "x86_64"},
		Compatibility: domain.NodeCompatibilityRecord{Class: "kvm_standard"},
		Health:        domain.NodeHealthSummary{State: "Healthy"},
		Freshness:     domain.NodeFreshnessRecord{LastCollectionTime: time.Now()},
		Capabilities: map[string]domain.NodeCapabilityRecord{
			"kvm_vm_launch": {State: "Supported"},
			// missing qemu_binary_present
		},
	}
	nodeStore.SaveNodeState(schema.SchedulerEnvelope{NodeID: "node-1"}, node)

	mockExec := &MockExecutor{}
	router := setupPlanTestRouter(nodeStore, mockExec)

	reqData := basePlanRequest("validate", "")
	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest("POST", "/plan/launch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var res plan.LaunchPlanResult
	json.Unmarshal(w.Body.Bytes(), &res)

	if res.Status != plan.PlanStatusValidationFail {
		t.Errorf("expected ValidationFail status, got %s", res.Status)
	}
	if res.ValidationResult == nil || res.ValidationResult.IsValid {
		t.Error("expected validation result to be invalid")
	}
	if mockExec.PrepareCalled {
		t.Error("expected Prepare NOT to be called on validation failure")
	}
}

func TestPlanLaunch_DryRunPreparedEvidence(t *testing.T) {
	nodeStore := store.NewInMemoryStore()
	node := domain.NodeRecord{
		ID:            "node-1",
		Identity:      domain.NodeIdentity{Architecture: "x86_64"},
		Compatibility: domain.NodeCompatibilityRecord{Class: "kvm_standard"},
		Health:        domain.NodeHealthSummary{State: "Healthy"},
		Freshness:     domain.NodeFreshnessRecord{LastCollectionTime: time.Now()},
		Capabilities: map[string]domain.NodeCapabilityRecord{
			"kvm_vm_launch":       {State: "Supported"},
			"qemu_binary_present": {State: "Supported"},
		},
	}
	nodeStore.SaveNodeState(schema.SchedulerEnvelope{NodeID: "node-1"}, node)

	mockExec := &MockExecutor{
		PreparedLaunch: launch.PreparedLaunch{
			RuntimeBackend: "kvm_qemu",
			MemoryMB:       1024,
			Vcpu:           2,
		},
	}
	router := setupPlanTestRouter(nodeStore, mockExec)

	reqData := basePlanRequest("dry_run", "")
	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest("POST", "/plan/launch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var res plan.LaunchPlanResult
	json.Unmarshal(w.Body.Bytes(), &res)

	if res.Status != plan.PlanStatusReady {
		t.Errorf("expected Ready status, got %s", res.Status)
	}
	if !res.DryRunPrepared {
		t.Error("expected DryRunPrepared to be true")
	}
	if res.ValidationResult == nil || !res.ValidationResult.IsValid {
		t.Error("expected validation result to be valid")
	}
	if !mockExec.PrepareCalled {
		t.Error("expected Prepare to be called during dry_run")
	}
	if mockExec.ExecuteCalled {
		t.Error("expected Execute NOT to be called")
	}
}

func TestPlanLaunch_ConflictingNodeID(t *testing.T) {
	nodeStore := store.NewInMemoryStore()
	node := domain.NodeRecord{
		ID:            "node-1",
		Identity:      domain.NodeIdentity{Architecture: "x86_64"},
		Compatibility: domain.NodeCompatibilityRecord{Class: "kvm_standard"},
		Health:        domain.NodeHealthSummary{State: "Healthy"},
		Freshness:     domain.NodeFreshnessRecord{LastCollectionTime: time.Now()},
		Capabilities: map[string]domain.NodeCapabilityRecord{
			"kvm_vm_launch":       {State: "Supported"},
			"qemu_binary_present": {State: "Supported"},
		},
	}
	nodeStore.SaveNodeState(schema.SchedulerEnvelope{NodeID: "node-1"}, node)

	mockExec := &MockExecutor{}
	router := setupPlanTestRouter(nodeStore, mockExec)

	// Explicitly requesting node-2 but only node-1 is available/eligible
	reqData := basePlanRequest("validate", "node-2")
	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest("POST", "/plan/launch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var res plan.LaunchPlanResult
	json.Unmarshal(w.Body.Bytes(), &res)

	if res.Status != plan.PlanStatusConflict {
		t.Errorf("expected Conflict status, got %s", res.Status)
	}
	if len(res.Warnings) == 0 {
		t.Error("expected warnings explaining the conflict")
	}
}
