#[cfg(test)]
mod tests {
    use crate::capabilities::{CpuFacts, MemoryFacts, OsFacts, NodeFacts, CompatibilityClassification, CompatibilityClassType, NodeCapability, SupportState, Provenance, NodeConstraint};
    use crate::health::{NodeHealth, HealthState, ActiveAlarm, AlarmSeverity};
    use crate::scheduler_contract::{SchedulerEnvelope, CollectorStatus};
    use std::fs;
    use std::path::PathBuf;

    fn write_fixture(name: &str, envelope: &SchedulerEnvelope) {
        let mut path = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
        path.push("../testdata/fixtures");
        fs::create_dir_all(&path).unwrap();
        path.push(name);
        
        let json = serde_json::to_string_pretty(envelope).unwrap();
        fs::write(path, json).unwrap();
    }

    #[test]
    fn generate_healthy_arm_production() {
        let facts = NodeFacts {
            cpu: CpuFacts {
                architecture: "aarch64".to_string(),
                cores: 128,
                vendor_id: Some("ARM".to_string()),
            },
            memory: MemoryFacts { total_mb: 262144 },
            os: OsFacts {
                hostname: "arm-prod-01".to_string(),
                name: "Ubuntu".to_string(),
                kernel_version: Some("6.8.0".to_string()),
            },
        };

        let class = CompatibilityClassification {
            class: CompatibilityClassType::ArmProduction,
            reason_codes: vec!["CLASS_ARM_PROD_READY".to_string()],
        };

        let capabilities = vec![
            NodeCapability {
                feature: "kvm_vm_launch".to_string(),
                state: SupportState::Supported,
                provenance: Provenance::Observed,
                reason_code: Some("KVM_OPENABLE".to_string()),
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            },
            NodeCapability {
                feature: "hardware_tpm".to_string(),
                state: SupportState::Supported,
                provenance: Provenance::Observed,
                reason_code: Some("CAP_TPM_PRESENT".to_string()),
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            }
        ];

        let health = NodeHealth {
            state: HealthState::Healthy,
            active_alarms: vec![],
        };

        let status = CollectorStatus {
            collector_name: "MockCollector".to_string(),
            success: true,
            duration_ms: 10,
            error_message: None,
        };

        let envelope = SchedulerEnvelope::new(
            "arm-prod-01".to_string(),
            class,
            facts,
            capabilities,
            vec![], // No constraints
            health,
            vec![status],
        );

        write_fixture("healthy_arm_production.json", &envelope);
    }

    #[test]
    fn generate_missing_kvm_x86() {
        let facts = NodeFacts {
            cpu: CpuFacts {
                architecture: "x86_64".to_string(),
                cores: 16,
                vendor_id: Some("GenuineIntel".to_string()),
            },
            memory: MemoryFacts { total_mb: 65536 },
            os: OsFacts {
                hostname: "x86-storage-01".to_string(),
                name: "Ubuntu".to_string(),
                kernel_version: Some("6.8.0".to_string()),
            },
        };

        let class = CompatibilityClassification {
            class: CompatibilityClassType::Unsupported,
            reason_codes: vec!["CLASS_UNSUPPORTED_NO_KVM".to_string()],
        };

        let capabilities = vec![
            NodeCapability {
                feature: "kvm_vm_launch".to_string(),
                state: SupportState::Unsupported,
                provenance: Provenance::Observed,
                reason_code: Some("KVM_MISSING".to_string()),
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            }
        ];

        let constraints = vec![
            NodeConstraint {
                constraint_type: "VirtualizationDisabled".to_string(),
                code: "CONSTRAINT_NO_KVM".to_string(),
                description: "Cannot schedule KVM VMs on this node.".to_string(),
                observed_value: Some("missing".to_string()),
                expected_value: Some("present".to_string()),
            }
        ];

        let health = NodeHealth {
            state: HealthState::Healthy,
            active_alarms: vec![],
        };

        let status = CollectorStatus {
            collector_name: "MockCollector".to_string(),
            success: true,
            duration_ms: 15,
            error_message: None,
        };

        let envelope = SchedulerEnvelope::new(
            "x86-storage-01".to_string(),
            class,
            facts,
            capabilities,
            constraints,
            health,
            vec![status],
        );

        write_fixture("missing_kvm_x86.json", &envelope);
    }

