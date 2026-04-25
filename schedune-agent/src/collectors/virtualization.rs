use crate::capabilities::{NodeCapability, NodeConstraint, Provenance, SupportState};
use crate::scheduler_contract::CollectorStatus;
use std::fs::OpenOptions;
use std::path::Path;
use std::time::{SystemTime, UNIX_EPOCH};

pub struct VirtualizationCollector;

impl VirtualizationCollector {
    fn build_cap(
        feature: &str,
        state: SupportState,
        provenance: Provenance,
        reason_code: &str,
        now_sec: u64,
        stale_after_sec: u64,
    ) -> NodeCapability {
        NodeCapability {
            feature: feature.to_string(),
            state,
            provenance,
            reason_code: Some(reason_code.to_string()),
            observed_at_sec: now_sec,
            stale_after_sec: Some(stale_after_sec),
        }
    }

    pub fn collect() -> (Vec<NodeCapability>, Vec<NodeConstraint>, CollectorStatus) {
        let start = SystemTime::now();
        let now_sec = start
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();
        let default_stale_after = now_sec + 300;

        let mut capabilities = Vec::new();
        let constraints = Vec::new();

        // --- KVM Checks ---
        let kvm_path = Path::new("/dev/kvm");
        let kvm_exists = kvm_path.exists();
        let kvm_openable = OpenOptions::new()
            .read(true)
            .write(true)
            .open(kvm_path)
            .is_ok();

        let (kvm_state, kvm_reason) = if kvm_openable {
            (SupportState::Supported, "CAP_KVM_OPENABLE")
        } else if kvm_exists {
            (SupportState::Unavailable, "CAP_KVM_NOT_OPENABLE_PERMS")
        } else {
            (SupportState::Unsupported, "CAP_KVM_MISSING")
        };

        capabilities.push(Self::build_cap(
            "kvm_vm_launch",
            kvm_state,
            Provenance::Observed,
            kvm_reason,
            now_sec,
            default_stale_after,
        ));

        // --- Binary Presence Checks (Mocking standard paths for V0) ---
        let arch = std::env::consts::ARCH;
        let qemu_binary_name = match arch {
            "x86_64" => "qemu-system-x86_64",
            "aarch64" => "qemu-system-aarch64",
            _ => "",
        };

        let qemu_binary_present = if !qemu_binary_name.is_empty() {
            Path::new(&format!("/usr/bin/{}", qemu_binary_name)).exists()
                || Path::new(&format!("/usr/local/bin/{}", qemu_binary_name)).exists()
        } else {
            false
        };

        let (qemu_bin_state, qemu_bin_reason) = if qemu_binary_present {
            (SupportState::Supported, "CAP_QEMU_BINARY_PRESENT")
        } else if qemu_binary_name.is_empty() {
            (SupportState::Unsupported, "CAP_QEMU_UNSUPPORTED_ARCH")
        } else {
            (SupportState::Unsupported, "CAP_QEMU_BINARY_MISSING")
        };

        capabilities.push(Self::build_cap(
            "qemu_binary_present",
            qemu_bin_state,
            Provenance::Observed,
            qemu_bin_reason,
            now_sec,
            default_stale_after,
        ));

        let ch_binary_present = Path::new("/usr/bin/cloud-hypervisor").exists()
            || Path::new("/usr/local/bin/cloud-hypervisor").exists();
        let fc_binary_present = Path::new("/usr/bin/firecracker").exists()
            || Path::new("/usr/local/bin/firecracker").exists();

        let (ch_bin_state, ch_bin_reason) = if ch_binary_present {
            (
                SupportState::Supported,
                "CAP_CLOUDHYPERVISOR_BINARY_PRESENT",
            )
        } else {
            (
                SupportState::Unsupported,
                "CAP_CLOUDHYPERVISOR_BINARY_MISSING",
            )
        };

        capabilities.push(Self::build_cap(
            "cloud_hypervisor_binary_present",
            ch_bin_state,
            Provenance::Observed,
            ch_bin_reason,
            now_sec,
            default_stale_after,
        ));

        let (fc_bin_state, fc_bin_reason) = if fc_binary_present {
            (SupportState::Supported, "CAP_FIRECRACKER_BINARY_PRESENT")
        } else {
            (SupportState::Unsupported, "CAP_FIRECRACKER_BINARY_MISSING")
        };

        capabilities.push(Self::build_cap(
            "firecracker_binary_present",
            fc_bin_state,
            Provenance::Observed,
            fc_bin_reason,
            now_sec,
            default_stale_after,
        ));

        // --- Cloud Hypervisor Launch Readiness ---
        let ch_launch_ready = kvm_openable && ch_binary_present;
        let (ch_launch_state, ch_launch_reason) = if ch_launch_ready {
            (SupportState::Supported, "CAP_CLOUDHYPERVISOR_READY")
        } else {
            (
                SupportState::Unavailable,
                "CAP_CLOUDHYPERVISOR_PREREQS_MISSING",
            )
        };

        capabilities.push(Self::build_cap(
            "cloud_hypervisor_launch",
            ch_launch_state,
            Provenance::Inferred,
            ch_launch_reason,
            now_sec,
            default_stale_after,
        ));

        // --- Firecracker Host Prerequisites ---
        let tun_exists = Path::new("/dev/net/tun").exists();
        let cgroup_v2 = Path::new("/sys/fs/cgroup/cgroup.controllers").exists();

        let (fc_tun_state, fc_tun_reason) = if tun_exists {
            (SupportState::Supported, "CAP_FIRECRACKER_TUN_READY")
        } else {
            (SupportState::Unsupported, "CAP_FIRECRACKER_TUN_MISSING")
        };

        capabilities.push(Self::build_cap(
            "firecracker_tun_ready",
            fc_tun_state,
            Provenance::Observed,
            fc_tun_reason,
            now_sec,
            default_stale_after,
        ));

        let (fc_cg_state, fc_cg_reason) = if cgroup_v2 {
            (SupportState::Supported, "CAP_FIRECRACKER_CGROUPS_READY")
        } else {
            (SupportState::Unsupported, "CAP_FIRECRACKER_CGROUPS_MISSING")
        };

        capabilities.push(Self::build_cap(
            "firecracker_cgroups_ready",
            fc_cg_state,
            Provenance::Observed,
            fc_cg_reason,
            now_sec,
            default_stale_after,
        ));

        let fc_supported = kvm_openable && tun_exists && cgroup_v2 && fc_binary_present;
        let (fc_launch_state, fc_launch_reason) = if fc_supported {
            (SupportState::Supported, "CAP_FIRECRACKER_READY")
        } else {
            (SupportState::Unavailable, "CAP_FIRECRACKER_PREREQS_MISSING")
        };

        capabilities.push(Self::build_cap(
            "firecracker_launch",
            fc_launch_state,
            Provenance::Inferred,
            fc_launch_reason,
            now_sec,
            default_stale_after,
        ));

        let status = CollectorStatus {
            collector_name: "VirtualizationCollector".to_string(),
            success: true,
            duration_ms: start.elapsed().unwrap_or_default().as_millis() as u64,
            error_message: None,
        };

        (capabilities, constraints, status)
    }
}
