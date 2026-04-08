package terminal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/domain"
)

// RefStore persists terminal refs across CLI invocations.
// The refs map is opaque — each adapter stores its own keys.
type RefStore interface {
	SaveRefs(ctx context.Context, runID, memberID string, refs map[string]string) error
	GetRefs(ctx context.Context, runID, memberID string) (map[string]string, error)
	GetRunRefs(ctx context.Context, runID string) (map[string]string, error)
	DeleteRefs(ctx context.Context, runID string) error
}

// TmuxTerminal manages agent terminals using tmux.
// One tmux session per clier run, one window per member.
type TmuxTerminal struct {
	refs     RefStore
	runFn    func(args ...string) (string, error)
	attachFn func(sess string) error
	sleep    func(d time.Duration)
}

func NewTmuxTerminal(refs RefStore) *TmuxTerminal {
	t := &TmuxTerminal{refs: refs}
	t.runFn = t.defaultRun
	t.attachFn = t.defaultAttach
	t.sleep = time.Sleep
	return t
}

func (t *TmuxTerminal) Launch(runID, runName string, members []domain.MemberPlan) error {
	if len(members) == 0 {
		return errors.New("no members to launch")
	}

	sess := runName

	// Create tmux session (first window is created automatically).
	if _, err := t.runFn("new-session", "-d", "-s", sess); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_, _ = t.runFn("kill-session", "-t", sess)
			_, _ = t.runFn("set-environment", "-g", "-u", runEnvKey(sess))
			_ = t.deleteRefs(runID)
		}
	}()

	// Store full run ID in tmux server env so the session-closed hook
	// can look it up by session name.
	if _, err := t.runFn("set-environment", "-g", runEnvKey(sess), runID); err != nil {
		return fmt.Errorf("set run env: %w", err)
	}

	// Force base-index 0 on this session so window indices are predictable,
	// regardless of user's global tmux config.
	_, _ = t.runFn("set-option", "-t", sess, "base-index", "0")

	for i, m := range members {
		win := strconv.Itoa(i)

		if i > 0 {
			if _, err := t.runFn("new-window", "-t", sess); err != nil {
				return fmt.Errorf("create window: %w", err)
			}
		}

		if err := t.setupMemberWindow(sess, win, m); err != nil {
			return err
		}

		if err := t.saveRefs(runID, strconv.FormatInt(m.TeamMemberID, 10), sess, win); err != nil {
			return fmt.Errorf("save refs: %w", err)
		}
	}

	// Wait for all members to be ready before returning.
	for i, m := range members {
		if m.Terminal.Command == "" {
			continue
		}
		if err := t.waitReady(sess, strconv.Itoa(i), 60*time.Second); err != nil {
			return fmt.Errorf("wait ready %s: %w", m.MemberName, err)
		}
	}

	// Register global session-closed hook for reverse sync (idempotent).
	// A single hook handles all clier runs: looks up the full run ID
	// from a tmux server env var keyed by session name, and calls stop.
	if err := t.ensureSessionClosedHook(); err != nil {
		return fmt.Errorf("set session-closed hook: %w", err)
	}

	success = true
	return nil
}

func (t *TmuxTerminal) Send(runID, memberID, text string) error {
	refs, err := t.getRefs(runID, memberID)
	if err != nil {
		return fmt.Errorf("get refs for %s: %w", memberID, err)
	}
	return t.sendKeys(refs["session"], refs["window"], text)
}

func (t *TmuxTerminal) Terminate(runID string) error {
	refs, err := t.getRunRefs(runID)
	if err == nil {
		sess := refs["session"]
		// Gracefully exit each agent before killing the session.
		t.exitAllWindows(sess)
		_, _ = t.runFn("kill-session", "-t", sess)
		_, _ = t.runFn("set-environment", "-g", "-u", runEnvKey(sess))
	}
	return t.deleteRefs(runID)
}

