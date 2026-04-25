package schema

var knownReasonCodes = map[string]struct{}{
	ReasonCapKvmOpenable:                            {},
	ReasonCapKvmMissing:                             {},
	ReasonCapKvmNotOpenablePerms:                    {},
	ReasonCapQemuBinaryPresent:                      {},
	ReasonCapQemuBinaryMissing:                      {},
	ReasonCapQemuUnsupportedArch:                    {},
	ReasonCapCloudHypervisorBinaryPresent:           {},
	ReasonCapCloudHypervisorBinaryMissing:           {},
	ReasonCapCloudHypervisorReady:                   {},
	ReasonCapCloudHypervisorPrereqsMissing:          {},
	ReasonCapFirecrackerBinaryPresent:               {},
	ReasonCapFirecrackerBinaryMissing:               {},
	ReasonCapFirecrackerTunReady:                    {},
	ReasonCapFirecrackerTunMissing:                  {},
	ReasonCapFirecrackerCgroupsReady:                {},
	ReasonCapFirecrackerCgroupsMissing:              {},
	ReasonCapFirecrackerReady:                       {},
	ReasonCapFirecrackerPrereqsMissing:              {},
	ReasonRejectArchitectureMismatch:                {},
	ReasonRejectCompatibilityClassMismatch:          {},
	ReasonRejectMissingKVM:                          {},
	ReasonRejectMissingTPM:                          {},
	ReasonRejectNodeUnhealthy:                       {},
	ReasonRejectTelemetryStale:                      {},
	ReasonRejectForbiddenConstraintPrefix:           {},
	ReasonErrLaunchArchMismatch:                     {},
	ReasonErrLaunchBackendNotSupported:              {},
	ReasonErrLaunchMissingArtifact:                  {},
	ReasonErrLaunchInvalidStorageFormat:             {},
	ReasonErrLaunchInvalidFirecrackerArtifactModel:  {},
	ReasonErrLaunchMissingCapabilityCloudHypervisor: {},
	ReasonErrLaunchMissingCapabilityChBinary:        {},
	ReasonErrLaunchMissingCapabilityFcBinary:        {},
	ReasonErrLaunchMissingCapabilityFcTun:           {},
	ReasonErrLaunchMissingCapabilityFcCgroups:       {},
	ReasonErrLaunchMissingCapabilityKvmQemu:         {},
	ReasonErrLaunchMissingCapabilityQemuBinary:      {},
	ReasonErrLaunchMissingCapabilitySeccomp:         {},
	ReasonErrLaunchMissingCapabilityNamespaces:      {},
	ReasonWarnDeprecatedImageReference:              {},
	ReasonWarnDeprecatedNetworkAttachments:          {},
	ReasonErrPreparationFailed:                      {},
	ReasonErrNodeNotFound:                           {},
	ReasonErrValidationFailed:                       {},
	ReasonErrExecRuntimeSpawnFailed:                 {},
	ReasonErrExecRuntimeCrashed:                     {},
	ReasonErrExecRuntimeExitedEarly:                 {},
	ReasonErrTermSignalFailed:                       {},
	ReasonErrReadyProbeFailed:                       {},
	ReasonErrReadyTimeout:                           {},
	ReasonErrReadyQemuSocketTimeout:                 {},
	ReasonErrReadyCloudHypervisorSocketTimeout:      {},
	ReasonErrReadyBackendUnsupported:                {},
	ReasonErrReconcileProcessMissing:                {},
	ReasonErrReconcileStatusUnreadable:              {},
	ReasonErrRecoveryExecutionMissing:               {},
	ReasonErrRecoveryReassociationAmbiguous:         {},
	ReasonErrRecoveryRehydrateFailed:                {},
	ReasonErrRecoveryStaleHandle:                    {},
	ReasonRecoveryConfirmed:                         {},
	ReasonRecoveryTerminatedMissing:                 {},
	ReasonErrOrphanPossibleScheduneProcess:          {},
	ReasonErrOrphanStaleArtifactState:               {},
	ReasonErrOrphanUnmanagedBackendProcess:          {},
}

// KnownReasonCodes returns a map of all registered reason codes.
func KnownReasonCodes() map[string]struct{} {
	copied := make(map[string]struct{}, len(knownReasonCodes))
	for k, v := range knownReasonCodes {
		copied[k] = v
	}
	return copied
}

// IsKnownReasonCode checks if a given reason code is registered.
func IsKnownReasonCode(code string) bool {
	_, exists := knownReasonCodes[code]
	return exists
}
