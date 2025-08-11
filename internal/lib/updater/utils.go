package updater

import "github.com/mistweaverco/zana-client/internal/lib/shell_out"

type CheckRequirementsResult struct {
	HasNPM             bool `json:"hasNPM"`
	HasPython          bool `json:"hasPython"`
	HasPythonDistutils bool `json:"hasPythonDistutils"`
	HasGo              bool `json:"hasGo"`
}

// CheckRequirements checks if the system meets the requirements for running the updater.
// It checks for:
// - The presence of the `npm` command
// - The presence of the `python` command
// - The availability of `distutils` for Python packages (e.g. node-gyp needs this)
// - The presence of the `go` command
// If any of these requirements are not met,
// it returns a `CheckRequirementsResult` with the respective fields set to false.

func CheckRequirements() CheckRequirementsResult {

	result := CheckRequirementsResult{
		HasNPM:             shell_out.HasCommand("npm", []string{"--version"}, nil),
		HasPython:          shell_out.HasCommand("python", []string{"--version"}, nil),
		HasPythonDistutils: shell_out.HasCommand("python", []string{"-c", "import distutils"}, nil),
		HasGo:              shell_out.HasCommand("go", []string{"version"}, nil),
	}

	return result
}
