package inspect

// RuntimeObservation is the internal object representing the current state of a runtime process
type RuntimeObservation struct {
	ProcessExists        bool
	PID                  int
	ExitCode             *int
	ObservedAtSec        int64
	BackendSpecificReady bool
	BackendSpecificError *string
}

// Inspector defines the abstraction for checking runtime health
type Inspector interface {
	Inspect(pid int) (RuntimeObservation, error)
}
