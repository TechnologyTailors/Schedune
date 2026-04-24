package domain

type RecoveryStatus string

const (
	RecoveryPending   RecoveryStatus = "RecoveryPending"
	RecoveryConfirmed RecoveryStatus = "RecoveryConfirmed"
	RecoveryUnknown   RecoveryStatus = "RecoveryUnknown"
	RecoveryOrphaned  RecoveryStatus = "RecoveryOrphaned"
)
