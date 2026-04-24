use crate::capabilities::{NodeCapability, NodeConstraint, Provenance, SupportState};
use crate::scheduler_contract::CollectorStatus;
use std::path::Path;
use std::fs::OpenOptions;
use std::time::{SystemTime, UNIX_EPOCH};

pub struct VirtualizationCollector;

impl VirtualizationCollector {
    pub fn collect() -> (Vec<NodeCapability>, Vec<NodeConstraint>, CollectorStatus) {
        let start = SystemTime::now();
        let now_sec = start.duration_since(UNIX_EPOCH).unwrap_or_default().as_secs();
        let default_stale_after = now_sec + 300;

        let mut capabilities = Vec::new();
        let constraints = Vec::new();

        // --- KVM Checks ---
        let kvm_path = Path::new("/dev/kvm");
        let kvm_exists = kvm_path.exists();
        let kvm_openable = OpenOptions::new().read(true).write(true).open(kvm_path).is_ok();

        capabilities.push(NodeCapability {
            feature: "kvm_vm_launch".to_string(),
            state: if kvm_openable { SupportState::Supported } else if kvm_exists { SupportState::Unavailable } else { SupportState::Unsupported },
            provenance: Provenance::Observed,
            reason_code: if kvm_openable { Some("CAP_KVM_OPENABLE".to_string()) } else if kvm_exists { Some("CAP_KVM_NOT_OPENABLE_PERMS".to_string()) } else { Some("CAP_KVM_MISSING".to_string()) },
            observed_at_sec: now_sec,
            stale_after_sec: Some(default_stale_after),
        });

        // --- Binary Presence Checks (Mocking standard paths for V0) ---
        let ch_binary_present = Path::new("/usr/bin/cloud-hypervisor").exists() || Path::new("/usr/local/bin/cloud-hypervisor").exists();
        let fc_binary_present = Path::new("/usr/bin/firecracker").exists() || Path::new("/usr/local/bin/firecracker").exists();

        capabilities.push(NodeCapability {
            feature: "cloud_hypervisor_binary_present".to_string(),
            state: if ch_binary_present { SupportState::Supported } else { SupportState::Unsupported },
            provenance: Provenance::Observed,
            reason_code: if ch_binary_present { Some("CAP_CLOUDHYPERVISOR_BINARY_PRESENT".to_string()) } else { Some("CAP_CLOUDHYPERVISOR_BINARY_MISSING".to_string()) },
            observed_at_sec: now_sec,
            stale_after_sec: Some(default_stale_after),
        });

        capabilities.push(NodeCapability {
            feature: "firecracker_binary_present".to_string(),
            state: if fc_binary_present { SupportState::Supported } else { SupportState::Unsupported },
            provenance: Provenance::Observed,
            reason_code: if fc_binary_present { Some("CAP_FIRECRACKER_BINARY_PRESENT".to_string()) } else { Some("CAP_FIRECRACKER_BINARY_MISSING".to_string()) },
            observed_at_sec: now_sec,
            stale_after_sec: Some(default_stale_after),
        });

        // --- Cloud Hypervisor Launch Readiness ---
        let ch_launch_ready = kvm_openable && ch_binary_present;
        capabilities.push(NodeCapability {
            feature: "cloud_hypervisor_launch".to_string(),
            state: if ch_launch_ready { SupportState::Supported } else { SupportState::Unavailable },
            provenance: Provenance::Inferred,
            reason_code: if ch_launch_ready { Some("CAP_CLOUDHYPERVISOR_READY".to_string()) } else { Some("CAP_CLOUDHYPERVISOR_PREREQS_MISSING".to_string()) },
            observed_at_sec: now_sec,
            stale_after_sec: Some(default_stale_after),
        });

        // --- Firecracker Host Prerequisites ---
        let tun_exists = Path::new("/dev/net/tun").exists();
        let cgroup_v2 = Path::new("/sys/fs/cgroup/cgroup.controllers").exists();

        capabilities.push(NodeCapability {
            feature: "firecracker_tun_ready".to_string(),
            state: if tun_exists { SupportState::Supported } else { SupportState::Unsupported },
            provenance: Provenance::Observed,
            reason_code: if tun_exists { Some("CAP_FIRECRACKER_TUN_READY".to_string()) } else { Some("CAP_FIRECRACKER_TUN_MISSING".to_string()) },
            observed_at_sec: now_sec,
            stale_after_sec: Some(default_stale_after),
        });

        capabilities.push(NodeCapability {
            feature: "firecracker_cgroups_ready".to_string(),
            state: if cgroup_v2 { SupportState::Supported } else { SupportState::Unsupported },
            provenance: Provenance::Observed,
            reason_code: if cgroup_v2 { Some("CAP_FIRECRACKER_CGROUPS_READY".to_string()) } else { Some("CAP_FIRECRACKER_CGROUPS_MISSING".to_string()) },
            observed_at_sec: now_sec,
            stale_after_sec: Some(default_stale_after),
        });

        let fc_supported = kvm_openable && tun_exists && cgroup_v2 && fc_binary_present;
        capabilities.push(NodeCapability {
            feature: "firecracker_launch".to_string(),
            state: if fc_supported { SupportState::Supported } else { SupportState::Unavailable },
            provenance: Provenance::Inferred,
            reason_code: if fc_supported { Some("CAP_FIRECRACKER_READY".to_string()) } else { Some("CAP_FIRECRACKER_PREREQS_MISSING".to_string()) },
            observed_at_sec: now_sec,
            stale_after_sec: Some(default_stale_after),
        });

        let status = CollectorStatus {
            collector_name: "VirtualizationCollector".to_string(),
            success: true,
            duration_ms: start.elapsed().unwrap_or_default().as_millis() as u64,
            error_message: None,
        };

        (capabilities, constraints, status)
    }
}
