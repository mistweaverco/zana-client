package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// IsGreater compares two semver version strings and
// returns true if the second version is greater than the first.
func IsGreater(v1, v2 string) bool {
	// Split version strings into parts (major, minor, patch)
	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

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
			fmt.Println("Invalid version format.")
			fmt.Println("v1: ", v1)
			fmt.Println("v2: ", v2)
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
