# Runtime Support

Schedune aims to abstract runtime nuances without obscuring backend-specific validation.

## Supported Runtimes

### KVM / QEMU
- **Class:** `VirtualMachine`
- **Execution:** Supported
- **Artifact Model:** Single monolithic disk image (qcow2, raw).
- **Prerequisites:** `/dev/kvm` read-write access.

### Cloud Hypervisor
- **Class:** `VirtualMachine`
- **Execution:** Supported
- **Artifact Model:** Disk path passed to `--disk`.
- **Prerequisites:** `/dev/kvm` access, `cloud-hypervisor` binary in PATH.

### Firecracker
- **Class:** `MicroVM`
- **Execution:** Validate / Dry-Run (Full execution in development)
- **Artifact Model:** Split artifact requirements. Expects `kernel_image_path` and `rootfs_path`.
- **Prerequisites:** `/dev/kvm`, `/dev/net/tun`, `cgroups v2`, and the `firecracker` binary.

## Runtime Selection Logic

If an operator specifies `VirtualMachine`, Schedune prefers `cloud_hypervisor` if the node capabilities support it. If Cloud Hypervisor is missing, it transparently falls back to `kvm_qemu`, noting the fallback in the scheduling explanation trace.
