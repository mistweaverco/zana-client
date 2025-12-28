package semver

import (
	"log"
	"strconv"
	"strings"
)

func trimVersion(version string) string {
	return strings.TrimPrefix(version, "v")
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
	v1 = trimVersion(v1)
	v2 = trimVersion(v2)

	// Handle empty versions - if either is empty, return false
	if v1 == "" || v2 == "" {
		return false
	}

	// Split version strings into parts (major, minor, patch)
	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	// Normalize empty parts to "0" before padding
	for i := range v1Parts {
		if v1Parts[i] == "" {
			v1Parts[i] = "0"
		}
	}
	for i := range v2Parts {
		if v2Parts[i] == "" {
			v2Parts[i] = "0"
		}
	}

	// Ensure both versions have exactly 3 parts (major.minor.patch)
	for len(v1Parts) < 3 {
		v1Parts = append(v1Parts, "0")
	}
	for len(v2Parts) < 3 {
		v2Parts = append(v2Parts, "0")
	}

	// Compare each part (major, minor, patch)
	for i := 0; i < 3; i++ {
		v1Num, err1 := strconv.Atoi(v1Parts[i])
		v2Num, err2 := strconv.Atoi(v2Parts[i])

		if err1 != nil || err2 != nil {
			log.Println("Invalid version format.", err1, err2)
			log.Println("v1: ", v1)
			log.Println("v2: ", v2)
			return false
		}

		if v2Num > v1Num {
			return true
		} else if v2Num < v1Num {
			return false
		}
	}

	// If all parts are equal, the versions are the same
	return false
}