    #[test]
    fn generate_stale_telemetry() {
        let facts = NodeFacts {
            cpu: CpuFacts { architecture: "aarch64".to_string(), cores: 64, vendor_id: None },
            memory: MemoryFacts { total_mb: 131072 },
            os: OsFacts { hostname: "stale-arm-01".to_string(), name: "Ubuntu".to_string(), kernel_version: None },
        };

        let class = CompatibilityClassification {
            class: CompatibilityClassType::ArmProduction,
            reason_codes: vec!["CLASS_ARM_PROD_READY".to_string()],
        };

        // Note: The timestamps here are very old, making them explicitly stale
        let capabilities = vec![
            NodeCapability {
                feature: "kvm_vm_launch".to_string(),
                state: SupportState::Supported,
                provenance: Provenance::Observed,
                reason_code: Some("KVM_OPENABLE".to_string()),
                observed_at_sec: 1000000000, // Very old
                stale_after_sec: Some(1000000300),
            }
        ];

        let health = NodeHealth {
            state: HealthState::Healthy,
            active_alarms: vec![],
        };

        let status = CollectorStatus {
            collector_name: "MockCollector".to_string(),
            success: false, // Explicit collector failure
            duration_ms: 5000,
            error_message: Some("Connection timeout".to_string()),
        };

        let mut envelope = SchedulerEnvelope::new(
            "stale-arm-01".to_string(),
            class,
            facts,
            capabilities,
            vec![],
            health,
            vec![status],
        );

        envelope.timestamp_sec = 1000000300; // Old envelope timestamp
        write_fixture("stale_telemetry.json", &envelope);
    }

    #[test]
    fn generate_healthy_x86_kvm_openable() {
        let facts = NodeFacts {
            cpu: CpuFacts { architecture: "x86_64".to_string(), cores: 32, vendor_id: Some("AMD".to_string()) },
            memory: MemoryFacts { total_mb: 131072 },
            os: OsFacts { hostname: "x86-pool-01".to_string(), name: "Ubuntu".to_string(), kernel_version: Some("6.8.0".to_string()) },
        };

        let class = CompatibilityClassification {
            class: CompatibilityClassType::X86HoldingPool,
            reason_codes: vec!["CLASS_X86_HOLDING_READY".to_string()],
        };

        let capabilities = vec![
            NodeCapability {
                feature: "kvm_vm_launch".to_string(),
                state: SupportState::Supported,
                provenance: Provenance::Observed,
                reason_code: Some("KVM_OPENABLE".to_string()),
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            }
        ];

        let envelope = SchedulerEnvelope::new("x86-pool-01".to_string(), class, facts, capabilities, vec![], NodeHealth { state: HealthState::Healthy, active_alarms: vec![] }, vec![CollectorStatus { collector_name: "MockCollector".to_string(), success: true, duration_ms: 10, error_message: None }]);

        write_fixture("healthy_x86_kvm_openable.json", &envelope);
    }

