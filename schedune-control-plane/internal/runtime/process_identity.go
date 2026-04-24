package runtime

type ProcessIdentity struct {
	PID                int
	Backend            string
	StartedAtSec       *int64
	CommandFingerprint string
	ExecutionIDHint    string
}
