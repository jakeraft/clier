package terminal

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	apprun "github.com/jakeraft/clier/internal/app/run"
	"github.com/jakeraft/clier/internal/domain"
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

	members := []domain.MemberPlan{
		{TeamMemberID: 1, MemberName: "leader", Terminal: domain.TerminalPlan{Command: "echo hello"}, Workspace: domain.WorkspacePlan{Memberspace: "/tmp/leader"}},
		{TeamMemberID: 2, MemberName: "worker", Terminal: domain.TerminalPlan{}, Workspace: domain.WorkspacePlan{Memberspace: "/tmp/worker"}},
	}
	plan := apprun.NewPlan("s-1", "my-team", members)
	planPath := filepath.Join(t.TempDir(), "s-1.json")

	if err := tm.Launch("s-1", planPath, plan, members); err != nil {
		t.Fatalf("Launch: %v", err)
	}

	if !hasCall(runner.calls, "new-session") {
		t.Error("expected new-session call")
	}
	if !hasCall(runner.calls, "set-option") {
		t.Error("expected set-option call for base-index")
	}
	if !hasCall(runner.calls, "new-window") {
		t.Error("expected new-window call for second member")
	}
	if countCalls(runner.calls, "rename-window") != 2 {
		t.Errorf("expected 2 rename-window calls, got %d", countCalls(runner.calls, "rename-window"))
	}
	if !hasCall(runner.calls, "set-environment -g CLIER_RUN_my-team s-1") {
		t.Error("expected session->run env registration")
	}
	if !hasCall(runner.calls, "set-environment -g CLIER_RUN_PLAN_s-1 "+planPath) {
		t.Error("expected run->plan env registration")
	}
	if !hasCall(runner.calls, "send-keys") {
		t.Error("expected send-keys call for member command")
	}
}

func TestTmuxTerminal_Send(t *testing.T) {
	planPath := writePlan(t, "s-1", "my-team-s-1", []apprun.MemberTerminal{{
		TeamMemberID: 1,
		Name:         "leader",
		Window:       0,
		Memberspace:  "/tmp/leader",
		Cwd:          "/tmp/leader/project",
		Command:      "echo hello",
	}})
	runner := &fakeRunner{output: map[string]string{
		"show-environment -g CLIER_RUN_PLAN_s-1": "CLIER_RUN_PLAN_s-1=" + planPath,
	}}
	tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}

	if err := tm.Send("s-1", "1", "do the work"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(runner.calls) != 4 {
		t.Fatalf("expected 4 calls (show-env + copy-mode + send-keys text + send-keys Enter), got %d: %v", len(runner.calls), runner.calls)
	}
	if !strings.Contains(runner.calls[1], "copy-mode") {
		t.Errorf("copy-mode call missing, got: %v", runner.calls)
	}
	if !strings.Contains(runner.calls[2], "-l") || !strings.Contains(runner.calls[2], "do the work") {
		t.Errorf("literal send-keys call missing, got: %s", runner.calls[2])
	}
	if !strings.Contains(runner.calls[3], "Enter") {
		t.Errorf("Enter send-keys missing, got: %s", runner.calls[3])
	}
}

func TestTmuxTerminal_Terminate(t *testing.T) {
	planPath := writePlan(t, "s-1", "my-team-s-1", []apprun.MemberTerminal{{
		TeamMemberID: 1,
		Name:         "leader",
		Window:       0,
	}, {
		TeamMemberID: 2,
		Name:         "worker",
		Window:       1,
	}, {
		TeamMemberID: 3,
		Name:         "reviewer",
		Window:       2,
	}})
	runner := &fakeRunner{output: map[string]string{
		"show-environment -g CLIER_RUN_PLAN_s-1": "CLIER_RUN_PLAN_s-1=" + planPath,
		"list-windows":                           "0\n1\n2",
	}}
	tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}

	if err := tm.Terminate("s-1"); err != nil {
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
	if !hasCall(runner.calls, "set-environment -g -u CLIER_RUN_my-team-s-1") {
		t.Error("expected session env cleanup")
	}
	if !hasCall(runner.calls, "set-environment -g -u CLIER_RUN_PLAN_s-1") {
		t.Error("expected plan env cleanup")
	}
}

