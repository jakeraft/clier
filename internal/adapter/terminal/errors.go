package terminal

// Sentinel error types from the terminal adapter. Error() returns a
// debug-shaped string only — user-facing rendering is owned by the CLI
// message catalog.

// ErrNoTTY is returned when an interactive command is run without a
// real terminal attached to stdin.
type ErrNoTTY struct{}

func (*ErrNoTTY) Error() string { return "[terminal: no_tty]" }

// ErrSessionGone is returned when the underlying terminal session no
// longer exists (e.g., the run was stopped or never started).
type ErrSessionGone struct {
	Session string
}

func (e *ErrSessionGone) Error() string {
	if e == nil || e.Session == "" {
		return "[terminal: session_gone]"
	}
	return "[terminal: session_gone session=" + e.Session + "]"
}
