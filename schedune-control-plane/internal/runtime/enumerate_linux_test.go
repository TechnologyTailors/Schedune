package runtime

import (
	"testing"
)

func TestFingerprintProcess(t *testing.T) {
	backend := "cloud_hypervisor"
	command := "cloud-hypervisor"
	args := []string{"--api-socket", "/run/schedune/exe-123/ch.sock", "--memory", "size=1024M"}

	fp1 := FingerprintProcess(backend, command, args)
	
	// Same command, should produce same fingerprint
	fp2 := FingerprintProcess(backend, command, args)

	if fp1 != fp2 {
		t.Errorf("expected stable fingerprint, got %s and %s", fp1, fp2)
	}

	// Different args, different fingerprint
	args3 := []string{"--api-socket", "/run/schedune/exe-456/ch.sock", "--memory", "size=2048M"}
	fp3 := FingerprintProcess(backend, command, args3)

	if fp1 == fp3 {
		t.Errorf("expected different fingerprints for different args, got %s", fp1)
	}
}

func TestLinuxProcEnumerator_Basic(t *testing.T) {
	// Without mocking /proc this test just ensures it doesn't panic
	enumerator := &LinuxProcEnumerator{}
	_, err := enumerator.Enumerate()
	if err != nil {
		t.Fatalf("unexpected error enumerating /proc: %v", err)
	}
}
