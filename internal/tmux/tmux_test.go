package tmux

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeRun records every invocation and lets tests script return values.
type fakeRun struct {
	calls    [][]string
	returns  func(args ...string) (string, error)
	defaults map[string]string
}

func newFakeRun() *fakeRun {
	return &fakeRun{
		defaults: map[string]string{
			"display-message": "0",
		},
	}
}

func (f *fakeRun) handler(args ...string) (string, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if f.returns != nil {
		return f.returns(args...)
	}
	return f.defaults[args[0]], nil
}

func newFakeReal() (*Real, *fakeRun) {
	f := newFakeRun()
	r := &Real{
		sleep: func(time.Duration) {},
		run:   f.handler,
	}
	return r, f
}

func TestNewSession_storesPhysicalWindowIndex(t *testing.T) {
	r, f := newFakeReal()
	f.returns = func(args ...string) (string, error) {
		if args[0] == "display-message" {
			// Simulate a user with `set -g base-index 1`.
			return "1", nil
		}
		return "", nil
	}

	idx, err := r.NewSession("clier-x", "agent-a", "/scratch/agent-a")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if idx != 1 {
		t.Errorf("idx: got %d, want 1 (caller's base-index)", idx)
	}
	// Two calls: create session, then ask tmux for the actual window index.
	wantArgs := [][]string{
		{"new-session", "-d", "-s", "clier-x", "-n", "agent-a", "-c", "/scratch/agent-a"},
		{"display-message", "-p", "-t", "clier-x", "#{window_index}"},
	}
	if len(f.calls) != len(wantArgs) {
		t.Fatalf("calls: %v", f.calls)
	}
	for i, w := range wantArgs {
		if !equal(f.calls[i], w) {
			t.Errorf("call %d: got %v, want %v", i, f.calls[i], w)
		}
	}
}

func TestNewWindow_returnsCurrentWindowIndex(t *testing.T) {
	r, f := newFakeReal()
	f.returns = func(args ...string) (string, error) {
		if args[0] == "display-message" {
			return "3", nil
		}
		return "", nil
	}

	idx, err := r.NewWindow("clier-x", "agent-b", "/scratch/agent-b")
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	if idx != 3 {
		t.Errorf("idx: got %d, want 3", idx)
	}
	if f.calls[0][0] != "new-window" {
		t.Errorf("first call should be new-window, got %v", f.calls[0])
	}
}

func TestSendLine_literalThenEnter(t *testing.T) {
	r, f := newFakeReal()
	if err := r.SendLine("clier-x", 0, "claude --append-system-prompt 'hi'"); err != nil {
		t.Fatalf("SendLine: %v", err)
	}

	// copy-mode → send-keys -l <text> → send-keys Enter
	if len(f.calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(f.calls), f.calls)
	}
	if f.calls[1][0] != "send-keys" || f.calls[1][1] != "-l" {
		t.Errorf("literal send-keys missing -l flag: %v", f.calls[1])
	}
	if f.calls[1][len(f.calls[1])-1] != "claude --append-system-prompt 'hi'" {
		t.Errorf("payload not last arg: %v", f.calls[1])
	}
	enter := f.calls[2]
	if enter[0] != "send-keys" || enter[len(enter)-1] != "Enter" {
		t.Errorf("expected trailing Enter, got %v", enter)
	}
}

func TestKillSession_swallowsAlreadyGone(t *testing.T) {
	r, f := newFakeReal()
	f.returns = func(args ...string) (string, error) {
		return "", errors.New("can't find session: clier-x")
	}
	if err := r.KillSession("clier-x"); err != nil {
		t.Errorf("KillSession on missing session should be nil, got %v", err)
	}
}

func TestHasSession(t *testing.T) {
	t.Run("alive returns true", func(t *testing.T) {
		r, f := newFakeReal()
		f.returns = func(args ...string) (string, error) { return "", nil }
		alive, err := r.HasSession("clier-x")
		if err != nil || !alive {
			t.Errorf("alive=%v err=%v, want true,nil", alive, err)
		}
	})
	t.Run("missing returns false", func(t *testing.T) {
		r, f := newFakeReal()
		f.returns = func(args ...string) (string, error) {
			return "", errors.New("no such session: clier-x")
		}
		alive, err := r.HasSession("clier-x")
		if err != nil || alive {
			t.Errorf("alive=%v err=%v, want false,nil", alive, err)
		}
	})
}

func TestPaneTitleWrapsSessionGone(t *testing.T) {
	r, f := newFakeReal()
	f.returns = func(args ...string) (string, error) {
		return "", errors.New("can't find session: clier-x")
	}
	_, err := r.PaneTitle("clier-x", 0)
	var gone *ErrSessionGone
	if !errors.As(err, &gone) {
		t.Fatalf("expected ErrSessionGone, got %T %v", err, err)
	}
	if gone.Session != "clier-x" {
		t.Errorf("session: got %q, want clier-x", gone.Session)
	}
}

func TestNewSession_omitsNameAndCwdWhenEmpty(t *testing.T) {
	r, f := newFakeReal()
	f.returns = func(args ...string) (string, error) {
		if args[0] == "display-message" {
			return "0", nil
		}
		return "", nil
	}
	if _, err := r.NewSession("clier-x", "", ""); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	first := f.calls[0]
	got := strings.Join(first, " ")
	if !strings.HasPrefix(got, "new-session -d -s clier-x") {
		t.Errorf("unexpected new-session: %s", got)
	}
	if strings.Contains(got, " -n ") || strings.Contains(got, " -c ") {
		t.Errorf("empty name/cwd must not surface: %s", got)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
