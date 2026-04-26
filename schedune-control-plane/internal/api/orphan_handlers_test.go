package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store/sqlite"
	"github.com/gin-gonic/gin"
)

func TestOrphanHandler_ListOrphans_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Use an in-memory sqlite db for tests
	dbStore, err := sqlite.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}

	handler := NewOrphanHandler(dbStore)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Add mock request to context so query params don't panic
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1alpha1/recovery/orphans", nil)

	handler.ListOrphans(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Since there are no orphans in the fresh DB, it should return an empty slice, not nil
	orphansInter, ok := response["orphans"]
	if !ok {
		t.Fatalf("Expected 'orphans' key in response")
	}

	orphansList, ok := orphansInter.([]interface{})
	if !ok {
		t.Fatalf("Expected 'orphans' to be a list, got %T", orphansInter)
	}

	if len(orphansList) != 0 {
		t.Errorf("Expected empty list, got length %d", len(orphansList))
	}
}
