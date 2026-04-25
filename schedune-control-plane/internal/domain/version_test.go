package domain

import (
	"testing"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		raw    string
		expect parsedVersion
		ok     bool
	}{
		{"QEMU emulator version 6.2.0 (Debian 1:6.2+dfsg-2ubuntu6.22)", parsedVersion{6, 2, 0}, true},
		{"cloud-hypervisor v32.0.0", parsedVersion{32, 0, 0}, true},
		{"Firecracker v1.4.0", parsedVersion{1, 4, 0}, true},
		{"v2.0", parsedVersion{2, 0, 0}, true},
		{"2.1", parsedVersion{2, 1, 0}, true},
		{"invalid", parsedVersion{}, false},
	}

	for _, tt := range tests {
		v, ok := parseVersion(tt.raw)
		if ok != tt.ok {
			t.Errorf("parseVersion(%q) expected ok=%v, got %v", tt.raw, tt.ok, ok)
			continue
		}
		if ok && (*v != tt.expect) {
			t.Errorf("parseVersion(%q) expected %v, got %v", tt.raw, tt.expect, *v)
		}
	}
}

func TestCheckVersionRequirement(t *testing.T) {
	v1_4_0 := "Firecracker v1.4.0"
	v1_5_0 := "Firecracker v1.5.0"
	invalid := "invalid"

	tests := []struct {
		name        string
		observedRaw string
		req         *launch.RuntimeVersionRequirement
		expectOk    bool
		expectErr   string
	}{
		{"No requirement", v1_4_0, nil, true, ""},
		{"Empty requirement", v1_4_0, &launch.RuntimeVersionRequirement{}, true, ""},
		{"Missing observed version", "", &launch.RuntimeVersionRequirement{MinimumVersion: "1.0.0"}, false, schema.ReasonErrLaunchRuntimeVersionUnknown},
		{"Unparseable observed version", invalid, &launch.RuntimeVersionRequirement{MinimumVersion: "1.0.0"}, false, schema.ReasonErrLaunchRuntimeVersionUnparseable},
		{"Unparseable minimum version", v1_4_0, &launch.RuntimeVersionRequirement{MinimumVersion: "invalid"}, false, schema.ReasonErrLaunchRuntimeVersionUnparseable},
		{"Unparseable exact version", v1_4_0, &launch.RuntimeVersionRequirement{ExactVersion: "invalid"}, false, schema.ReasonErrLaunchRuntimeVersionUnparseable},
		{"Minimum satisfied", v1_5_0, &launch.RuntimeVersionRequirement{MinimumVersion: "1.4.0"}, true, ""},
		{"Minimum not satisfied", v1_4_0, &launch.RuntimeVersionRequirement{MinimumVersion: "1.5.0"}, false, schema.ReasonErrLaunchRuntimeVersionTooOld},
		{"Exact satisfied", v1_4_0, &launch.RuntimeVersionRequirement{ExactVersion: "1.4.0"}, true, ""},
		{"Exact not satisfied", v1_5_0, &launch.RuntimeVersionRequirement{ExactVersion: "1.4.0"}, false, schema.ReasonErrLaunchRuntimeVersionMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := checkVersionRequirement(tt.observedRaw, tt.req)
			if ok != tt.expectOk {
				t.Errorf("checkVersionRequirement expected ok=%v, got %v", tt.expectOk, ok)
			}
			if err != tt.expectErr {
				t.Errorf("checkVersionRequirement expected err=%v, got %v", tt.expectErr, err)
			}
		})
	}
}
