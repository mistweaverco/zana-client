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
	result := CheckRequirementsResult{
		HasNPM:             shell_out.HasCommand("npm", []string{"--version"}, nil),
		HasPython:          shell_out.HasCommand("python", []string{"--version"}, nil),
		HasPythonDistutils: shell_out.HasCommand("python", []string{"-c", "import distutils"}, nil),
		HasGo:              shell_out.HasCommand("go", []string{"version"}, nil),
		HasCargo:           shell_out.HasCommand("cargo", []string{"--version"}, nil),
	}
	return result
}
