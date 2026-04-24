package inspect

import (
	"os"
	"syscall"
	"time"
)

// ProcessInspector is the generic fallback inspector for simple PID-based runtimes
type ProcessInspector struct{}

func (i *ProcessInspector) Inspect(pid int) (RuntimeObservation, error) {
	obs := RuntimeObservation{
		PID:           pid,
		ObservedAtSec: time.Now().Unix(),
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		obs.ProcessExists = false
		return obs, nil
	}

	// Sending signal 0 is the POSIX way to check if a process exists
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		obs.ProcessExists = true
		obs.BackendSpecificReady = true // For V1, if it's alive, it's "ready"
	} else {
		// Process does not exist or we don't have permission (which implies it's gone for us)
		obs.ProcessExists = false
		// We cannot easily get the exit code from an un-Wait()ed child in Go 
		// without a dedicated supervisor, so we leave ExitCode nil.
	}

	return obs, nil
}
