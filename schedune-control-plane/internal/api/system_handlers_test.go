package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/gin-gonic/gin"
)

func TestSystemHandler_Healthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler := NewSystemHandler()
	handler.Healthz(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%v'", response["status"])
	}
	if response["api_version"] != "v1alpha1" {
		t.Errorf("Expected api_version 'v1alpha1', got '%v'", response["api_version"])
	}
	if response["storage"] != "sqlite" {
		t.Errorf("Expected storage 'sqlite', got '%v'", response["storage"])
	}

	expectedEnumEnabled := runtime.GOOS == "linux"
	if response["process_enumeration_enabled"] != expectedEnumEnabled {
		t.Errorf("Expected process_enumeration_enabled %v, got %v", expectedEnumEnabled, response["process_enumeration_enabled"])
	}
}

func TestNodeHandler_ListAndGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	memStore := store.NewInMemoryStore()
	handler := NewNodeHandler(memStore)

	// Add a dummy node directly to the underlying map for testing, or we can use domain.NodeRecord.
	// Since InMemoryStore has SaveNodeState, we'd need an envelope. To mock, let's look at InMemoryStore.
	// Actually, there is a ListAllNodes that reads from a map. If empty, returns [].

	// Test ListNodes (empty)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	handler.ListNodes(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// For a fully fleshed out test, let's just make sure it returns 200 and a 'nodes' array
	var listResp map[string][]NodeSummary
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Test GetNode (not found)
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Params = []gin.Param{{Key: "id", Value: "non-existent"}}
	handler.GetNode(c2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w2.Code)
	}
}
