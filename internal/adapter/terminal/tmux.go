package terminal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// RefStore persists terminal refs across CLI invocations.
// The refs map is opaque — each adapter stores its own keys.
type RefStore interface {
	SaveRefs(ctx context.Context, sessionID, memberID string, refs map[string]string) error
	GetRefs(ctx context.Context, sessionID, memberID string) (map[string]string, error)
	GetSessionRefs(ctx context.Context, sessionID string) (map[string]string, error)
	DeleteRefs(ctx context.Context, sessionID string) error
}

// TmuxTerminal manages agent terminals using tmux.
// One tmux session per clier session, one window per member.
type TmuxTerminal struct {
	refs     RefStore
	runFn    func(args ...string) (string, error)
	attachFn func(sess string) error
}

func NewTmuxTerminal(refs RefStore) *TmuxTerminal {
	t := &TmuxTerminal{refs: refs}
	t.runFn = t.defaultRun
	t.attachFn = t.defaultAttach
	return t
}

func (t *TmuxTerminal) Launch(sessionID, sessionName string, members []domain.MemberPlan) error {
	if len(members) == 0 {
		return errors.New("no members to launch")
	}

	sess := tmuxSessionName(sessionID)

	// Create tmux session (first window is created automatically).
	if _, err := t.runFn("new-session", "-d", "-s", sess); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	// Force base-index 0 on this session so window indices are predictable,
	// regardless of user's global tmux config.
	_, _ = t.runFn("set-option", "-t", sess, "base-index", "0")

	success := false
	defer func() {
		if !success {
			_, _ = t.runFn("kill-session", "-t", sess)
			_ = t.deleteRefs(sessionID)
		}
	}()

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

		if err := t.saveRefs(sessionID, m.TeamMemberID, sess, win); err != nil {
			return fmt.Errorf("save refs: %w", err)
		}
	}

	// Register global session-closed hook for reverse sync (idempotent).
	// A single hook handles all clier sessions: matches "clier-*" pattern,
	// extracts session ID from the tmux session name, and calls stop.
	if err := t.ensureSessionClosedHook(); err != nil {
		return fmt.Errorf("set session-closed hook: %w", err)
	}

	success = true
	return nil
}

func (t *TmuxTerminal) Send(sessionID, memberID, text string) error {
	refs, err := t.getRefs(sessionID, memberID)
	if err != nil {
		return fmt.Errorf("get refs for %s: %w", memberID, err)
	}
	return t.sendKeys(refs["session"], refs["window"], text)
}

func (t *TmuxTerminal) Terminate(sessionID string) error {
	refs, err := t.getSessionRefs(sessionID)
	if err == nil {
		sess := refs["session"]
		// Gracefully exit each agent before killing the session.
		t.exitAllWindows(sess)
		_, _ = t.runFn("kill-session", "-t", sess)
	}
	return t.deleteRefs(sessionID)
}

func (t *TmuxTerminal) Attach(sessionID string, memberID *string) error {
	refs, err := t.getSessionRefs(sessionID)
	if err != nil {
		return fmt.Errorf("get session refs: %w", err)
	}
	sess := refs["session"]

	if memberID != nil {
		memberRefs, err := t.getRefs(sessionID, *memberID)
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
	for _, win := range strings.Split(strings.TrimSpace(out), "\n") {
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

func (t *TmuxTerminal) saveRefs(sessionID, memberID, sess, win string) error {
	return t.refs.SaveRefs(context.Background(), sessionID, memberID, map[string]string{
		"session": sess,
		"window":  win,
	})
}

func (t *TmuxTerminal) getRefs(sessionID, memberID string) (map[string]string, error) {
	return t.refs.GetRefs(context.Background(), sessionID, memberID)
}

func (t *TmuxTerminal) getSessionRefs(sessionID string) (map[string]string, error) {
	return t.refs.GetSessionRefs(context.Background(), sessionID)
}

func (t *TmuxTerminal) deleteRefs(sessionID string) error {
	return t.refs.DeleteRefs(context.Background(), sessionID)
}

// ensureSessionClosedHook registers a global tmux hook (idempotent) that
// handles cleanup for any clier session. It matches "clier-*" session names,
// extracts the session ID, and calls "clier session stop".
func (t *TmuxTerminal) ensureSessionClosedHook() error {
	hookCmd := `if-shell -F '#{m:clier-*,#{hook_session_name}}' "run-shell 'clier session stop #{s/clier-//:hook_session_name}'"`
	_, err := t.runFn("set-hook", "-g", "session-closed", hookCmd)
	return err
}

// tmux command helpers

func (t *TmuxTerminal) sendKeys(sess, win, text string) error {
	target := sess + ":" + win
	// Send text literally (-l) to avoid tmux interpreting special characters,
	// then send Enter as a key press separately.
	if _, err := t.runFn("send-keys", "-l", "-t", target, text); err != nil {
		return err
	}
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

func tmuxSessionName(sessionID string) string {
	return "clier-" + sessionID
}
