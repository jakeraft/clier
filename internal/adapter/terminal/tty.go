package terminal

import (
	"os"

	"golang.org/x/term"
)

// isTerminal reports whether f is connected to an interactive terminal.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
