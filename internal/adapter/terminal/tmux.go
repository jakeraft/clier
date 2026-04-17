package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	apprun "github.com/jakeraft/clier/internal/app/run"
	"github.com/jakeraft/clier/internal/domain"
)

// errReadyTimeout is wrapped by Launch into a KindRunStartTimeout
// Fault. Kept as a package-level sentinel so the wait loop can stay a
// pure helper without constructing user-facing strings.
var errReadyTimeout = stringError("ready marker not observed before deadline")

type stringError string

func (e stringError) Error() string { return string(e) }

// TmuxTerminal manages agent terminals using tmux.
// One tmux session per clier run, one window per agent.
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
	if len(plan.Agents) == 0 {
		return &domain.Fault{
			Kind:    domain.KindWorkingCopyIncomplete,
			Subject: map[string]string{"detail": "no runnable agents in plan"},
		}
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

	for i, m := range plan.Agents {
		win := strconv.Itoa(i)

		if i > 0 {
			if _, err := t.runFn("new-window", "-t", sess); err != nil {
				return fmt.Errorf("create window: %w", err)
			}
		}

		if err := t.setupAgentWindow(sess, win, m); err != nil {
			return err
		}
	}

	// Wait for all agents to be ready before returning.
	for i, m := range plan.Agents {
		if m.Command == "" {
			continue
		}
		if err := t.waitReady(sess, strconv.Itoa(i), 60*time.Second, m.AgentType); err != nil {
			return &domain.Fault{
				Kind:    domain.KindRunStartTimeout,
				Subject: map[string]string{"agent": m.ID},
				Cause:   err,
			}
		}
	}

	success = true
	return nil
}

func (t *TmuxTerminal) Send(plan *apprun.RunPlan, agentName string, text string) error {
	agent, ok := plan.FindAgent(agentName)
	if !ok {
		return &domain.Fault{
			Kind:    domain.KindInternal,
			Subject: map[string]string{"detail": "agent " + agentName + " not found in run plan"},
		}
	}
	if err := t.sendKeys(plan.Session, strconv.Itoa(agent.Window), text); err != nil {
		return wrapSessionError(plan.Session, err)
	}
	return nil
}

func (t *TmuxTerminal) Terminate(plan *apprun.RunPlan) error {
	sess := plan.Session
	// Gracefully exit each agent before killing the session.
	t.exitAllWindows(sess, plan.Agents)
	_, _ = t.runFn("kill-session", "-t", sess)
	return nil
}

func (t *TmuxTerminal) Attach(plan *apprun.RunPlan, agentName *string) error {
	sess := plan.Session

	if agentName != nil {
		agent, ok := plan.FindAgent(*agentName)
		if !ok {
			return &domain.Fault{
				Kind:    domain.KindInternal,
				Subject: map[string]string{"detail": "agent " + *agentName + " not found in run plan"},
			}
		}
		if _, err := t.runFn("select-window", "-t", sess+":"+strconv.Itoa(agent.Window)); err != nil {
			return wrapSessionError(sess, err)
		}
	}

	if err := t.attachFn(sess); err != nil {
		return wrapSessionError(sess, err)
	}
	return nil
}

// exitAllWindows sends the agent-specific exit command to every agent window.
func (t *TmuxTerminal) exitAllWindows(sess string, agents []apprun.AgentTerminal) {
	for _, m := range agents {
		profile, err := domain.ProfileFor(m.AgentType)
		if err != nil || profile.ExitCommand == "" {
			continue
		}
		_ = t.sendKeys(sess, strconv.Itoa(m.Window), profile.ExitCommand)
	}
}

func (t *TmuxTerminal) setupAgentWindow(sess, win string, m apprun.AgentTerminal) error {
	if _, err := t.runFn("rename-window", "-t", sess+":"+win, m.ID); err != nil {
		return fmt.Errorf("rename window: %w", err)
	}
	if m.Command != "" {
		if err := t.sendKeys(sess, win, m.Command); err != nil {
			return fmt.Errorf("send command: %w", err)
		}
	}
	return nil
}

// waitReady polls the pane title until the agent's TUI marker appears.
func (t *TmuxTerminal) waitReady(sess, win string, timeout time.Duration, agentType string) error {
	profile, err := domain.ProfileFor(agentType)
	if err != nil {
		return err
	}
	if profile.ReadyMarker == "" {
		return nil
	}
	target := sess + ":" + win
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		title, err := t.runFn("display-message", "-p", "-t", target, "#{pane_title}")
		if err == nil && strings.Contains(title, profile.ReadyMarker) {
			return nil
		}
		t.sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%w (after %v)", errReadyTimeout, timeout)
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
	if !isTerminal(os.Stdin) {
		return &ErrNoTTY{}
	}
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

// wrapSessionError translates tmux "session not found" failures into the
// adapter-neutral ErrSessionGone so the CLI presenter can show a hint
// without leaking tmux internals. Other tmux errors pass through.
func wrapSessionError(sess string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "can't find session") ||
		strings.Contains(msg, "session not found") ||
		strings.Contains(msg, "no server running") {
		return &ErrSessionGone{Session: sess}
	}
	return err
}
