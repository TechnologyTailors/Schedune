package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/gin-gonic/gin"
)

func TestIntake_ValidEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	path := filepath.Join("..", "..", "..", "testdata", "fixtures", "healthy_arm_production.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	memStore := store.NewInMemoryStore()
	handler := NewIntakeHandler(memStore)

	r := gin.New()
	r.POST("/api/v1alpha1/intake/envelope", handler.Ingest)

	req, _ := http.NewRequest("POST", "/api/v1alpha1/intake/envelope", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %v: %s", w.Code, w.Body.String())
	}

	// Verify node was stored
	node, err := memStore.GetNode("arm-prod-01")
	if err != nil {
		t.Errorf("expected node to be stored, got error: %v", err)
	}
	if node.Identity.Architecture != "aarch64" {
		t.Errorf("expected architecture 'aarch64', got %s", node.Identity.Architecture)
	}
}

func TestIntake_InvalidSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Missing mandatory fields like collection_id and facts
	invalidJSON := []byte(`{
		"schema_version": "v1alpha1",
		"node_id": "test-node",
		"compatibility": { "class": "ArmProduction", "reason_codes": [] },
		"health": { "state": "Healthy", "active_alarms": [] }
	}`)

	memStore := store.NewInMemoryStore()
	handler := NewIntakeHandler(memStore)

	r := gin.New()
	r.POST("/api/v1alpha1/intake/envelope", handler.Ingest)

	req, _ := http.NewRequest("POST", "/api/v1alpha1/intake/envelope", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request for invalid schema, got %v", w.Code)
	}
}
