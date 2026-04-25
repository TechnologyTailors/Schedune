use crate::capabilities::{
    CompatibilityClassType, CompatibilityClassification, CpuFacts, MemoryFacts, NodeCapability,
    NodeConstraint, NodeFacts, OsFacts, Provenance, SupportState,
};
use crate::health::{HealthState, NodeHealth};
use crate::scheduler_contract::CollectorStatus;
use std::path::Path;
use std::time::{SystemTime, UNIX_EPOCH};
use sysinfo::System;

pub struct SystemCollector;

impl SystemCollector {
    pub fn collect() -> (
        CompatibilityClassification,
        NodeFacts,
        Vec<NodeCapability>,
        Vec<NodeConstraint>,
        NodeHealth,
        CollectorStatus,
    ) {
        let start = SystemTime::now();
        let mut sys = System::new_all();
        sys.refresh_all();

        let now_sec = start
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();
        let default_stale_after = now_sec + 300; // 5 minutes

        let hostname = System::host_name().unwrap_or_else(|| "unknown".to_string());
        let architecture = System::cpu_arch().unwrap_or_else(|| "unknown".to_string());
        let os_name = System::name().unwrap_or_else(|| "unknown".to_string());
        let cpu_cores = sys.cpus().len() as u32;
        let total_memory_mb = sys.total_memory() / 1024 / 1024;
        let kernel_version = System::kernel_version();

        // 1. Typed Facts
        let facts = NodeFacts {
            cpu: CpuFacts {
                architecture: architecture.clone(),
                cores: cpu_cores,
                vendor_id: sys.cpus().first().map(|c| c.vendor_id().to_string()),
            },
            memory: MemoryFacts {
                total_mb: total_memory_mb,
            },
            os: OsFacts {
                hostname,
                name: os_name,
                kernel_version,
            },
        };

        // Hardware Probes
        let kvm_available = Path::new("/dev/kvm").exists();
        let tpm_present = Path::new("/sys/class/tpm/").exists();

        // 2. Capabilities (SupportState + Reason Codes)
        let capabilities = vec![
            NodeCapability {
                feature: "kvm_virtualization".to_string(),
                state: if kvm_available {
                    SupportState::Supported
                } else {
                    SupportState::Unsupported
                },
                provenance: Provenance::Observed,
                reason_code: if kvm_available {
                    Some("CAP_KVM_SUPPORTED".to_string())
                } else {
                    Some("CAP_KVM_MISSING".to_string())
                },
                version: None,
                observed_at_sec: now_sec,
                stale_after_sec: Some(default_stale_after),
            },
            NodeCapability {
                feature: "hardware_tpm".to_string(),
                state: if tpm_present {
                    SupportState::Supported
                } else {
                    SupportState::Unsupported
                },
                provenance: Provenance::Observed,
                reason_code: if tpm_present {
                    Some("CAP_TPM_PRESENT".to_string())
                } else {
                    Some("CAP_TPM_MISSING".to_string())
                },
                version: None,
                observed_at_sec: now_sec,
                stale_after_sec: Some(default_stale_after),
            },
            NodeCapability {
                feature: "dpu_offload".to_string(),
                state: SupportState::Unknown,
                provenance: Provenance::Unavailable("PCIe scan not implemented".to_string()),
                reason_code: None,
                version: None,
                observed_at_sec: now_sec,
                stale_after_sec: Some(default_stale_after),
            },
        ];

        let mut constraints = Vec::new();
        let mut reason_codes = Vec::new();
        // 3. Decouple Health from Eligibility
        // If KVM is missing, the host is operationally Healthy (just an OS running normally),
        // but it is Ineligible for the ARM_PROD class.
        let health_state = HealthState::Healthy;
        let active_alarms = Vec::new();

        if !kvm_available {
            constraints.push(NodeConstraint {
                constraint_type: "VirtualizationDisabled".to_string(),
                code: "CONSTRAINT_NO_KVM".to_string(),
                description: "Cannot schedule KVM or Firecracker microVMs on this node."
                    .to_string(),
                observed_value: Some("missing".to_string()),
                expected_value: Some("present".to_string()),
            });
            reason_codes.push("CLASS_UNSUPPORTED_NO_KVM".to_string());
        }

        let is_arm = architecture == "aarch64" || architecture == "arm";
        let is_x86 = architecture == "x86_64";

        let class_type = if is_arm && kvm_available {
            reason_codes.push("CLASS_ARM_PROD_READY".to_string());
            CompatibilityClassType::ArmProduction
        } else if is_x86 && kvm_available {
            reason_codes.push("CLASS_X86_HOLDING_READY".to_string());
            CompatibilityClassType::X86HoldingPool
        } else {
            reason_codes.push("CLASS_UNSUPPORTED_ARCH".to_string());
            CompatibilityClassType::Unsupported
        };

        let compatibility = CompatibilityClassification {
            class: class_type,
            reason_codes,
        };

        let health = NodeHealth {
            state: health_state,
            active_alarms,
        };

        let duration_ms = start.elapsed().unwrap_or_default().as_millis() as u64;
        let status = CollectorStatus {
            collector_name: "SystemCollector".to_string(),
            success: true,
            duration_ms,
            error_message: None,
        };

        (
            compatibility,
            facts,
            capabilities,
            constraints,
            health,
            status,
        )
    }
}
