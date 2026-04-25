package domain

import (
	"regexp"
	"strconv"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

var versionRegex = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)

type parsedVersion struct {
	major int
	minor int
	patch int
}

func parseVersion(raw string) (*parsedVersion, bool) {
	matches := versionRegex.FindStringSubmatch(raw)
	if len(matches) < 3 {
		return nil, false
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, false
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, false
	}
	patch := 0
	if len(matches) > 3 && matches[3] != "" {
		patch, err = strconv.Atoi(matches[3])
		if err != nil {
			return nil, false
		}
	}

	return &parsedVersion{major, minor, patch}, true
}

func compareVersions(a, b *parsedVersion) int {
	if a.major != b.major {
		if a.major < b.major {
			return -1
		}
		return 1
	}
	if a.minor != b.minor {
		if a.minor < b.minor {
			return -1
		}
		return 1
	}
	if a.patch != b.patch {
		if a.patch < b.patch {
			return -1
		}
		return 1
	}
	return 0
}

func checkVersionRequirement(observedRaw string, req *launch.RuntimeVersionRequirement) (bool, string) {
	if req == nil || (req.MinimumVersion == "" && req.ExactVersion == "") {
		return true, ""
	}

	if observedRaw == "" {
		return false, schema.ReasonErrLaunchRuntimeVersionUnknown
	}

	observed, ok := parseVersion(observedRaw)
	if !ok {
		return false, schema.ReasonErrLaunchRuntimeVersionUnparseable
	}

	if req.ExactVersion != "" {
		exact, ok := parseVersion(req.ExactVersion)
		if !ok {
			return false, schema.ReasonErrLaunchRuntimeVersionUnparseable
		}
		if compareVersions(observed, exact) != 0 {
			return false, schema.ReasonErrLaunchRuntimeVersionMismatch
		}
	}

	if req.MinimumVersion != "" {
		min, ok := parseVersion(req.MinimumVersion)
		if !ok {
			return false, schema.ReasonErrLaunchRuntimeVersionUnparseable
		}
		if compareVersions(observed, min) < 0 {
			return false, schema.ReasonErrLaunchRuntimeVersionTooOld
		}
	}

	return true, ""
}
