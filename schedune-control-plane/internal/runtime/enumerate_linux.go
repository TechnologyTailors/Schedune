package runtime

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LinuxProcEnumerator scans /proc to find supported backend processes
type LinuxProcEnumerator struct{}

func (e *LinuxProcEnumerator) Enumerate() ([]EnumeratedProcess, error) {
	var procs []EnumeratedProcess

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // not a PID directory
		}

		exePath := filepath.Join("/proc", entry.Name(), "exe")
		target, err := os.Readlink(exePath)
		if err != nil {
			continue // permission denied or kernel process
		}

		basename := filepath.Base(target)

		// Filter for supported backends
		backend := ""
		if strings.HasPrefix(basename, "qemu-system-") {
			backend = "kvm_qemu"
		} else if basename == "cloud-hypervisor" {
			backend = "cloud_hypervisor"
		} else if basename == "firecracker" {
			backend = "firecracker"
		} else {
			continue // Not a hypervisor process
		}

		cmdlinePath := filepath.Join("/proc", entry.Name(), "cmdline")
		cmdlineData, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}

		// cmdline is separated by null bytes
		args := strings.Split(string(bytes.TrimRight(cmdlineData, "\x00")), "\x00")
		if len(args) == 0 {
			continue
		}

		var ppid *int
		statPath := filepath.Join("/proc", entry.Name(), "stat")
		statData, err := os.ReadFile(statPath)
		if err == nil {
			// stat format: pid (comm) state ppid ...
			if endParen := bytes.LastIndexByte(statData, ')'); endParen != -1 {
				fields := strings.Fields(string(statData[endParen+2:]))
				if len(fields) >= 2 {
					if parsedPpid, err := strconv.Atoi(fields[1]); err == nil {
						ppid = &parsedPpid
					}
				}
			}
		}

		// Extract hints
		var execHint, workHint string
		for _, arg := range args {
			// Naive heuristic: if an arg contains an execution marker like /run/schedune/<exec_id>/...
			if strings.Contains(arg, "/run/schedune/") {
				parts := strings.Split(arg, "/")
				for i, p := range parts {
					if p == "schedune" && i+1 < len(parts) {
						execHint = parts[i+1]
						break
					}
				}
			}
			// Another heuristic if execution ID is passed explicitly
			if strings.HasPrefix(arg, "--exec-id=") {
				execHint = strings.TrimPrefix(arg, "--exec-id=")
			}
		}

		commandArgs := []string{}
		if len(args) > 1 {
			commandArgs = args[1:]
		}

		procs = append(procs, EnumeratedProcess{
			PID:                pid,
			PPID:               ppid,
			Backend:            backend,
			Command:            args[0],
			CommandArgs:        commandArgs,
			CommandFingerprint: FingerprintProcess(backend, args[0], commandArgs),
			ExecutionIDHint:    execHint,
			WorkloadIDHint:     workHint,
			ObservedAtSec:      now,
			Details:            make(map[string]interface{}),
		})
	}

	return procs, nil
}