func (t *TmuxTerminal) Attach(runID string, memberID *string) error {
	refs, err := t.getRunRefs(runID)
	if err != nil {
		return fmt.Errorf("get run refs: %w", err)
	}
	sess := refs["session"]

	if memberID != nil {
		memberRefs, err := t.getRefs(runID, *memberID)
		if err != nil {
			return fmt.Errorf("get member refs: %w", err)
		}
		if _, err := t.runFn("select-window", "-t", sess+":"+memberRefs["window"]); err != nil {
			return fmt.Errorf("select window: %w", err)
		}
	}

	return t.attachFn(sess)
}

// exitAllWindows sends /exit to every window so agents shut down gracefully.
func (t *TmuxTerminal) exitAllWindows(sess string) {
	out, err := t.runFn("list-windows", "-t", sess, "-F", "#{window_index}")
	if err != nil {
		return
	}
	for win := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		win = strings.TrimSpace(win)
		if win != "" {
			_ = t.sendKeys(sess, win, "/exit")
		}
	}
}

func (t *TmuxTerminal) setupMemberWindow(sess, win string, m domain.MemberPlan) error {
	if _, err := t.runFn("rename-window", "-t", sess+":"+win, m.MemberName); err != nil {
		return fmt.Errorf("rename window: %w", err)
	}
	if m.Terminal.Command != "" {
		if err := t.sendKeys(sess, win, m.Terminal.Command); err != nil {
			return fmt.Errorf("send command: %w", err)
		}
	}
	return nil
}

// persistence — delegated to RefStore

func (t *TmuxTerminal) saveRefs(runID, memberID, sess, win string) error {
	return t.refs.SaveRefs(context.Background(), runID, memberID, map[string]string{
		"session": sess,
		"window":  win,
	})
}

func (t *TmuxTerminal) getRefs(runID, memberID string) (map[string]string, error) {
	return t.refs.GetRefs(context.Background(), runID, memberID)
}

func (t *TmuxTerminal) getRunRefs(runID string) (map[string]string, error) {
	return t.refs.GetRunRefs(context.Background(), runID)
}

func (t *TmuxTerminal) deleteRefs(runID string) error {
	return t.refs.DeleteRefs(context.Background(), runID)
}

// ensureSessionClosedHook registers a global tmux hook (idempotent) that
// handles cleanup for any clier run. It looks up the full run ID from
// a tmux server env var keyed by session name, then calls "clier run stop".
func (t *TmuxTerminal) ensureSessionClosedHook() error {
	hookCmd := `run-shell 'ID=$(tmux show-environment -g CLIER_RUN_#{hook_session_name} 2>/dev/null | cut -d= -f2); [ -n "$ID" ] && clier run stop "$ID" && tmux set-environment -g -u CLIER_RUN_#{hook_session_name}'`
	_, err := t.runFn("set-hook", "-g", "session-closed", hookCmd)
	return err
}

// waitReady polls the pane title until Claude Code's TUI marker appears.
// Claude Code sets the pane title via OSC escape sequences:
// - Braille characters (U+2800-U+28FF) while working/starting
// - Done markers when idle
func (t *TmuxTerminal) waitReady(sess, win string, timeout time.Duration) error {
	target := sess + ":" + win
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		title, err := t.runFn("display-message", "-p", "-t", target, "#{pane_title}")
		if err == nil && hasClaudeMarker(title) {
			return nil
		}
		t.sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("not ready after %v", timeout)
}

// hasClaudeMarker returns true if the pane title indicates Claude Code is running.
func hasClaudeMarker(title string) bool {
	return strings.Contains(title, "Claude")
}

// tmux command helpers

func (t *TmuxTerminal) sendKeys(sess, win, text string) error {
	target := sess + ":" + win
	_, _ = t.runFn("copy-mode", "-q", "-t", target)
	if _, err := t.runFn("send-keys", "-l", "-t", target, text); err != nil {
		return err
	}
	// Claude Code's Ink TUI needs time to process text before Enter.
	// Without this delay, Enter is swallowed. 300ms matches industry practice.
	t.sleep(300 * time.Millisecond)
	_, err := t.runFn("send-keys", "-t", target, "Enter")
	return err
}

func (t *TmuxTerminal) defaultAttach(sess string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", sess)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (t *TmuxTerminal) defaultRun(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tmux %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func runEnvKey(sess string) string {
	return "CLIER_RUN_" + sess
}
