package inspect

import "github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"

// RuntimeObservation is the internal object representing the current state of a runtime process
type RuntimeObservation struct {
	Backend              string
	ProcessExists        bool
	PID                  *int
	ExitCode             *int
	ObservedAtSec        int64
	ControlSocketPath    string
	ControlSocketExists  bool
	ControlSocketDialOK  bool
	StartupGraceElapsed  bool
	BackendReadySignal   bool
	BackendSignalSource  string
	BackendSpecificError *string
}

// Inspector defines the abstraction for checking runtime health
type Inspector interface {
	Inspect(executionID string, pid *int, prepared launch.PreparedLaunch) (RuntimeObservation, error)
}
