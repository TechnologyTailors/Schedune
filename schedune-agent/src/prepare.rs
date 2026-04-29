use serde::{Deserialize, Serialize};
use std::env;
use std::path::Path;

// -----------------------------------------------------------------------------
// Models
// -----------------------------------------------------------------------------

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct StorageAttachmentSpec {
    pub volume_id: Option<String>,
    pub host_path: String,
    #[serde(default)]
    pub read_only: bool,
    pub format: String,
    pub mount_point: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct NetworkAttachmentSpec {
    pub network_id: Option<String>,
    pub r#type: String,
    pub host_device: String,
    pub mac_address: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SecurityContextSpec {
    #[serde(default)]
    pub privileged: bool,
    pub seccomp_profile: Option<String>,
    pub apparmor_profile: Option<String>,
    #[serde(default)]
    pub drop_capabilities: Vec<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct RuntimeVersionRequirement {
    pub minimum_version: Option<String>,
    pub exact_version: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct LaunchSpec {
    pub schema_version: String,
    pub workload_id: String,
    pub tenant_id: String,
    pub node_id: String,
    pub runtime_class: String,
    pub architecture: String,

    pub image_reference: Option<String>,
    pub kernel_image_path: Option<String>,
    pub rootfs_path: Option<String>,
    #[serde(default)]
    pub network_attachments: Vec<String>,

    #[serde(default)]
    pub storage: Vec<StorageAttachmentSpec>,
    #[serde(default)]
    pub networks: Vec<NetworkAttachmentSpec>,
    pub security: Option<SecurityContextSpec>,

    pub vcpu: u32,
    pub memory_mb: u64,
    pub launch_mode: String,
    pub runtime_backend_preference: Option<String>,
    #[serde(default)]
    pub allow_backend_fallback: bool,
    pub runtime_version: Option<RuntimeVersionRequirement>,
}

#[derive(Debug, Serialize, Deserialize, PartialEq, Clone)]
pub enum LaunchPreparationStatus {
    PendingNodeAgent,
    Failed,
    Success,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct PreparedQemuLaunch {
    pub binary_path: String,
    pub artifact_path: String,
    pub command_args: Vec<String>,
    pub control_socket_path: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct PreparedCloudHypervisorLaunch {
    pub binary_path: String,
    pub artifact_path: String,
    pub command_args: Vec<String>,
    pub control_socket_path: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct PreparedFirecrackerLaunch {
    pub binary_path: String,
    pub kernel_image_path: String,
    pub rootfs_path: String,
    pub command_args: Vec<String>,
    pub control_socket_path: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct PreparedLaunch {
    pub runtime_backend: String,
    pub memory_mb: u64,
    pub vcpu: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub startup_grace_sec: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub kvm_qemu: Option<PreparedQemuLaunch>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cloud_hypervisor: Option<PreparedCloudHypervisorLaunch>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub firecracker: Option<PreparedFirecrackerLaunch>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct LaunchPreparationResult {
    pub schema_version: String,
    pub status: LaunchPreparationStatus,
    pub node_id: String,
    pub backend: String,
    pub is_preparable: bool,
    #[serde(default)]
    pub blocking_reason_codes: Vec<String>,
    #[serde(default)]
    pub warnings: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub prepared_launch: Option<PreparedLaunch>,
}

// -----------------------------------------------------------------------------
// Logic
// -----------------------------------------------------------------------------

const ERR_LAUNCH_MISSING_ARTIFACT: &str = "ERR_LAUNCH_MISSING_ARTIFACT";
const ERR_LAUNCH_BACKEND_NOT_SUPPORTED: &str = "ERR_LAUNCH_BACKEND_NOT_SUPPORTED";
const ERR_LAUNCH_MISSING_CAPABILITY_QEMU_BINARY: &str = "ERR_LAUNCH_MISSING_CAPABILITY_QEMU_BINARY";
const ERR_LAUNCH_MISSING_CAPABILITY_CH_BINARY: &str = "ERR_LAUNCH_MISSING_CAPABILITY_CH_BINARY";
const ERR_LAUNCH_MISSING_CAPABILITY_FC_BINARY: &str = "ERR_LAUNCH_MISSING_CAPABILITY_FC_BINARY";
const ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL: &str =
    "ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL";
const ERR_LAUNCH_UNSAFE_WORKLOAD_ID: &str = "ERR_LAUNCH_UNSAFE_WORKLOAD_ID";

fn is_safe_id(id: &str) -> bool {
    if id.is_empty() || id == "." || id == ".." || id.contains('/') || id.contains('\\') {
        return false;
    }
    id.chars()
        .all(|c| c.is_ascii_alphanumeric() || c == '-' || c == '_')
}

// Default path resolver
fn default_find_in_path(executable: &str) -> Option<String> {
    if let Ok(paths) = env::var("PATH") {
        for path in env::split_paths(&paths) {
            let cand = Path::new(&path).join(executable);
            if cand.is_file() {
                return Some(cand.to_string_lossy().to_string());
            }
        }
    }
    None
}

fn default_artifact_exists(path: &str) -> bool {
    Path::new(path).exists()
}

pub fn prepare_launch(spec: LaunchSpec) -> LaunchPreparationResult {
    prepare_launch_internal(spec, &default_find_in_path, &default_artifact_exists)
}

fn prepare_launch_internal(
    spec: LaunchSpec,
    find_bin: &dyn Fn(&str) -> Option<String>,
    artifact_exists: &dyn Fn(&str) -> bool,
) -> LaunchPreparationResult {
    let warnings = vec![];
    let mut backend = "".to_string();

    if !is_safe_id(&spec.workload_id) {
        return fail_prep(
            &spec,
            "unknown",
            vec![ERR_LAUNCH_UNSAFE_WORKLOAD_ID.to_string()],
            vec![],
        );
    }

    // Determine backend
    let mut candidates = vec![];
    if spec.runtime_class == "MicroVM" {
        candidates.push("firecracker".to_string());
    } else {
        if let Some(pref) = &spec.runtime_backend_preference {
            if pref == "kvm_qemu" || pref == "cloud_hypervisor" {
                candidates.push(pref.clone());
                if spec.allow_backend_fallback {
                    if pref == "cloud_hypervisor" {
                        candidates.push("kvm_qemu".to_string());
                    } else {
                        candidates.push("cloud_hypervisor".to_string());
                    }
                }
            } else {
                return fail_prep(
                    &spec,
                    "unknown",
                    vec![ERR_LAUNCH_BACKEND_NOT_SUPPORTED.to_string()],
                    warnings,
                );
            }
        }
        if candidates.is_empty() {
            candidates = vec!["cloud_hypervisor".to_string(), "kvm_qemu".to_string()];
        }
    }

    if candidates.is_empty() {
        return fail_prep(
            &spec,
            "unknown",
            vec![ERR_LAUNCH_BACKEND_NOT_SUPPORTED.to_string()],
            warnings,
        );
    }

    let mut last_fail_reasons = vec![];

    for cand in candidates {
        backend = cand.clone();
        match cand.as_str() {
            "kvm_qemu" => match prepare_kvm_qemu(&spec, find_bin, artifact_exists) {
                Ok(prep) => return success_prep(&spec, cand, prep, warnings),
                Err(e) => {
                    last_fail_reasons.push(e);
                }
            },
            "cloud_hypervisor" => {
                match prepare_cloud_hypervisor(&spec, find_bin, artifact_exists) {
                    Ok(prep) => return success_prep(&spec, cand, prep, warnings),
                    Err(e) => {
                        last_fail_reasons.push(e);
                    }
                }
            }
            "firecracker" => match prepare_firecracker(&spec, find_bin, artifact_exists) {
                Ok(prep) => return success_prep(&spec, cand, prep, warnings),
                Err(e) => {
                    last_fail_reasons.push(e);
                }
            },
            _ => {
                last_fail_reasons.push(ERR_LAUNCH_BACKEND_NOT_SUPPORTED.to_string());
            }
        }
    }

    fail_prep(&spec, &backend, last_fail_reasons, warnings)
}

fn fail_prep(
    spec: &LaunchSpec,
    backend: &str,
    reasons: Vec<String>,
    warnings: Vec<String>,
) -> LaunchPreparationResult {
    LaunchPreparationResult {
        schema_version: "v1alpha1".to_string(),
        status: LaunchPreparationStatus::Failed,
        node_id: spec.node_id.clone(),
        backend: backend.to_string(),
        is_preparable: false,
        blocking_reason_codes: reasons,
        warnings,
        prepared_launch: None,
    }
}

fn success_prep(
    spec: &LaunchSpec,
    backend: String,
    prep: PreparedLaunch,
    warnings: Vec<String>,
) -> LaunchPreparationResult {
    LaunchPreparationResult {
        schema_version: "v1alpha1".to_string(),
        status: LaunchPreparationStatus::Success,
        node_id: spec.node_id.clone(),
        backend,
        is_preparable: true,
        blocking_reason_codes: vec![],
        warnings,
        prepared_launch: Some(prep),
    }
}

fn resolve_primary_disk(spec: &LaunchSpec) -> Option<(String, String)> {
    // (path, format)
    let mut primary: Option<&StorageAttachmentSpec> = None;
    let mut fallback: Option<&StorageAttachmentSpec> = None;

    for s in &spec.storage {
        if fallback.is_none() {
            fallback = Some(s);
        }
        if s.mount_point.is_none()
            || s.mount_point.as_deref() == Some("/")
            || s.mount_point.as_deref() == Some("")
        {
            primary = Some(s);
            break;
        }
    }

    if let Some(p) = primary.or(fallback) {
        let fmt = if p.format.is_empty() {
            "raw".to_string()
        } else {
            p.format.clone()
        };
        return Some((p.host_path.clone(), fmt));
    }

    if let Some(img) = &spec.image_reference {
        return Some((img.clone(), "qcow2".to_string()));
    }
    None
}

fn generate_socket_path(spec: &LaunchSpec, backend_token: &str) -> String {
    format!(
        "var/run/schedune/{}/{}.sock",
        spec.workload_id, backend_token
    )
}

fn prepare_kvm_qemu(
    spec: &LaunchSpec,
    find_bin: &dyn Fn(&str) -> Option<String>,
    artifact_exists: &dyn Fn(&str) -> bool,
) -> Result<PreparedLaunch, String> {
    let qemu_bin = format!("qemu-system-{}", spec.architecture);
    let binary_path =
        find_bin(&qemu_bin).ok_or_else(|| ERR_LAUNCH_MISSING_CAPABILITY_QEMU_BINARY.to_string())?;

    let (disk_path, disk_format) =
        resolve_primary_disk(spec).ok_or_else(|| ERR_LAUNCH_MISSING_ARTIFACT.to_string())?;

    if !artifact_exists(&disk_path) {
        return Err(ERR_LAUNCH_MISSING_ARTIFACT.to_string());
    }

    let socket_path = generate_socket_path(spec, "qemu");

    let args = vec![
        "-enable-kvm".to_string(),
        "-m".to_string(),
        format!("{}", spec.memory_mb),
        "-smp".to_string(),
        format!("{}", spec.vcpu),
        "-drive".to_string(),
        format!("file={},format={}", disk_path, disk_format),
        "-nographic".to_string(),
        "-qmp".to_string(),
        format!("unix:{},server,nowait", socket_path),
    ];

    Ok(PreparedLaunch {
        runtime_backend: "kvm_qemu".to_string(),
        memory_mb: spec.memory_mb,
        vcpu: spec.vcpu,
        startup_grace_sec: Some(5),
        kvm_qemu: Some(PreparedQemuLaunch {
            binary_path,
            artifact_path: disk_path,
            command_args: args,
            control_socket_path: Some(socket_path),
        }),
        cloud_hypervisor: None,
        firecracker: None,
    })
}

fn prepare_cloud_hypervisor(
    spec: &LaunchSpec,
    find_bin: &dyn Fn(&str) -> Option<String>,
    artifact_exists: &dyn Fn(&str) -> bool,
) -> Result<PreparedLaunch, String> {
    let binary_path = find_bin("cloud-hypervisor")
        .ok_or_else(|| ERR_LAUNCH_MISSING_CAPABILITY_CH_BINARY.to_string())?;

    let (disk_path, _) =
        resolve_primary_disk(spec).ok_or_else(|| ERR_LAUNCH_MISSING_ARTIFACT.to_string())?;

    if !artifact_exists(&disk_path) {
        return Err(ERR_LAUNCH_MISSING_ARTIFACT.to_string());
    }

    let socket_path = generate_socket_path(spec, "cloudhypervisor");

    let args = vec![
        "--memory".to_string(),
        format!("size={}M", spec.memory_mb),
        "--cpus".to_string(),
        format!("boot={}", spec.vcpu),
        "--disk".to_string(),
        format!("path={}", disk_path),
        "--api-socket".to_string(),
        socket_path.clone(),
    ];

    Ok(PreparedLaunch {
        runtime_backend: "cloud_hypervisor".to_string(),
        memory_mb: spec.memory_mb,
        vcpu: spec.vcpu,
        startup_grace_sec: Some(3),
        kvm_qemu: None,
        cloud_hypervisor: Some(PreparedCloudHypervisorLaunch {
            binary_path,
            artifact_path: disk_path,
            command_args: args,
            control_socket_path: Some(socket_path),
        }),
        firecracker: None,
    })
}

fn prepare_firecracker(
    spec: &LaunchSpec,
    find_bin: &dyn Fn(&str) -> Option<String>,
    artifact_exists: &dyn Fn(&str) -> bool,
) -> Result<PreparedLaunch, String> {
    let binary_path = find_bin("firecracker")
        .ok_or_else(|| ERR_LAUNCH_MISSING_CAPABILITY_FC_BINARY.to_string())?;

    let mut kernel = spec.kernel_image_path.clone();
    let mut rootfs = spec.rootfs_path.clone();

    for storage in &spec.storage {
        if storage.mount_point.as_deref() == Some("/") || storage.format == "ext4" {
            if rootfs.is_none() {
                rootfs = Some(storage.host_path.clone());
            }
        } else if storage.format == "raw" && storage.read_only && kernel.is_none() {
            kernel = Some(storage.host_path.clone());
        }
    }

    let kernel = kernel.ok_or_else(|| ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL.to_string())?;
    let rootfs = rootfs.ok_or_else(|| ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL.to_string())?;

    if !artifact_exists(&kernel) || !artifact_exists(&rootfs) {
        return Err(ERR_LAUNCH_MISSING_ARTIFACT.to_string());
    }

    let socket_path = generate_socket_path(spec, "firecracker");
    let config_path = format!("var/run/schedune/{}/fc-config.json", spec.workload_id);

    let args = vec![
        "--api-sock".to_string(),
        socket_path.clone(),
        "--config-file".to_string(),
        config_path,
    ];

    Ok(PreparedLaunch {
        runtime_backend: "firecracker".to_string(),
        memory_mb: spec.memory_mb,
        vcpu: spec.vcpu,
        startup_grace_sec: Some(2),
        kvm_qemu: None,
        cloud_hypervisor: None,
        firecracker: Some(PreparedFirecrackerLaunch {
            binary_path,
            kernel_image_path: kernel,
            rootfs_path: rootfs,
            command_args: args,
            control_socket_path: Some(socket_path),
        }),
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    fn make_test_spec() -> LaunchSpec {
        LaunchSpec {
            schema_version: "v1alpha1".to_string(),
            workload_id: "test-wl-123".to_string(),
            tenant_id: "tenant-1".to_string(),
            node_id: "node-1".to_string(),
            runtime_class: "VirtualMachine".to_string(),
            architecture: "x86_64".to_string(),
            image_reference: None,
            kernel_image_path: None,
            rootfs_path: None,
            network_attachments: vec![],
            storage: vec![],
            networks: vec![],
            security: None,
            vcpu: 2,
            memory_mb: 1024,
            launch_mode: "DryRun".to_string(),
            runtime_backend_preference: None,
            allow_backend_fallback: false,
            runtime_version: None,
        }
    }

    #[test]
    fn test_unsafe_workload_id() {
        let mut spec = make_test_spec();
        spec.workload_id = "../etc/passwd".to_string();
        let res = prepare_launch_internal(spec, &|_| None, &|_| true);
        assert_eq!(res.status, LaunchPreparationStatus::Failed);
        assert!(res
            .blocking_reason_codes
            .contains(&"ERR_LAUNCH_UNSAFE_WORKLOAD_ID".to_string()));
    }

    #[test]
    fn test_unsupported_backend_preference() {
        let mut spec = make_test_spec();
        spec.runtime_backend_preference = Some("docker".to_string());
        let res = prepare_launch_internal(spec, &|_| None, &|_| true);
        assert_eq!(res.status, LaunchPreparationStatus::Failed);
        assert!(res
            .blocking_reason_codes
            .contains(&"ERR_LAUNCH_BACKEND_NOT_SUPPORTED".to_string()));
    }

    #[test]
    fn test_missing_artifact_kvm_qemu() {
        let mut spec = make_test_spec();
        spec.runtime_backend_preference = Some("kvm_qemu".to_string());
        spec.storage = vec![StorageAttachmentSpec {
            volume_id: None,
            host_path: "/does/not/exist.qcow2".to_string(),
            read_only: false,
            format: "qcow2".to_string(),
            mount_point: None,
        }];

        // Mock binary exists, but artifact does not
        let res = prepare_launch_internal(
            spec,
            &|bin| {
                if bin == "qemu-system-x86_64" {
                    Some("/usr/bin/qemu".to_string())
                } else {
                    None
                }
            },
            &|_| false, // artifact doesn't exist
        );

        assert_eq!(res.status, LaunchPreparationStatus::Failed);
        assert!(res
            .blocking_reason_codes
            .contains(&"ERR_LAUNCH_MISSING_ARTIFACT".to_string()));
    }

    #[test]
    fn test_success_kvm_qemu() {
        let mut spec = make_test_spec();
        spec.runtime_backend_preference = Some("kvm_qemu".to_string());
        spec.storage = vec![StorageAttachmentSpec {
            volume_id: None,
            host_path: "/test.qcow2".to_string(),
            read_only: false,
            format: "qcow2".to_string(),
            mount_point: None,
        }];

        let res = prepare_launch_internal(
            spec,
            &|bin| {
                if bin == "qemu-system-x86_64" {
                    Some("/usr/bin/qemu-system-x86_64".to_string())
                } else {
                    None
                }
            },
            &|path| path == "/test.qcow2",
        );

        assert_eq!(res.status, LaunchPreparationStatus::Success);
        let prep = res.prepared_launch.unwrap();
        assert_eq!(prep.runtime_backend, "kvm_qemu");
        assert_eq!(prep.startup_grace_sec, Some(5));
        let kvm = prep.kvm_qemu.unwrap();
        assert_eq!(kvm.binary_path, "/usr/bin/qemu-system-x86_64");
        assert_eq!(kvm.command_args[0], "-enable-kvm");
        assert_eq!(
            kvm.control_socket_path.unwrap(),
            "var/run/schedune/test-wl-123/qemu.sock"
        );
    }

    #[test]
    fn test_missing_artifact_firecracker() {
        let mut spec = make_test_spec();
        spec.runtime_class = "MicroVM".to_string();

        let res = prepare_launch_internal(
            spec,
            &|bin| {
                if bin == "firecracker" {
                    Some("/usr/bin/firecracker".to_string())
                } else {
                    None
                }
            },
            &|_| true,
        );

        assert_eq!(res.status, LaunchPreparationStatus::Failed);
        assert!(res
            .blocking_reason_codes
            .contains(&"ERR_LAUNCH_INVALID_FIRECRACKER_ARTIFACT_MODEL".to_string()));
    }

    #[test]
    fn test_backend_fallback() {
        let mut spec = make_test_spec();
        spec.runtime_backend_preference = Some("cloud_hypervisor".to_string());
        spec.allow_backend_fallback = true;
        spec.image_reference = Some("/test.qcow2".to_string());

        let res = prepare_launch_internal(
            spec,
            &|bin| {
                if bin == "qemu-system-x86_64" {
                    Some("/usr/bin/qemu-system-x86_64".to_string())
                } else {
                    None
                }
            },
            &|_| true,
        );

        assert_eq!(res.status, LaunchPreparationStatus::Success);
        let prep = res.prepared_launch.unwrap();
        assert_eq!(prep.runtime_backend, "kvm_qemu"); // fell back to qemu
    }
}
