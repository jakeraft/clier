package terminal

import (
	"errors"
	"strings"
	"testing"
	"time"

	apprun "github.com/jakeraft/clier/internal/app/run"
)

// fakeRunner captures tmux commands for verification.
type fakeRunner struct {
	calls       []string
	output      map[string]string
	err         error
	errByPrefix map[string]error
}

func (f *fakeRunner) run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
	for prefix, err := range f.errByPrefix {
		if strings.HasPrefix(key, prefix) {
			return "", err
		}
	}
	if f.err != nil {
		return "", f.err
	}
	for prefix, out := range f.output {
		if strings.HasPrefix(key, prefix) {
			return out, nil
		}
	}
	return "", nil
}

func TestTmuxTerminal_Launch(t *testing.T) {
	runner := &fakeRunner{output: map[string]string{
		"display-message": "✳ Claude Code",
	}}
	tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}

	agents := []apprun.AgentTerminal{
		{Name: "leader", Window: 0, Workspace: "/tmp/leader", Cwd: "/tmp/leader", Command: "echo hello"},
		{Name: "worker", Window: 1, Workspace: "/tmp/worker", Cwd: "/tmp/worker"},
	}
	plan := apprun.NewPlan("s-1", "my-team", agents)

	if err := tm.Launch(plan); err != nil {
		t.Fatalf("Launch: %v", err)
	}

	if !hasCall(runner.calls, "new-session") {
		t.Error("expected new-session call")
	}
	if !hasCall(runner.calls, "set-option") {
		t.Error("expected set-option call for base-index")
	}
	if !hasCall(runner.calls, "new-window") {
		t.Error("expected new-window call for second agent")
	}
	if countCalls(runner.calls, "rename-window") != 2 {
		t.Errorf("expected 2 rename-window calls, got %d", countCalls(runner.calls, "rename-window"))
	}
	if !hasCall(runner.calls, "send-keys") {
		t.Error("expected send-keys call for agent command")
	}
}

func TestTmuxTerminal_Send(t *testing.T) {
	plan := &apprun.RunPlan{
		RunID:   "s-1",
		Session: "my-team-s-1",
		Agents: []apprun.AgentTerminal{{
			Name:      "leader",
			Window:    0,
			Workspace: "/tmp/leader",
			Cwd:       "/tmp/leader",
			Command:   "echo hello",
		}},
	}
	runner := &fakeRunner{}
	tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}

	if err := tm.Send(plan, "leader", "do the work"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("expected 3 calls (copy-mode + send-keys text + send-keys Enter), got %d: %v", len(runner.calls), runner.calls)
	}
	if !strings.Contains(runner.calls[0], "copy-mode") {
		t.Errorf("copy-mode call missing, got: %v", runner.calls)
	}
	if !strings.Contains(runner.calls[1], "-l") || !strings.Contains(runner.calls[1], "do the work") {
		t.Errorf("literal send-keys call missing, got: %s", runner.calls[1])
	}
	if !strings.Contains(runner.calls[2], "Enter") {
		t.Errorf("Enter send-keys missing, got: %s", runner.calls[2])
	}
}

func TestTmuxTerminal_Terminate(t *testing.T) {
	plan := &apprun.RunPlan{
		RunID:   "s-1",
		Session: "my-team-s-1",
		Agents: []apprun.AgentTerminal{{
			Name:   "leader",
			Window: 0,
		}, {
			Name:   "worker",
			Window: 1,
		}, {
			Name:   "reviewer",
			Window: 2,
		}},
	}
	runner := &fakeRunner{}
	tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}

	if err := tm.Terminate(plan); err != nil {
		t.Fatalf("Terminate: %v", err)
	}
	exitCount := 0
	for _, c := range runner.calls {
		if strings.Contains(c, "/exit") {
			exitCount++
		}
	}
	if exitCount != 3 {
		t.Errorf("expected 3 /exit sends, got %d", exitCount)
	}
	if !hasCall(runner.calls, "kill-session") {
		t.Error("expected kill-session call")
	}
}

func TestTmuxTerminal_Terminate_AlreadyDead(t *testing.T) {
	plan := &apprun.RunPlan{
		RunID:   "s-1",
		Session: "my-team-s-1",
		Agents: []apprun.AgentTerminal{{
			Name:   "leader",
			Window: 0,
		}},
	}
	runner := &fakeRunner{
		errByPrefix: map[string]error{
			"list-windows": errors.New("session not found"),
			"kill-session": errors.New("session not found"),
		},
	}
	tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}

	if err := tm.Terminate(plan); err != nil {
		t.Fatalf("Terminate (already dead): %v", err)
	}
}

func TestTmuxTerminal_Attach(t *testing.T) {
	plan := &apprun.RunPlan{
		RunID:   "s-1",
		Session: "my-team-s-1",
		Agents: []apprun.AgentTerminal{{
			Name:   "leader",
			Window: 1,
		}, {
			Name:   "worker",
			Window: 2,
		}},
	}

	t.Run("with agent selects window", func(t *testing.T) {
		runner := &fakeRunner{}
		tm := &TmuxTerminal{runFn: runner.run, attachFn: func(string) error { return nil }, sleep: func(time.Duration) {}}
		agentName := "worker"
		if err := tm.Attach(plan, &agentName); err != nil {
			t.Fatalf("Attach: %v", err)
		}
		if !hasCall(runner.calls, "select-window") {
			t.Error("expected select-window call")
		}
		for _, c := range runner.calls {
			if strings.Contains(c, "select-window") && !strings.Contains(c, ":2") {
				t.Errorf("select-window should target window 2, got: %s", c)
			}
		}
	})

	t.Run("without agent skips select-window", func(t *testing.T) {
		runner := &fakeRunner{}
		tm := &TmuxTerminal{runFn: runner.run, attachFn: func(string) error { return nil }, sleep: func(time.Duration) {}}
		if err := tm.Attach(plan, nil); err != nil {
			t.Fatalf("Attach: %v", err)
		}
		if hasCall(runner.calls, "select-window") {
			t.Error("select-window should not be called without agent")
		}
	})
}

func TestTmuxTerminal_WaitReady_Timeout(t *testing.T) {
	runner := &fakeRunner{output: map[string]string{
		"display-message": "zsh",
	}}
	tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}

	err := tm.waitReady("sess", "0", 10*time.Millisecond, "claude")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("unexpected error: %v", err)
	}
}

func hasCall(calls []string, substr string) bool {
	for _, c := range calls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

func countCalls(calls []string, substr string) int {
	n := 0
	for _, c := range calls {
		if strings.Contains(c, substr) {
			n++
		}
	}
	return n
}
