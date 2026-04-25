package runtime

import "github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"

func resolvePrimaryDisk(spec launch.LaunchSpec) (string, string) {
	if len(spec.Storage) > 0 {
		for _, s := range spec.Storage {
			if s.MountPoint == "/" || s.MountPoint == "" {
				format := s.Format
				if format == "" {
					format = "raw"
				}
				return s.HostPath, format
			}
		}
		format := spec.Storage[0].Format
		if format == "" {
			format = "raw"
		}
		return spec.Storage[0].HostPath, format
	}
	if spec.ImageReference != "" {
		return spec.ImageReference, "qcow2"
	}
	return "", ""
}

func resolveFirecrackerDisks(spec launch.LaunchSpec) (kernel string, rootfs string) {
	if len(spec.Storage) > 0 {
		for _, s := range spec.Storage {
			if s.MountPoint == "/" {
				rootfs = s.HostPath
			} else if s.MountPoint == "/boot/vmlinux" || (s.Format == "raw" && s.ReadOnly) {
				kernel = s.HostPath
			}
		}
		if kernel != "" && rootfs != "" {
			return kernel, rootfs
		}
	}
	return spec.KernelImagePath, spec.RootfsPath
}
