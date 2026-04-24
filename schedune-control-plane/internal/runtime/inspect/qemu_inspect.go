package inspect

import (
	"os"
	"syscall"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

type QemuInspector struct{}

func (i *QemuInspector) Inspect(executionID string, pid *int, prepared launch.PreparedLaunch) (RuntimeObservation, error) {
	now := time.Now().Unix()
	obs := RuntimeObservation{
		Backend:       "kvm_qemu",
		ObservedAtSec: now,
		PID:           pid,
	}

	if prepared.KvmQemu != nil {
		obs.ControlSocketPath = prepared.KvmQemu.ControlSocketPath
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

	// Startup Grace Window
	// If the execution started less than 2 seconds ago, we don't expect a socket yet
	// Note: the orchestrator doesn't track StartedAtSec in the PreparedLaunch, 
	// so we'll just implement a dummy 2 sec check for now.
	obs.StartupGraceElapsed = true // Assume elapsed unless we explicitly track start time

	// QMP Socket Checking
	if obs.ControlSocketPath != "" {
		obs.ControlSocketExists = runtime.SocketExists(obs.ControlSocketPath)
		if obs.ControlSocketExists {
			obs.ControlSocketDialOK = runtime.UnixSocketDialOK(obs.ControlSocketPath, 200*time.Millisecond)
		}
	}

	if obs.ControlSocketDialOK {
		obs.BackendReadySignal = true
		obs.BackendSignalSource = "qmp_socket_dial_ok"
	} else if obs.ControlSocketExists {
		obs.BackendReadySignal = true
		obs.BackendSignalSource = "qmp_socket_exists"
	}

	return obs, nil
}
