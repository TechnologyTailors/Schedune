package main

import (
	"os"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/api"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure zerolog
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	gin.SetMode(gin.ReleaseMode)

	// Initialize the domain store
	memStore := store.NewInMemoryStore()

	// Initialize HTTP Handlers
	intakeHandler := api.NewIntakeHandler(memStore)
	schedulerHandler := api.NewSchedulerHandler(memStore)
	launchHandler := api.NewLaunchHandler(memStore)

	// Setup Router
	r := gin.New()
	r.Use(gin.Recovery())
	
	// API Group
	v1 := r.Group("/api/v1alpha1")
	{
		// Data Plane -> Control Plane
		v1.POST("/intake/envelope", intakeHandler.Ingest)
		
		// Operator -> Control Plane (Explainability)
		v1.GET("/nodes/:id/explain", intakeHandler.ExplainNodeDecision)
		
		// Workload Orchestration -> Control Plane
		v1.POST("/schedule/explain", schedulerHandler.ExplainSchedule)
		v1.POST("/schedule/select", schedulerHandler.SelectNode)

		// Execution Contract -> Control Plane
		v1.POST("/launch/validate", launchHandler.ValidateLaunch)
		v1.POST("/launch/dry-run", launchHandler.DryRunLaunch)
		v1.POST("/launch/execute", launchHandler.ExecuteLaunch)
		v1.GET("/launch/:id", launchHandler.InspectLaunch)
		v1.POST("/launch/:id/terminate", launchHandler.TerminateLaunch)
	}

	log.Info().Msg("Starting Schedune Control Plane on :9090")
	if err := r.Run(":9090"); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
