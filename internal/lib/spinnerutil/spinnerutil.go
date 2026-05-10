// Package spinnerutil wraps github.com/charmbracelet/huh/spinner for shared CLI patterns.
package spinnerutil

import (
	"os"

	"github.com/charmbracelet/huh/spinner"
	"github.com/mattn/go-isatty"
)

// Run shows a huh spinner with title while action runs.
func Run(title string, action func()) error {
	return spinner.New().Title(title).Action(action).Run()
}

// RunIfTTY runs action inside a spinner only when stderr is a terminal; otherwise runs action with no spinner.
func RunIfTTY(title string, action func()) error {
	return RunWithTTYOrPlain(title, nil, action)
}

// RunWithTTYOrPlain runs action with a spinner when stderr is a TTY; otherwise runs plainBefore (if non-nil) then action.
func RunWithTTYOrPlain(title string, plainBefore func(), action func()) error {
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		if plainBefore != nil {
			plainBefore()
		}
		action()
		return nil
	}
	return Run(title, action)
}
