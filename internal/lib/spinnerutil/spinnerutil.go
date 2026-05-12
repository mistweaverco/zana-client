// Package spinnerutil wraps github.com/charmbracelet/huh/spinner for shared CLI patterns.
package spinnerutil

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/charmbracelet/huh/spinner"
	"github.com/mattn/go-isatty"
)

var spinnerDepth int32

// Run shows a huh spinner with title while action runs.
// When another Run is already active (nested), a second Bubble Tea program would corrupt the
// terminal; nested calls print the title to stderr and run the action without a spinner.
func Run(title string, action func()) error {
	n := atomic.AddInt32(&spinnerDepth, 1)
	defer atomic.AddInt32(&spinnerDepth, -1)
	if n > 1 {
		if isatty.IsTerminal(os.Stderr.Fd()) {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", title)
		}
		action()
		return nil
	}
	return spinner.New().Title(title).Action(action).Run()
}

// RunIfTTY runs action inside a spinner only when stderr is a terminal; otherwise prints the
// title to stderr and runs the action (useful for CI / logs).
func RunIfTTY(title string, action func()) error {
	if !isatty.IsTerminal(os.Stderr.Fd()) {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", title)
		action()
		return nil
	}
	return Run(title, action)
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
