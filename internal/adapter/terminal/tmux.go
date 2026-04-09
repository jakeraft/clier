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
	"github.com/jakeraft/clier/internal/domain"
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

func (t *TmuxTerminal) Launch(runID, planPath string, plan *apprun.RunPlan, members []domain.MemberPlan) error {
	if len(members) == 0 {
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
			_ = t.unregisterRun(runID, sess)
		}
	}()

	// tmux keeps an index to the persisted run plan. The plan file remains
	// the canonical runtime state; tmux env only makes it discoverable by run ID.
	if err := t.registerRun(runID, sess, planPath); err != nil {
		return fmt.Errorf("register run: %w", err)
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
	plan, err := t.loadPlan(runID)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	teamMemberID, err := apprun.ParseTeamMemberID(memberID)
	if err != nil {
		return err
	}
	member, ok := plan.FindMember(teamMemberID)
	if !ok {
		return fmt.Errorf("member %s not found in run plan", memberID)
	}
	return t.sendKeys(plan.Session, strconv.Itoa(member.Window), text)
}

func (t *TmuxTerminal) Terminate(runID string) error {
	plan, err := t.loadPlan(runID)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}

	sess := plan.Session
	// Gracefully exit each agent before killing the session.
	t.exitAllWindows(sess)
	_, _ = t.runFn("kill-session", "-t", sess)
	_ = t.unregisterRun(runID, sess)
	return nil
}

func (t *TmuxTerminal) Attach(runID string, memberID *string) error {
	plan, err := t.loadPlan(runID)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	sess := plan.Session

	if memberID != nil {
		teamMemberID, err := apprun.ParseTeamMemberID(*memberID)
		if err != nil {
			return err
		}
		member, ok := plan.FindMember(teamMemberID)
		if !ok {
			return fmt.Errorf("member %s not found in run plan", *memberID)
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

func (t *TmuxTerminal) registerRun(runID, sess, planPath string) error {
	if _, err := t.runFn("set-environment", "-g", runEnvKey(sess), runID); err != nil {
		return err
	}
	if _, err := t.runFn("set-environment", "-g", runPlanEnvKey(runID), planPath); err != nil {
		return err
	}
	return nil
}

func (t *TmuxTerminal) unregisterRun(runID, sess string) error {
	_, _ = t.runFn("set-environment", "-g", "-u", runEnvKey(sess))
	_, _ = t.runFn("set-environment", "-g", "-u", runPlanEnvKey(runID))
	return nil
}

func (t *TmuxTerminal) loadPlan(runID string) (*apprun.RunPlan, error) {
	planPath, err := t.lookupEnv(runPlanEnvKey(runID))
	if err != nil {
		return nil, err
	}
	plan, err := apprun.LoadPlanFromPath(planPath)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func (t *TmuxTerminal) lookupEnv(key string) (string, error) {
	out, err := t.runFn("show-environment", "-g", key)
	if err != nil {
		return "", err
	}
	prefix := key + "="
	if !strings.HasPrefix(out, prefix) {
		return "", fmt.Errorf("unexpected tmux env value for %s", key)
	}
	value := strings.TrimPrefix(out, prefix)
	if value == "" {
		return "", fmt.Errorf("empty tmux env value for %s", key)
	}
	return value, nil
}

// ensureSessionClosedHook registers a global tmux hook (idempotent) that
// handles cleanup for any clier run. It looks up the full run ID from
// a tmux server env var keyed by session name, then calls "clier run stop".
func (t *TmuxTerminal) ensureSessionClosedHook() error {
	hookCmd := `run-shell 'ID=$(tmux show-environment -g CLIER_RUN_#{hook_session_name} 2>/dev/null | cut -d= -f2); [ -n "$ID" ] && clier run stop "$ID"'`
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

func runPlanEnvKey(runID string) string {
	return "CLIER_RUN_PLAN_" + runID
}
