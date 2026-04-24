package inspect

import (
	"os"
	"syscall"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

// ProcessInspector is the generic fallback inspector for simple PID-based runtimes
type ProcessInspector struct{}

func (i *ProcessInspector) Inspect(executionID string, pid *int, prepared launch.PreparedLaunch) (RuntimeObservation, error) {
	obs := RuntimeObservation{
		PID:           pid,
		ObservedAtSec: time.Now().Unix(),
	}

	if pid == nil {
		obs.ProcessExists = false
		return obs, nil
	}

	proc, err := os.FindProcess(*pid)
	if err != nil {
		obs.ProcessExists = false
		return obs, nil
	}

	// Sending signal 0 is the POSIX way to check if a process exists
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		obs.ProcessExists = true
		obs.BackendReadySignal = true // For V1, if it's alive, it's "ready"
		obs.BackendSignalSource = "process_exists"
	} else {
		// Process does not exist or we don't have permission (which implies it's gone for us)
		obs.ProcessExists = false
	}

	return obs, nil
}
