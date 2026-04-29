package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/plan"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
	"github.com/gin-gonic/gin"
)

func setupPlanTestRouter(nodeStore *store.InMemoryStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewPlanHandler(nodeStore)
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
		LaunchTemplate: plan.LaunchTemplateSpec{
			SchemaVersion: "v1alpha1",
			WorkloadID:    "wl-1",
			TenantID:      "tenant-1",
			NodeID:        "",
			RuntimeClass:  "VirtualMachine",
			Architecture:  "x86_64",
			Vcpu:          2,
			MemoryMB:      1024,
			LaunchMode:    "",
			Storage: []launch.StorageAttachmentSpec{
				{HostPath: "/tmp/test.qcow2", Format: "qcow2"},
			},
		},
	}
}

func TestPlanLaunch_NoEligibleNode(t *testing.T) {
	nodeStore := store.NewInMemoryStore()
	router := setupPlanTestRouter(nodeStore)

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

	router := setupPlanTestRouter(nodeStore)

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

	router := setupPlanTestRouter(nodeStore)

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
	if res.DryRunPrepared {
		t.Error("expected DryRunPrepared to be false for node-scoped prep")
	}
	if res.PreparationResult == nil {
		t.Error("expected PreparationResult to be populated")
	} else {
		if res.PreparationResult.Status != plan.PreparationStatusPendingNodeAgent {
			t.Errorf("expected PendingNodeAgent, got %s", res.PreparationResult.Status)
		}
	}
	if res.ValidationResult == nil || !res.ValidationResult.IsValid {
		t.Error("expected validation result to be valid")
	}

	// Check NextActions is correctly updated to PrepareOnNode
	foundPrepareOnNode := false
	for _, action := range res.NextActions {
		if action == plan.ActionPrepareOnNode {
			foundPrepareOnNode = true
			break
		}
	}
	if !foundPrepareOnNode {
		t.Errorf("expected next_actions to contain PrepareOnNode, got %v", res.NextActions)
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

	router := setupPlanTestRouter(nodeStore)

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

func TestPlanLaunch_TemplateConflict(t *testing.T) {
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

	router := setupPlanTestRouter(nodeStore)

	reqData := basePlanRequest("validate", "")
	reqData.LaunchTemplate.NodeID = "node-2" // Conflict with selected node-1

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
}

func TestPlanLaunch_ValidateModeHydration(t *testing.T) {
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

	router := setupPlanTestRouter(nodeStore)

	reqData := basePlanRequest("validate", "") // empty node_id and launch_mode in template
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
	if res.PlannedLaunchSpec == nil {
		t.Fatal("expected planned launch spec to be non-nil")
	}
	if res.PlannedLaunchSpec.LaunchMode != "Validate" {
		t.Errorf("expected launch mode Validate, got %s", res.PlannedLaunchSpec.LaunchMode)
	}
	if res.PlannedLaunchSpec.NodeID != "node-1" {
		t.Errorf("expected node ID node-1, got %s", res.PlannedLaunchSpec.NodeID)
	}
}

func TestPlanLaunch_DryRunModeHydration(t *testing.T) {
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

	router := setupPlanTestRouter(nodeStore)

	reqData := basePlanRequest("dry_run", "") // empty node_id and launch_mode in template
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
	if res.PlannedLaunchSpec == nil {
		t.Fatal("expected planned launch spec to be non-nil")
	}
	if res.PlannedLaunchSpec.LaunchMode != "DryRun" {
		t.Errorf("expected launch mode DryRun, got %s", res.PlannedLaunchSpec.LaunchMode)
	}
	if res.PlannedLaunchSpec.NodeID != "node-1" {
		t.Errorf("expected node ID node-1, got %s", res.PlannedLaunchSpec.NodeID)
	}
	if res.DryRunPrepared {
		t.Errorf("expected DryRunPrepared false for node-scoped prep")
	}
}