    #[test]
    fn generate_kvm_exists_not_openable() {
        let facts = NodeFacts {
            cpu: CpuFacts { architecture: "x86_64".to_string(), cores: 32, vendor_id: Some("AMD".to_string()) },
            memory: MemoryFacts { total_mb: 131072 },
            os: OsFacts { hostname: "x86-perms-fail-01".to_string(), name: "Ubuntu".to_string(), kernel_version: Some("6.8.0".to_string()) },
        };

        let class = CompatibilityClassification {
            class: CompatibilityClassType::Degraded,
            reason_codes: vec!["CLASS_DEGRADED_KVM_PERMS".to_string()],
        };

        let capabilities = vec![
            NodeCapability {
                feature: "kvm_vm_launch".to_string(),
                state: SupportState::Unavailable,
                provenance: Provenance::Observed,
                reason_code: Some("KVM_NOT_OPENABLE_PERMS".to_string()),
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            }
        ];

        let constraints = vec![
            NodeConstraint {
                constraint_type: "VirtualizationDisabled".to_string(),
                code: "CONSTRAINT_KVM_PERMS".to_string(),
                description: "/dev/kvm exists but is not RW".to_string(),
                observed_value: Some("read_only".to_string()),
                expected_value: Some("read_write".to_string()),
            }
        ];

        let envelope = SchedulerEnvelope::new("x86-perms-fail-01".to_string(), class, facts, capabilities, constraints, NodeHealth { state: HealthState::Healthy, active_alarms: vec![] }, vec![CollectorStatus { collector_name: "MockCollector".to_string(), success: true, duration_ms: 10, error_message: None }]);

        write_fixture("kvm_exists_not_openable.json", &envelope);
    }

    #[test]
    fn generate_firecracker_partial_fail() {
        let facts = NodeFacts {
            cpu: CpuFacts { architecture: "aarch64".to_string(), cores: 64, vendor_id: Some("ARM".to_string()) },
            memory: MemoryFacts { total_mb: 131072 },
            os: OsFacts { hostname: "arm-fc-fail-01".to_string(), name: "Ubuntu".to_string(), kernel_version: Some("6.8.0".to_string()) },
        };

        let class = CompatibilityClassification {
            class: CompatibilityClassType::ArmProduction,
            reason_codes: vec!["CLASS_ARM_PROD_READY".to_string()],
        };

        let capabilities = vec![
            NodeCapability {
                feature: "kvm_vm_launch".to_string(),
                state: SupportState::Supported,
                provenance: Provenance::Observed,
                reason_code: Some("KVM_OPENABLE".to_string()),
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            },
            NodeCapability {
                feature: "firecracker_launch".to_string(),
                state: SupportState::Unavailable,
                provenance: Provenance::Observed,
                reason_code: Some("FC_PREREQS_MISSING".to_string()),
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            }
        ];

        let envelope = SchedulerEnvelope::new("arm-fc-fail-01".to_string(), class, facts, capabilities, vec![], NodeHealth { state: HealthState::Healthy, active_alarms: vec![] }, vec![CollectorStatus { collector_name: "MockCollector".to_string(), success: true, duration_ms: 10, error_message: None }]);

        write_fixture("firecracker_partial_fail.json", &envelope);
    }

    #[test]
    fn generate_healthy_unsupported_compatibility() {
        let facts = NodeFacts {
            cpu: CpuFacts { architecture: "x86_64".to_string(), cores: 8, vendor_id: Some("Intel".to_string()) },
            memory: MemoryFacts { total_mb: 16384 },
            os: OsFacts { hostname: "x86-storage-only-01".to_string(), name: "Ubuntu".to_string(), kernel_version: Some("6.8.0".to_string()) },
        };

        let class = CompatibilityClassification {
            class: CompatibilityClassType::Unsupported,
            reason_codes: vec!["CLASS_UNSUPPORTED_NO_KVM".to_string(), "CLASS_STORAGE_ONLY".to_string()],
        };

        let capabilities = vec![
            NodeCapability {
                feature: "kvm_vm_launch".to_string(),
                state: SupportState::Unsupported,
                provenance: Provenance::Observed,
                reason_code: Some("KVM_MISSING".to_string()),
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            }
        ];

        let envelope = SchedulerEnvelope::new("x86-storage-only-01".to_string(), class, facts, capabilities, vec![], NodeHealth { state: HealthState::Healthy, active_alarms: vec![] }, vec![CollectorStatus { collector_name: "MockCollector".to_string(), success: true, duration_ms: 10, error_message: None }]);

        write_fixture("healthy_unsupported_compatibility.json", &envelope);
    }
}
