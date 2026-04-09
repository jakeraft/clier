package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	apprun "github.com/jakeraft/clier/internal/app/run"
)

// TmuxTerminal manages agent terminals using tmux.
// One tmux session per clier run, one window per member.
type TmuxTerminal struct {
	runFn    func(args ...string) (string, error)
	attachFn func(sess string) error
	sleep    func(d time.Duration)
}

func NewTmuxTerminal() *TmuxTerminal {
	t := &TmuxTerminal{}
	t.runFn = t.defaultRun
	t.attachFn = t.defaultAttach
	t.sleep = time.Sleep
	return t
}

func (t *TmuxTerminal) Launch(plan *apprun.RunPlan) error {
	if len(plan.Members) == 0 {
		return errors.New("no members to launch")
	}

	sess := plan.Session

	// Create tmux session (first window is created automatically).
	if _, err := t.runFn("new-session", "-d", "-s", sess); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_, _ = t.runFn("kill-session", "-t", sess)
		}
	}()

	// Force base-index 0 on this session so window indices are predictable,
	// regardless of user's global tmux config.
	_, _ = t.runFn("set-option", "-t", sess, "base-index", "0")

	for i, m := range plan.Members {
		win := strconv.Itoa(i)

		if i > 0 {
			if _, err := t.runFn("new-window", "-t", sess); err != nil {
				return fmt.Errorf("create window: %w", err)
			}
		}

		if err := t.setupMemberWindow(sess, win, m); err != nil {
			return err
		}
	}

	// Wait for all members to be ready before returning.
	for i, m := range plan.Members {
		if m.Command == "" {
			continue
		}
		if err := t.waitReady(sess, strconv.Itoa(i), 60*time.Second); err != nil {
			return fmt.Errorf("wait ready %s: %w", m.Name, err)
		}
	}

	success = true
	return nil
}

func (t *TmuxTerminal) Send(plan *apprun.RunPlan, teamMemberID int64, text string) error {
	member, ok := plan.FindMember(teamMemberID)
	if !ok {
		return fmt.Errorf("member %d not found in run plan", teamMemberID)
	}
	return t.sendKeys(plan.Session, strconv.Itoa(member.Window), text)
}

func (t *TmuxTerminal) Terminate(plan *apprun.RunPlan) error {
	sess := plan.Session
	// Gracefully exit each agent before killing the session.
	t.exitAllWindows(sess)
	_, _ = t.runFn("kill-session", "-t", sess)
	return nil
}

func (t *TmuxTerminal) Attach(plan *apprun.RunPlan, memberID *int64) error {
	sess := plan.Session

	if memberID != nil {
		member, ok := plan.FindMember(*memberID)
		if !ok {
			return fmt.Errorf("member %d not found in run plan", *memberID)
		}
		if _, err := t.runFn("select-window", "-t", sess+":"+strconv.Itoa(member.Window)); err != nil {
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

func (t *TmuxTerminal) setupMemberWindow(sess, win string, m apprun.MemberTerminal) error {
	if _, err := t.runFn("rename-window", "-t", sess+":"+win, m.Name); err != nil {
		return fmt.Errorf("rename window: %w", err)
	}
	if m.Command != "" {
		if err := t.sendKeys(sess, win, m.Command); err != nil {
			return fmt.Errorf("send command: %w", err)
		}
	}
	return nil
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
