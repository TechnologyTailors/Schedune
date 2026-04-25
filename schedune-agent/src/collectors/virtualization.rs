use crate::capabilities::{NodeCapability, NodeConstraint, Provenance, SupportState};
use crate::scheduler_contract::CollectorStatus;
use std::fs::OpenOptions;
use std::path::Path;
use std::process::{Command, Stdio};
use std::time::{SystemTime, UNIX_EPOCH};

pub struct VirtualizationCollector;

impl VirtualizationCollector {
    fn get_binary_version(path: &str) -> Option<String> {
        let output = Command::new("timeout")
            .arg("1s")
            .arg(path)
            .arg("--version")
            .stdin(Stdio::null())
            .output()
            .ok()?;

        if output.status.success() {
            let stdout = String::from_utf8_lossy(&output.stdout);
            let first_line = stdout.lines().next()?.trim();
            if first_line.is_empty() {
                None
            } else {
                Some(first_line.chars().take(128).collect()) // truncate to reasonable length
            }
        } else {
            None
        }
    }

    fn build_cap(
        feature: &str,
        state: SupportState,
        provenance: Provenance,
        reason_code: &str,
        version: Option<String>,
        now_sec: u64,
        stale_after_sec: u64,
    ) -> NodeCapability {
        NodeCapability {
            feature: feature.to_string(),
            state,
            provenance,
            reason_code: Some(reason_code.to_string()),
            version,
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
            None,
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

        let qemu_version = if qemu_binary_present {
            let path = if Path::new(&format!("/usr/bin/{}", qemu_binary_name)).exists() {
                format!("/usr/bin/{}", qemu_binary_name)
            } else {
                format!("/usr/local/bin/{}", qemu_binary_name)
            };
            Self::get_binary_version(&path)
        } else {
            None
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
            qemu_version,
            now_sec,
            default_stale_after,
        ));

        let ch_binary_present = Path::new("/usr/bin/cloud-hypervisor").exists()
            || Path::new("/usr/local/bin/cloud-hypervisor").exists();
        let fc_binary_present = Path::new("/usr/bin/firecracker").exists()
            || Path::new("/usr/local/bin/firecracker").exists();

        let ch_version = if ch_binary_present {
            let path = if Path::new("/usr/bin/cloud-hypervisor").exists() {
                "/usr/bin/cloud-hypervisor"
            } else {
                "/usr/local/bin/cloud-hypervisor"
            };
            Self::get_binary_version(path)
        } else {
            None
        };

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
            ch_version,
            now_sec,
            default_stale_after,
        ));

        let fc_version = if fc_binary_present {
            let path = if Path::new("/usr/bin/firecracker").exists() {
                "/usr/bin/firecracker"
            } else {
                "/usr/local/bin/firecracker"
            };
            Self::get_binary_version(path)
        } else {
            None
        };

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
            fc_version,
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
            None,
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
            None,
            now_sec,
            default_stale_after,
        ));

        // --- Sandboxing Prerequisites ---
        let seccomp_exists = Path::new("/proc/sys/kernel/seccomp/actions_avail").exists();
        let (seccomp_state, seccomp_reason) = if seccomp_exists {
            (SupportState::Supported, "CAP_SECCOMP_SUPPORTED")
        } else {
            (SupportState::Unsupported, "CAP_SECCOMP_MISSING")
        };

        capabilities.push(Self::build_cap(
            "kernel_seccomp_supported",
            seccomp_state,
            Provenance::Observed,
            seccomp_reason,
            None,
            now_sec,
            default_stale_after,
        ));

        let ns_supported = Path::new("/proc/self/ns/user").exists()
            && Path::new("/proc/self/ns/pid").exists()
            && Path::new("/proc/self/ns/net").exists();
        let (ns_state, ns_reason) = if ns_supported {
            (SupportState::Supported, "CAP_NAMESPACES_SUPPORTED")
        } else {
            (SupportState::Unsupported, "CAP_NAMESPACES_MISSING")
        };

        capabilities.push(Self::build_cap(
            "kernel_namespaces_supported",
            ns_state,
            Provenance::Observed,
            ns_reason,
            None,
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