func TestTmuxTerminal_Terminate_AlreadyDead(t *testing.T) {
	planPath := writePlan(t, "s-1", "my-team-s-1", []apprun.MemberTerminal{{
		TeamMemberID: 1,
		Name:         "leader",
		Window:       0,
	}})
	runner := &fakeRunner{
		output: map[string]string{
			"show-environment -g CLIER_RUN_PLAN_s-1": "CLIER_RUN_PLAN_s-1=" + planPath,
		},
		errByPrefix: map[string]error{
			"list-windows": errors.New("session not found"),
			"kill-session": errors.New("session not found"),
		},
	}
	tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}

	if err := tm.Terminate("s-1"); err != nil {
		t.Fatalf("Terminate (already dead): %v", err)
	}
	if !hasCall(runner.calls, "set-environment -g -u CLIER_RUN_my-team-s-1") {
		t.Error("expected session env cleanup")
	}
	if !hasCall(runner.calls, "set-environment -g -u CLIER_RUN_PLAN_s-1") {
		t.Error("expected plan env cleanup")
	}
}

func TestTmuxTerminal_Attach(t *testing.T) {
	planPath := writePlan(t, "s-1", "my-team-s-1", []apprun.MemberTerminal{{
		TeamMemberID: 1,
		Name:         "leader",
		Window:       1,
	}, {
		TeamMemberID: 2,
		Name:         "worker",
		Window:       2,
	}})

	t.Run("session not found", func(t *testing.T) {
		runner := &fakeRunner{errByPrefix: map[string]error{
			"show-environment": errors.New("not found"),
		}}
		tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}
		err := tm.Attach("unknown", nil)
		if err == nil {
			t.Fatal("expected error for unknown session")
		}
	})

	t.Run("with member selects window", func(t *testing.T) {
		runner := &fakeRunner{output: map[string]string{
			"show-environment -g CLIER_RUN_PLAN_s-1": "CLIER_RUN_PLAN_s-1=" + planPath,
		}}
		tm := &TmuxTerminal{runFn: runner.run, attachFn: func(string) error { return nil }, sleep: func(time.Duration) {}}
		memberID := "2"
		if err := tm.Attach("s-1", &memberID); err != nil {
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

	t.Run("without member skips select-window", func(t *testing.T) {
		runner := &fakeRunner{output: map[string]string{
			"show-environment -g CLIER_RUN_PLAN_s-1": "CLIER_RUN_PLAN_s-1=" + planPath,
		}}
		tm := &TmuxTerminal{runFn: runner.run, attachFn: func(string) error { return nil }, sleep: func(time.Duration) {}}
		if err := tm.Attach("s-1", nil); err != nil {
			t.Fatalf("Attach: %v", err)
		}
		if hasCall(runner.calls, "select-window") {
			t.Error("select-window should not be called without member")
		}
	})
}

func TestHasClaudeMarker(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  bool
	}{
		{"idle title", "✳ Claude Code", true},
		{"working title", "⠋ Claude Code", true},
		{"empty", "", false},
		{"plain shell", "zsh", false},
		{"version number", "2.1.92", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasClaudeMarker(tt.title); got != tt.want {
				t.Errorf("hasClaudeMarker(%q) = %v, want %v", tt.title, got, tt.want)
			}
		})
	}
}

func TestTmuxTerminal_WaitReady_Timeout(t *testing.T) {
	runner := &fakeRunner{output: map[string]string{
		"display-message": "zsh",
	}}
	tm := &TmuxTerminal{runFn: runner.run, sleep: func(time.Duration) {}}

	err := tm.waitReady("sess", "0", 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("unexpected error: %v", err)
	}
}

func writePlan(t *testing.T, runID, session string, members []apprun.MemberTerminal) string {
	t.Helper()
	base := t.TempDir()
	plan := &apprun.RunPlan{
		RunID:   runID,
		Session: session,
		Members: members,
	}
	if err := apprun.SavePlan(base, runID, plan); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	return apprun.PlanPath(base, runID)
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
