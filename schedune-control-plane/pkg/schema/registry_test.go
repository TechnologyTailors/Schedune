package schema_test

import (
	"strings"
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
)

func TestReasonCodeRegistry_Constraints(t *testing.T) {
	codes := schema.KnownReasonCodes()

	allowedPrefixes := []string{"CAP_", "REJECT_", "ERR_", "WARN_", "RECOVERY_"}

	for code := range codes {
		if code == "" {
			t.Errorf("found empty reason code")
		}

		hasPrefix := false
		for _, prefix := range allowedPrefixes {
			if strings.HasPrefix(code, prefix) {
				hasPrefix = true
				break
			}
		}

		if !hasPrefix {
			t.Errorf("reason code %q does not start with an allowed prefix: %v", code, allowedPrefixes)
		}
	}
}

func TestReasonCodeRegistry_RepresentativeCodes(t *testing.T) {
	// Verify representative lifecycle/readiness/recovery codes
	expectedCodes := []string{
		schema.ReasonErrLaunchArchMismatch,
		schema.ReasonErrLaunchMissingCapabilitySeccomp,
		schema.ReasonRejectArchitectureMismatch,
		schema.ReasonRecoveryConfirmed,
		schema.ReasonErrReadyProbeFailed,
		schema.ReasonErrReconcileProcessMissing,
		schema.ReasonErrExecRuntimeCrashed,
	}

	for _, expected := range expectedCodes {
		if !schema.IsKnownReasonCode(expected) {
			t.Errorf("expected representative code %q to be registered", expected)
		}
	}
}
