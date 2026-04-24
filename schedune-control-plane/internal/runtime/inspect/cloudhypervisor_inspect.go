package inspect

import (
	"os"
	"syscall"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

type CloudHypervisorInspector struct{}

func (i *CloudHypervisorInspector) Inspect(executionID string, pid *int, prepared launch.PreparedLaunch) (RuntimeObservation, error) {
	now := time.Now().Unix()
	obs := RuntimeObservation{
		Backend:       "cloud_hypervisor",
		ObservedAtSec: now,
		PID:           pid,
	}

	if prepared.CloudHypervisor != nil {
		obs.ControlSocketPath = prepared.CloudHypervisor.ControlSocketPath
	}

	// Basic Process Existence
	if pid != nil {
		proc, err := os.FindProcess(*pid)
		if err == nil {
			if err := proc.Signal(syscall.Signal(0)); err == nil {
				obs.ProcessExists = true
			}
		}
	}

	obs.StartupGraceElapsed = true // Need to add Grace period calculation properly

	// CH Socket Checking
	if obs.ControlSocketPath != "" {
		obs.ControlSocketExists = runtime.SocketExists(obs.ControlSocketPath)
		if obs.ControlSocketExists {
			obs.ControlSocketDialOK = runtime.UnixSocketDialOK(obs.ControlSocketPath, 200*time.Millisecond)
		}
	}

	if obs.ControlSocketDialOK {
		obs.BackendReadySignal = true
		obs.BackendSignalSource = "ch_api_socket_dial_ok"
	} else if obs.ControlSocketExists {
		obs.BackendReadySignal = true
		obs.BackendSignalSource = "ch_api_socket_exists"
	}

	return obs, nil
}
