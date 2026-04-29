package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// Tmux is the surface the runner uses to drive tmux. NewSession and
// NewWindow return the physical window index assigned by tmux — that index
// already accounts for the user's `base-index` config, so callers must
// store it in the run plan and reuse it verbatim for tell/attach.
type Tmux interface {
	NewSession(session, windowName, cwd string) (windowIdx int, err error)
	NewWindow(session, name, cwd string) (windowIdx int, err error)
	SendLine(session string, windowIdx int, line string) error
	Attach(session string, windowIdx *int) error
	KillSession(session string) error
	PaneTitle(session string, windowIdx int) (string, error)
	HasSession(session string) (bool, error)
}

// ErrNoTTY is returned by Attach when stdin is not an interactive terminal.
var ErrNoTTY = errors.New("attach requires an interactive terminal (stdin is not a TTY)")

// ErrSessionGone is returned when a tmux operation targets a session that no
// longer exists — typically because the run was stopped or the host
// rebooted between commands.
type ErrSessionGone struct{ Session string }

func (e *ErrSessionGone) Error() string {
	return "tmux session no longer exists: " + e.Session
}

// Real shells out to the `tmux` binary. send-keys sleeps 300ms between the
// literal text and Enter — Claude Code's Ink TUI swallows Enter without it.
type Real struct {
	sleep func(d time.Duration)
}

func New() *Real {
	return &Real{sleep: time.Sleep}
}

func (r *Real) NewSession(session, windowName, cwd string) (int, error) {
	args := []string{"new-session", "-d", "-s", session}
	if windowName != "" {
		args = append(args, "-n", windowName)
	}
	if cwd != "" {
		args = append(args, "-c", cwd)
	}
	if _, err := r.run(args...); err != nil {
		return 0, fmt.Errorf("new-session %s: %w", session, err)
	}
	return r.currentWindowIndex(session)
}

func (r *Real) NewWindow(session, name, cwd string) (int, error) {
	args := []string{"new-window", "-t", session}
	if name != "" {
		args = append(args, "-n", name)
	}
	if cwd != "" {
		args = append(args, "-c", cwd)
	}
	if _, err := r.run(args...); err != nil {
		return 0, wrapSessionGone(session, fmt.Errorf("new-window %s: %w", session, err))
	}
	return r.currentWindowIndex(session)
}

func (r *Real) SendLine(session string, windowIdx int, line string) error {
	target := session + ":" + strconv.Itoa(windowIdx)
	// Best-effort copy-mode escape so send-keys lands in the prompt even if
	// the pane is currently scrolled into copy-mode.
	_, _ = r.run("copy-mode", "-q", "-t", target)

	if _, err := r.run("send-keys", "-l", "-t", target, line); err != nil {
		return wrapSessionGone(session, err)
	}
	r.sleep(300 * time.Millisecond)
	if _, err := r.run("send-keys", "-t", target, "Enter"); err != nil {
		return wrapSessionGone(session, err)
	}
	return nil
}

func (r *Real) Attach(session string, windowIdx *int) error {
	if !isTTY(os.Stdin) {
		return ErrNoTTY
	}
	if windowIdx != nil {
		target := session + ":" + strconv.Itoa(*windowIdx)
		if _, err := r.run("select-window", "-t", target); err != nil {
			return wrapSessionGone(session, err)
		}
	}
	cmd := exec.Command("tmux", "attach-session", "-t", session)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return wrapSessionGone(session, err)
	}
	return nil
}

func (r *Real) KillSession(session string) error {
	if _, err := r.run("kill-session", "-t", session); err != nil {
		// Goal state is "session is gone" — already-gone is success.
		if isSessionGoneMessage(err.Error()) {
			return nil
		}
		return err
	}
	return nil
}

func (r *Real) PaneTitle(session string, windowIdx int) (string, error) {
	target := session + ":" + strconv.Itoa(windowIdx)
	out, err := r.run("display-message", "-p", "-t", target, "#{pane_title}")
	if err != nil {
		return "", wrapSessionGone(session, err)
	}
	return out, nil
}

func (r *Real) HasSession(session string) (bool, error) {
	if _, err := r.run("has-session", "-t", session); err != nil {
		if isSessionGoneMessage(err.Error()) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Real) currentWindowIndex(session string) (int, error) {
	out, err := r.run("display-message", "-p", "-t", session, "#{window_index}")
	if err != nil {
		return 0, wrapSessionGone(session, err)
	}
	idx, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, fmt.Errorf("parse window index %q: %w", out, err)
	}
	return idx, nil
}

func (r *Real) run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tmux %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func wrapSessionGone(session string, err error) error {
	if err == nil {
		return nil
	}
	if isSessionGoneMessage(err.Error()) {
		return &ErrSessionGone{Session: session}
	}
	return err
}

func isSessionGoneMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "can't find session") ||
		strings.Contains(lower, "session not found") ||
		strings.Contains(lower, "no server running") ||
		strings.Contains(lower, "no such session")
}

func isTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
