package runtime

import (
	"strings"
	"testing"
)

func TestSanitizeWorkloadID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"valid-id-123", false},
		{"valid_id_123", false},
		{"invalid/id", true},
		{"invalid\\id", true},
		{"../traversal", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			_, err := SanitizeWorkloadID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeWorkloadID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestGetRuntimeDir(t *testing.T) {
	dir, err := GetRuntimeDir("safe-id")
	if err != nil {
		t.Fatalf("GetRuntimeDir failed: %v", err)
	}
	if !strings.HasSuffix(dir, "var/run/schedune/safe-id") && !strings.HasSuffix(dir, "var\\run\\schedune\\safe-id") {
		t.Errorf("unexpected dir path: %s", dir)
	}
}

func TestGetControlSocketPath_InvalidBackend(t *testing.T) {
	_, err := GetControlSocketPath("valid-id", "../bad")
	if err == nil {
		t.Errorf("expected GetControlSocketPath to fail with invalid backend token")
	}
}
