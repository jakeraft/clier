package terminal

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

// fakeRunner captures tmux commands for verification.
type fakeRunner struct {
	calls  []string
	output map[string]string // prefix -> output
	err    error
}

func (f *fakeRunner) run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
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

// fakeRefStore is an in-memory RefStore for testing.
type fakeRefStore struct {
	refs map[string]map[string]map[string]string // sessionID -> memberID -> refs
}

func newFakeRefStore() *fakeRefStore {
	return &fakeRefStore{refs: make(map[string]map[string]map[string]string)}
}

func (f *fakeRefStore) SaveRefs(_ context.Context, sessionID, memberID string, refs map[string]string) error {
	if f.refs[sessionID] == nil {
		f.refs[sessionID] = make(map[string]map[string]string)
	}
	f.refs[sessionID][memberID] = refs
	return nil
}

func (f *fakeRefStore) GetRefs(_ context.Context, sessionID, memberID string) (map[string]string, error) {
	if m, ok := f.refs[sessionID]; ok {
		if r, ok := m[memberID]; ok {
			return r, nil
		}
	}
	return nil, errors.New("not found")
}

func (f *fakeRefStore) GetSessionRefs(_ context.Context, sessionID string) (map[string]string, error) {
	if m, ok := f.refs[sessionID]; ok {
		for _, r := range m {
			return r, nil
		}
	}
	return nil, errors.New("not found")
}

func (f *fakeRefStore) DeleteRefs(_ context.Context, sessionID string) error {
	delete(f.refs, sessionID)
	return nil
}

func TestTmuxTerminal_Launch(t *testing.T) {
	runner := &fakeRunner{output: map[string]string{
		"list-windows": "0",
	}}
	store := newFakeRefStore()
	tm := &TmuxTerminal{refs: store, runFn: runner.run}

	members := []domain.MemberPlan{
		{TeamMemberID: "m-1", MemberName: "leader", Terminal: domain.TerminalPlan{Command: "echo hello"}},
		{TeamMemberID: "m-2", MemberName: "worker", Terminal: domain.TerminalPlan{}},
	}

	if err := tm.Launch("s-1", "my-team", members); err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Verify tmux session created
	if !hasCall(runner.calls, "new-session") {
		t.Error("expected new-session call")
	}
	// Verify base-index set to 0
	if !hasCall(runner.calls, "set-option") {
		t.Error("expected set-option call for base-index")
	}
	// Verify second window created for second member
	if !hasCall(runner.calls, "new-window") {
		t.Error("expected new-window call for second member")
	}
	// Verify rename-window called for both members
	renameCount := countCalls(runner.calls, "rename-window")
	if renameCount != 2 {
		t.Errorf("expected 2 rename-window calls, got %d", renameCount)
	}
	// Verify command sent to first member
	if !hasCall(runner.calls, "send-keys") {
		t.Error("expected send-keys call for member command")
	}
	// Verify refs saved
	refs, err := store.GetRefs(context.Background(), "s-1", "m-1")
	if err != nil {
		t.Fatalf("GetRefs m-1: %v", err)
	}
	if refs["session"] != "clier-s-1" {
		t.Errorf("session ref = %q, want clier-s-1", refs["session"])
	}
	if refs["window"] != "0" {
		t.Errorf("window ref = %q, want 0", refs["window"])
	}
}

func TestTmuxTerminal_Send(t *testing.T) {
	runner := &fakeRunner{}
	store := newFakeRefStore()
	_ = store.SaveRefs(context.Background(), "s-1", "m-1", map[string]string{
		"session": "clier-s-1", "window": "0",
	})
	tm := &TmuxTerminal{refs: store, runFn: runner.run}

	if err := tm.Send("s-1", "m-1", "do the work"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	call := runner.calls[0]
	if !strings.Contains(call, "send-keys") || !strings.Contains(call, "do the work") {
		t.Errorf("unexpected call: %s", call)
	}
}

func TestTmuxTerminal_Terminate(t *testing.T) {
	runner := &fakeRunner{output: map[string]string{
		"list-windows": "0\n1\n2",
	}}
	store := newFakeRefStore()
	_ = store.SaveRefs(context.Background(), "s-1", "m-1", map[string]string{
		"session": "clier-s-1", "window": "0",
	})
	tm := &TmuxTerminal{refs: store, runFn: runner.run}

	if err := tm.Terminate("s-1"); err != nil {
		t.Fatalf("Terminate: %v", err)
	}
	// Verify /exit sent to each window
	exitCount := 0
	for _, c := range runner.calls {
		if strings.Contains(c, "/exit") {
			exitCount++
		}
	}
	if exitCount != 3 {
		t.Errorf("expected 3 /exit sends, got %d", exitCount)
	}
	// Verify kill-session called
	if !hasCall(runner.calls, "kill-session") {
		t.Error("expected kill-session call")
	}
	// Verify refs deleted
	_, err := store.GetRefs(context.Background(), "s-1", "m-1")
	if err == nil {
		t.Error("expected refs to be deleted")
	}
}

func TestTmuxTerminal_Terminate_AlreadyDead(t *testing.T) {
	runner := &fakeRunner{err: errors.New("session not found")}
	store := newFakeRefStore()
	_ = store.SaveRefs(context.Background(), "s-1", "m-1", map[string]string{
		"session": "clier-s-1", "window": "0",
	})
	tm := &TmuxTerminal{refs: store, runFn: runner.run}

	// Should not error — idempotent
	if err := tm.Terminate("s-1"); err != nil {
		t.Fatalf("Terminate (already dead): %v", err)
	}
	// Refs should still be deleted
	_, err := store.GetRefs(context.Background(), "s-1", "m-1")
	if err == nil {
		t.Error("expected refs to be deleted even when tmux session is dead")
	}
}

func TestTmuxTerminal_Attach(t *testing.T) {
	store := newFakeRefStore()
	_ = store.SaveRefs(context.Background(), "s-1", "m-1", map[string]string{
		"session": "clier-s-1", "window": "1",
	})
	_ = store.SaveRefs(context.Background(), "s-1", "m-2", map[string]string{
		"session": "clier-s-1", "window": "2",
	})

	t.Run("session not found", func(t *testing.T) {
		runner := &fakeRunner{}
		tm := &TmuxTerminal{refs: store, runFn: runner.run}
		err := tm.Attach("unknown", nil)
		if err == nil {
			t.Fatal("expected error for unknown session")
		}
	})

	t.Run("with member selects window", func(t *testing.T) {
		runner := &fakeRunner{}
		// Override attachSession to avoid actual exec
		tm := &TmuxTerminal{refs: store, runFn: runner.run, attachFn: func(string) error { return nil }}
		memberID := "m-2"
		if err := tm.Attach("s-1", &memberID); err != nil {
			t.Fatalf("Attach: %v", err)
		}
		if !hasCall(runner.calls, "select-window") {
			t.Error("expected select-window call")
		}
		// Verify correct window targeted
		for _, c := range runner.calls {
			if strings.Contains(c, "select-window") && !strings.Contains(c, ":2") {
				t.Errorf("select-window should target window 2, got: %s", c)
			}
		}
	})

	t.Run("without member skips select-window", func(t *testing.T) {
		runner := &fakeRunner{}
		tm := &TmuxTerminal{refs: store, runFn: runner.run, attachFn: func(string) error { return nil }}
		if err := tm.Attach("s-1", nil); err != nil {
			t.Fatalf("Attach: %v", err)
		}
		if hasCall(runner.calls, "select-window") {
			t.Error("select-window should not be called without member")
		}
	})
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
