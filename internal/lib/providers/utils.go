package providers

import "github.com/mistweaverco/zana-client/internal/lib/shell_out"

type CheckRequirementsResult struct {
	HasNPM             bool `json:"hasNPM"`
	HasPython          bool `json:"hasPython"`
	HasPythonDistutils bool `json:"hasPythonDistutils"`
	HasGo              bool `json:"hasGo"`
	HasCargo           bool `json:"hasCargo"`
}

// CheckRequirements checks if the system meets the requirements for running providers.
func CheckRequirements() CheckRequirementsResult {
	// Prefer python3 over python for modern Python installations
	hasPython3 := shell_out.HasCommand("python3", []string{"--version"}, nil)
	hasPython := shell_out.HasCommand("python", []string{"--version"}, nil)
	hasPythonCmd := hasPython3 || hasPython

	// Check for distutils or setuptools (distutils was deprecated in Python 3.10 and removed in 3.12+)
	// setuptools includes a vendored distutils, so checking for setuptools covers both cases
	var hasPythonDistutils bool
	if hasPython3 {
		// Try setuptools first (works with Python 3.12+), then distutils (works with older versions)
		hasPythonDistutils = shell_out.HasCommand("python3", []string{"-c", "import setuptools"}, nil) ||
			shell_out.HasCommand("python3", []string{"-c", "import distutils"}, nil)
	} else if hasPython {
		// Fallback to python command
		hasPythonDistutils = shell_out.HasCommand("python", []string{"-c", "import setuptools"}, nil) ||
			shell_out.HasCommand("python", []string{"-c", "import distutils"}, nil)
	}

	result := CheckRequirementsResult{
		HasNPM:             shell_out.HasCommand("npm", []string{"--version"}, nil),
		HasPython:          hasPythonCmd,
		HasPythonDistutils: hasPythonDistutils,
		HasGo:              shell_out.HasCommand("go", []string{"version"}, nil),
		HasCargo:           shell_out.HasCommand("cargo", []string{"--version"}, nil),
	}
	return result
}