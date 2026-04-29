package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/plan"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
)

func TestBuildLaunchPlan_ConsistencyCheck(t *testing.T) {
	freshTime := time.Now()
	candidateNodes := []NodeRecord{
		{
			ID:            "node-1",
			Identity:      NodeIdentity{Architecture: "x86_64"},
			Health:        NodeHealthSummary{State: "Healthy", ActiveAlarms: []schema.ActiveAlarm{}},
			Compatibility: NodeCompatibilityRecord{Class: "kvm_standard"},
			Freshness:     NodeFreshnessRecord{LastCollectionTime: freshTime},
		},
	}

	tests := []struct {
		name     string
		intent   workload.WorkloadIntent
		template plan.LaunchTemplateSpec
		errWarn  string
	}{
		{
			name: "Mismatched WorkloadID",
			intent: workload.WorkloadIntent{
				WorkloadID:                   "w-1",
				TenantID:                     "t-1",
				RuntimeClass:                 "VirtualMachine",
				RequiredArchitecture:         "x86_64",
				MaxTelemetryAgeSec:           300,
				RequiredCompatibilityClasses: []string{"kvm_standard"},
			},
			template: plan.LaunchTemplateSpec{
				SchemaVersion: "v1alpha1",
				WorkloadID:    "w-2",
				TenantID:      "t-1",
				RuntimeClass:  "VirtualMachine",
				Architecture:  "x86_64",
				Vcpu:          2,
				MemoryMB:      1024,
			},
			errWarn: "Intent and Template WorkloadID mismatch",
		},
		{
			name: "Mismatched TenantID",
			intent: workload.WorkloadIntent{
				WorkloadID:                   "w-1",
				TenantID:                     "t-1",
				RuntimeClass:                 "VirtualMachine",
				RequiredArchitecture:         "x86_64",
				MaxTelemetryAgeSec:           300,
				RequiredCompatibilityClasses: []string{"kvm_standard"},
			},
			template: plan.LaunchTemplateSpec{
				SchemaVersion: "v1alpha1",
				WorkloadID:    "w-1",
				TenantID:      "t-2",
				RuntimeClass:  "VirtualMachine",
				Architecture:  "x86_64",
				Vcpu:          2,
				MemoryMB:      1024,
			},
			errWarn: "Intent and Template TenantID mismatch",
		},
		{
			name: "Mismatched RuntimeClass",
			intent: workload.WorkloadIntent{
				WorkloadID:                   "w-1",
				TenantID:                     "t-1",
				RuntimeClass:                 "VirtualMachine",
				RequiredArchitecture:         "x86_64",
				MaxTelemetryAgeSec:           300,
				RequiredCompatibilityClasses: []string{"kvm_standard"},
			},
			template: plan.LaunchTemplateSpec{
				SchemaVersion: "v1alpha1",
				WorkloadID:    "w-1",
				TenantID:      "t-1",
				RuntimeClass:  "MicroVM",
				Architecture:  "x86_64",
				Vcpu:          2,
				MemoryMB:      1024,
			},
			errWarn: "Intent and Template RuntimeClass mismatch",
		},
		{
			name: "Mismatched Architecture",
			intent: workload.WorkloadIntent{
				WorkloadID:                   "w-1",
				TenantID:                     "t-1",
				RuntimeClass:                 "VirtualMachine",
				RequiredArchitecture:         "aarch64",
				MaxTelemetryAgeSec:           300,
				RequiredCompatibilityClasses: []string{"kvm_standard"},
			},
			template: plan.LaunchTemplateSpec{
				SchemaVersion: "v1alpha1",
				WorkloadID:    "w-1",
				TenantID:      "t-1",
				RuntimeClass:  "VirtualMachine",
				Architecture:  "x86_64",
				Vcpu:          2,
				MemoryMB:      1024,
			},
			errWarn: "Intent RequiredArchitecture and Template Architecture mismatch",
		},
		{
			name: "Architecture match with any",
			intent: workload.WorkloadIntent{
				WorkloadID:                   "w-1",
				TenantID:                     "t-1",
				RuntimeClass:                 "VirtualMachine",
				RequiredArchitecture:         "any",
				MaxTelemetryAgeSec:           300,
				RequiredCompatibilityClasses: []string{"kvm_standard"},
			},
			template: plan.LaunchTemplateSpec{
				SchemaVersion: "v1alpha1",
				WorkloadID:    "w-1",
				TenantID:      "t-1",
				RuntimeClass:  "VirtualMachine",
				Architecture:  "x86_64",
				Vcpu:          2,
				MemoryMB:      1024,
			},
			errWarn: "", // Should pass consistency check
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := BuildLaunchPlan(tc.intent, tc.template, "validate", "", candidateNodes)
			if tc.errWarn != "" {
				if res.Status != plan.PlanStatusConflict {
					t.Errorf("expected status %s, got %s", plan.PlanStatusConflict, res.Status)
				}

				foundWarn := false
				for _, w := range res.Warnings {
					if w == tc.errWarn || strings.Contains(w, tc.errWarn) {
						foundWarn = true
						break
					}
				}
				if !foundWarn {
					t.Errorf("expected warning containing %q, got %v", tc.errWarn, res.Warnings)
				}
			} else {
				// Since we just have an empty node, it might fail scheduling, but shouldn't conflict on consistency
				if res.Status == plan.PlanStatusConflict {
					t.Errorf("expected status not to be %s", plan.PlanStatusConflict)
				}
			}
		})
	}
}
