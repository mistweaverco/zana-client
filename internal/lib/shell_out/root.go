package shell_out

import (
	"os/exec"
	"runtime"
)

var operatingSystem = runtime.GOOS

// ShellOut runs a command on the operating system
// based on the detected operating system
func ShellOut(args ...string) ([]byte, error) {
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Output()
}
