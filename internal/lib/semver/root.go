package semver

import (
	"log"
	"strconv"
	"strings"
)

func trimVersion(version string) string {
	return strings.TrimPrefix(version, "v")
}

// splitCoreAndPreRelease takes a version string and splits it into:
// - core: major.minor.patch (with missing parts padded with zeros)
// - pre:  optional pre-release identifier (e.g. "dev.20260225.1")
// Build metadata (the part after '+') is ignored for comparison purposes.
func splitCoreAndPreRelease(version string) ([]string, string) {
	// Remove leading "v" prefix if present
	version = trimVersion(version)

	if version == "" {
		return []string{}, ""
	}

	// Strip build metadata (everything after '+')
	if idx := strings.Index(version, "+"); idx != -1 {
		version = version[:idx]
	}

	pre := ""
	// Split out pre-release (everything after first '-')
	if idx := strings.Index(version, "-"); idx != -1 {
		pre = version[idx+1:]
		version = version[:idx]
	}

	coreParts := strings.Split(version, ".")

	// Normalize empty parts to "0" before padding
	for i := range coreParts {
		if coreParts[i] == "" {
			coreParts[i] = "0"
		}
	}

	// Ensure we always have exactly 3 parts (major.minor.patch)
	for len(coreParts) < 3 {
		coreParts = append(coreParts, "0")
	}

	return coreParts[:3], pre
}

// isNumeric returns true if the given string consists only of digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// IsNonNumericPreRelease reports whether the given version string has a
// pre-release part whose identifiers are not all purely numeric.
func IsNonNumericPreRelease(version string) bool {
	_, pre := splitCoreAndPreRelease(version)
	if pre == "" {
		return false
	}
	return !isAllNumericPreRelease(pre)
}

// isAllNumericPreRelease returns true if all dot-separated identifiers in the
// pre-release string are purely numeric (e.g. "1" or "1.2.3").
func isAllNumericPreRelease(pre string) bool {
	if pre == "" {
		return false
	}
	parts := strings.Split(pre, ".")
	for _, p := range parts {
		if !isNumeric(p) {
			return false
		}
	}
	return true
}

// comparePreRelease compares two pre-release identifiers according to SemVer rules,
// with one tweak: non-numeric prereleases (e.g. "alpha", "beta", "dev.2026...")
// are treated as an "unstable track" that should not be downgraded to the
// corresponding stable release when you're already on that track.
// Returns:
// -1 if pre1 < pre2
//
//	0 if pre1 == pre2
//	1 if pre1 > pre2
func comparePreRelease(pre1, pre2 string) int {
	// No pre-release means higher precedence than any pre-release
	if pre1 == "" && pre2 == "" {
		return 0
	}
	if pre2 == "" {
		// v1 has prerelease, v2 is stable.
		// - purely numeric pre-release (e.g. "1", "2.3") is treated as lower than stable
		// - non-numeric pre-release (e.g. "dev.2026", "alpha") is treated as higher
		//   so that we don't "downgrade" from an unstable track to stable.
		if isAllNumericPreRelease(pre1) {
			return -1
		}
		return 1
	}
	if pre1 == "" {
		// v1 is stable, v2 has prerelease: stable is always higher
		return 1
	}

	s1 := strings.Split(pre1, ".")
	s2 := strings.Split(pre2, ".")

	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(s1) && i >= len(s2) {
			break
		}
		if i >= len(s1) {
			// shorter set has lower precedence
			return -1
		}
		if i >= len(s2) {
			return 1
		}

		id1 := s1[i]
		id2 := s2[i]

		numeric1 := isNumeric(id1)
		numeric2 := isNumeric(id2)

		if numeric1 && numeric2 {
			n1, err1 := strconv.Atoi(id1)
			n2, err2 := strconv.Atoi(id2)
			if err1 != nil || err2 != nil {
				// Fall back to string comparison on unexpected parse error
				if id1 < id2 {
					return -1
				}
				if id1 > id2 {
					return 1
				}
				continue
			}
			if n1 < n2 {
				return -1
			}
			if n1 > n2 {
				return 1
			}
			continue
		}

		// Numeric identifiers always have lower precedence than non-numeric
		if numeric1 && !numeric2 {
			return -1
		}
		if !numeric1 && numeric2 {
			return 1
		}

		// Both non-numeric: use lexicographical order
		if id1 < id2 {
			return -1
		}
		if id1 > id2 {
			return 1
		}
	}

	return 0
}

// compareVersions compares two full version strings (including optional pre-release)
// and returns:
// -1 if v1 < v2
//
//	0 if v1 == v2
//	1 if v1 > v2
func compareVersions(v1, v2 string) int {
	v1Core, v1Pre := splitCoreAndPreRelease(v1)
	v2Core, v2Pre := splitCoreAndPreRelease(v2)

	// Compare each core part (major, minor, patch)
	for i := 0; i < 3; i++ {
		v1Num, err1 := strconv.Atoi(v1Core[i])
		v2Num, err2 := strconv.Atoi(v2Core[i])

		if err1 != nil || err2 != nil {
			log.Println("Invalid version format.", err1, err2)
			log.Println("v1: ", v1)
			log.Println("v2: ", v2)
			return 0
		}

		if v1Num < v2Num {
			return -1
		} else if v1Num > v2Num {
			return 1
		}
	}

	// Core parts equal, compare pre-release
	return comparePreRelease(v1Pre, v2Pre)
}

// IsGreater compares two semver version strings and
// returns true if the second argument is greater than the first.
// So given versions v1 and v2, it returns true if v2 > v1.
// With acutal values that would be:
// IsGreater("1.2.3", "1.2.4") returns true
// IsGreater("1.2.3", "1.2.3") returns false
// IsGreater("1.2.3", "1.2.2") returns false
// IsGreater("1.2.3", "1.3.0") returns true
func IsGreater(v1, v2 string) bool {
	// Handle empty versions - if either is empty, return false
	if v1 == "" || v2 == "" {
		return false
	}
	// Return true if v2 > v1 according to full SemVer comparison
	return compareVersions(v1, v2) == -1
}
