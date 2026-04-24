package runtime

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// FingerprintProcess creates a stable hash for a process based on its core identity
// This helps us deduplicate orphans across sweeps even if their PIDs change slightly 
// (e.g. process respawning with identical args but no manager).
func FingerprintProcess(backend, command string, args []string) string {
	// Filter out highly variable args for fingerprinting
	// For V1, we simply concatenate all args, but long-term we should strip 
	// things like temp file paths or dynamic file descriptors if they vary.
	
	canonicalArgs := strings.Join(args, " ")
	payload := fmt.Sprintf("backend=%s|bin=%s|args=%s", backend, command, canonicalArgs)
	
	hash := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%x", hash)
}
