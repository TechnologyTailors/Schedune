package runtime

import "runtime"

type EnumeratedProcess struct {
	PID                int
	PPID               *int
	Backend            string
	Command            string
	CommandArgs        []string
	CommandFingerprint string
	ExecutionIDHint    string
	WorkloadIDHint     string
	ObservedAtSec      int64
	Details            map[string]interface{}
}

type Enumerator interface {
	Enumerate() ([]EnumeratedProcess, error)
}

func NewEnumerator() Enumerator {
	if runtime.GOOS == "linux" {
		return &LinuxProcEnumerator{}
	}
	return &NoopEnumerator{}
}

type NoopEnumerator struct{}

func (e *NoopEnumerator) Enumerate() ([]EnumeratedProcess, error) {
	return nil, nil
}
