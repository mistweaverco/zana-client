package providers

// treeSitterDependencyInstallSuccessCount counts successful Install() calls made from
// tree-sitter inherit resolution (nested grammar installs), for CLI summaries.
var treeSitterDependencyInstallSuccessCount int

// ResetTreeSitterDependencyInstallSuccessCount clears the counter (call once per install command).
func ResetTreeSitterDependencyInstallSuccessCount() {
	treeSitterDependencyInstallSuccessCount = 0
}

func noteTreeSitterDependencyInstallSuccess() {
	treeSitterDependencyInstallSuccessCount++
}

// ConsumeTreeSitterDependencyInstallSuccessCount returns successful dependency installs
// since the last reset, then clears the counter.
func ConsumeTreeSitterDependencyInstallSuccessCount() int {
	n := treeSitterDependencyInstallSuccessCount
	treeSitterDependencyInstallSuccessCount = 0
	return n
}
