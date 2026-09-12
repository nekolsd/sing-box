package libbox

import (
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/common/badversion"

	"golang.org/x/mod/semver"
)

func CompareSemver(left string, right string) bool {
	leftVersion, leftRevision, loaded := parseComparableSemver(left)
	if !loaded {
		return false
	}
	rightVersion, rightRevision, loaded := parseComparableSemver(right)
	if !loaded {
		return false
	}
	if leftVersion == rightVersion {
		return leftRevision > rightRevision
	}
	return leftVersion.GreaterThan(rightVersion)
}

// forkSuffix marks a nekolsd rebuild of an upstream release. It may be
// followed by "-<n>" (n >= 1) to order rebuilds of the same upstream version;
// a bare suffix is revision 1, and an upstream version without the suffix is
// revision 0.
const forkSuffix = "-nekolsd"

func parseComparableSemver(version string) (badversion.Version, uint64, bool) {
	normalizedVersion := normalizeSemver(version)
	suffixIndex := strings.LastIndex(normalizedVersion, forkSuffix)
	if suffixIndex == -1 {
		if !semver.IsValid(normalizedVersion) {
			return badversion.Version{}, 0, false
		}
		return badversion.Parse(normalizedVersion), 0, true
	}
	baseVersion := normalizedVersion[:suffixIndex]
	if !semver.IsValid(baseVersion) {
		return badversion.Version{}, 0, false
	}
	revision := uint64(1)
	if revisionText := normalizedVersion[suffixIndex+len(forkSuffix):]; revisionText != "" {
		revisionDigits, isRevision := strings.CutPrefix(revisionText, "-")
		if !isRevision {
			return badversion.Version{}, 0, false
		}
		var err error
		revision, err = strconv.ParseUint(revisionDigits, 10, 64)
		if err != nil || revision == 0 {
			return badversion.Version{}, 0, false
		}
	}
	return badversion.Parse(baseVersion), revision, true
}

func normalizeSemver(version string) string {
	trimmedVersion := strings.TrimSpace(version)
	if strings.HasPrefix(trimmedVersion, "v") {
		return trimmedVersion
	}
	return "v" + trimmedVersion
}
