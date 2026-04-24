package runtime

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
