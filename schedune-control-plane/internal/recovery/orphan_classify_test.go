package recovery

import (
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

func TestMatchEnumeratedProcessToExecution(t *testing.T) {
	execs := []launch.LaunchExecutionRecord{
		{
			ExecutionID: "exec-123",
			PID:         func(i int) *int { return &i }(1234),
			PreparedState: &launch.PreparedLaunch{
				RuntimeBackend: "kvm_qemu",
			},
		},
		{
			ExecutionID: "exec-456",
			PID:         func(i int) *int { return &i }(5678),
			PreparedState: &launch.PreparedLaunch{
				RuntimeBackend: "cloud_hypervisor",
			},
		},
	}

	// Test Exact Match
	p1 := runtime.EnumeratedProcess{
		ExecutionIDHint: "exec-123",
		PID:             1234,
		Backend:         "kvm_qemu",
	}
	match1 := MatchEnumeratedProcessToExecution(p1, execs)
	if !match1.Matched || match1.ExecutionID != "exec-123" {
		t.Errorf("expected exact match for p1, got %v", match1)
	}

	// Test Weak Match
	p2 := runtime.EnumeratedProcess{
		ExecutionIDHint: "",
		PID:             5678,
		Backend:         "cloud_hypervisor",
	}
	match2 := MatchEnumeratedProcessToExecution(p2, execs)
	if !match2.Matched || match2.ExecutionID != "exec-456" {
		t.Errorf("expected weak match for p2, got %v", match2)
	}

	// Test No Match
	p3 := runtime.EnumeratedProcess{
		ExecutionIDHint: "",
		PID:             9999,
		Backend:         "cloud_hypervisor",
	}
	match3 := MatchEnumeratedProcessToExecution(p3, execs)
	if match3.Matched {
		t.Errorf("expected no match for p3, got %v", match3)
	}
}

func TestClassifyOrphan(t *testing.T) {
	p := runtime.EnumeratedProcess{
		ExecutionIDHint:    "stale-exec",
		CommandFingerprint: "hash123",
		Backend:            "kvm_qemu",
	}
	match := MatchResult{Matched: false, Ambiguous: false}
	
	orphan := ClassifyOrphan(p, match)
	if orphan.Classification != domain.OrphanStaleExecutionArtifact {
		t.Errorf("expected StaleExecutionArtifact, got %s", orphan.Classification)
	}

	p.ExecutionIDHint = ""
	orphan = ClassifyOrphan(p, match)
	if orphan.Classification != domain.OrphanUnmanagedBackendProcess {
		t.Errorf("expected UnmanagedBackendProcess, got %s", orphan.Classification)
	}

	match.Ambiguous = true
	orphan = ClassifyOrphan(p, match)
	if orphan.Classification != domain.OrphanPossiblyManagedButUnlinked {
		t.Errorf("expected PossiblyManagedButUnlinked, got %s", orphan.Classification)
	}
}
