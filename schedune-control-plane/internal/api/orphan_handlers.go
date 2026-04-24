package api

import (
	"context"
	"net/http"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store/sqlite"
	"github.com/gin-gonic/gin"
)

type OrphanHandler struct {
	orphanStore *sqlite.SQLiteStore
}

func NewOrphanHandler(orphanStore *sqlite.SQLiteStore) *OrphanHandler {
	return &OrphanHandler{orphanStore: orphanStore}
}

// ListOrphans returns a list of orphan processes matching optional filters
func (h *OrphanHandler) ListOrphans(c *gin.Context) {
	filter := domain.OrphanFilter{
		Backend:        c.Query("backend"),
		Status:         c.Query("status"),
		Classification: c.Query("classification"),
	}

	orphans, err := h.orphanStore.ListOrphans(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orphans": orphans})
}

// GetOrphan returns a specific orphan record
func (h *OrphanHandler) GetOrphan(c *gin.Context) {
	id := c.Param("id")

	orphan, found, err := h.orphanStore.GetOrphan(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Orphan not found"})
		return
	}

	c.JSON(http.StatusOK, orphan)
}
