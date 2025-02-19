package shell_out

import (
	"os"
	"os/exec"
)

func ShellOut(command string, args []string, dir string, env []string) (int, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if env != nil {
		env = append(env, os.Environ()...)
		cmd.Env = append(cmd.Env, env...)
	}
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), err
		}
		return -1, err
	}
	return 0, nil
}
