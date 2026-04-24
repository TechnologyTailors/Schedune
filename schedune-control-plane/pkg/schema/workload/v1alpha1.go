package workload

// WorkloadIntent is the V1Alpha1 contract for requesting compute from the Schedune Control Plane.
type WorkloadIntent struct {
	SchemaVersion string `json:"schema_version" binding:"required,eq=v1alpha1"`
	WorkloadID    string `json:"workload_id" binding:"required"`
	TenantID      string `json:"tenant_id" binding:"required"`

	// Hard Requirements
	RuntimeClass         string `json:"runtime_class" binding:"required,oneof=VirtualMachine MicroVM Container"`
	RequiredArchitecture string `json:"required_architecture" binding:"required,oneof=aarch64 x86_64 any"`
	MaxTelemetryAgeSec   int64  `json:"max_telemetry_age_sec" binding:"required,gt=0"`

	RequiresKVM bool `json:"requires_kvm"`
	RequiresTPM bool `json:"requires_tpm"`

	// Artifact specific requirements
	KernelImagePath string `json:"kernel_image_path"`
	RootfsPath      string `json:"rootfs_path"`

	// Backend preferences
	RuntimeBackendPreference string `json:"runtime_backend_preference"`
	AllowBackendFallback     bool   `json:"allow_backend_fallback"`

	// Explicitly Forbidden States
	ForbiddenConstraints []string `json:"forbidden_constraints"`

	// Target Pools
	RequiredCompatibilityClasses []string `json:"required_compatibility_classes" binding:"required,min=1"`

	// Soft Preferences (used for scoring, not filtering)
	PreferredFeatures []string `json:"preferred_features"`
}
