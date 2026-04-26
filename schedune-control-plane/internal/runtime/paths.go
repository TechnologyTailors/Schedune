package runtime

import (
	"fmt"
	"path/filepath"
	"regexp"
)

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// SanitizeWorkloadID returns the ID if it strictly matches safe characters, preventing path traversal.
func SanitizeWorkloadID(id string) (string, error) {
	if !validIDPattern.MatchString(id) {
		return "", fmt.Errorf("invalid characters in workload ID: %s", id)
	}
	return id, nil
}

// GetRuntimeDir returns the deterministic host directory for a specific workload's runtime artifacts.
// The resulting path is isolated under var/run/schedune to ensure a predictable and secure workspace.
func GetRuntimeDir(workloadID string) (string, error) {
	safeID, err := SanitizeWorkloadID(workloadID)
	if err != nil {
		return "", err
	}
	return filepath.Join("var", "run", "schedune", safeID), nil
}

// GetControlSocketPath returns the deterministic path for the runtime control socket.
func GetControlSocketPath(workloadID, backend string) (string, error) {
	dir, err := GetRuntimeDir(workloadID)
	if err != nil {
		return "", err
	}
	if !validIDPattern.MatchString(backend) {
		return "", fmt.Errorf("invalid characters in backend token: %s", backend)
	}
	return filepath.Join(dir, fmt.Sprintf("%s.sock", backend)), nil
}
