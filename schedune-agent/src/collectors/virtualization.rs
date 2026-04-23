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

        // 1. KVM deep check
        let kvm_path = Path::new("/dev/kvm");
        let kvm_exists = kvm_path.exists();
        let kvm_openable = OpenOptions::new().read(true).write(true).open(kvm_path).is_ok();

        capabilities.push(NodeCapability {
            feature: "kvm_vm_launch".to_string(),
            state: if kvm_openable { SupportState::Supported } else if kvm_exists { SupportState::Unavailable } else { SupportState::Unsupported },
            provenance: Provenance::Observed,
            reason_code: if kvm_openable { Some("KVM_OPENABLE".to_string()) } else if kvm_exists { Some("KVM_NOT_OPENABLE_PERMS".to_string()) } else { Some("KVM_MISSING".to_string()) },
            observed_at_sec: now_sec,
            stale_after_sec: Some(default_stale_after),
        });

        // 2. Firecracker prerequisites (KVM + cgroups + tun/tap)
        let tun_exists = Path::new("/dev/net/tun").exists();
        let cgroup_v2 = Path::new("/sys/fs/cgroup/cgroup.controllers").exists();

        let fc_supported = kvm_openable && tun_exists && cgroup_v2;
        capabilities.push(NodeCapability {
            feature: "firecracker_launch".to_string(),
            state: if fc_supported { SupportState::Supported } else { SupportState::Unavailable },
            provenance: Provenance::Observed,
            reason_code: if fc_supported { Some("FC_PREREQS_MET".to_string()) } else { Some("FC_PREREQS_MISSING".to_string()) },
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
