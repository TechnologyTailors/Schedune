package api

import (
	"net/http"
	"runtime"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/gin-gonic/gin"
)

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

func (h *SystemHandler) Healthz(c *gin.Context) {
	processEnumerationEnabled := runtime.GOOS == "linux"

	c.JSON(http.StatusOK, gin.H{
		"status":                      "ok",
		"api_version":                 "v1alpha1",
		"storage":                     "sqlite",
		"process_enumeration_enabled": processEnumerationEnabled,
	})
}

type NodeSummary struct {
	ID           string `json:"id"`
	Hostname     string `json:"hostname"`
	Architecture string `json:"architecture"`
	Health       string `json:"health"`
	Class        string `json:"class"`
}

type NodeHandler struct {
	store *store.InMemoryStore
}

func NewNodeHandler(s *store.InMemoryStore) *NodeHandler {
	return &NodeHandler{store: s}
}

func (h *NodeHandler) ListNodes(c *gin.Context) {
	nodes := h.store.ListAllNodes()
	var summaries []NodeSummary

	for _, n := range nodes {
		summaries = append(summaries, NodeSummary{
			ID:           n.ID,
			Hostname:     n.Identity.Hostname,
			Architecture: n.Identity.Architecture,
			Health:       n.Health.State,
			Class:        n.Compatibility.Class,
		})
	}

	c.JSON(http.StatusOK, gin.H{"nodes": summaries})
}

func (h *NodeHandler) GetNode(c *gin.Context) {
	nodeID := c.Param("id")
	record, err := h.store.GetNode(nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	c.JSON(http.StatusOK, record)
}
