package run

import (
	"errors"
	"testing"
	"time"
)

type stubTerminal struct {
	sent    []string
	stopErr error
}

func (t *stubTerminal) Terminate(_ *RunPlan) error {
	return t.stopErr
}
func (t *stubTerminal) Send(_ *RunPlan, _ int64, text string) error {
	t.sent = append(t.sent, text)
	return nil
}

func int64Ptr(v int64) *int64 { return &v }

func TestService_Note(t *testing.T) {
	svc := New(&stubTerminal{})

	t.Run("success", func(t *testing.T) {
		if err := svc.Note(int64Ptr(7), "run done"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		err := svc.Note(int64Ptr(7), "  ")
		if err == nil {
			t.Fatal("expected error for empty content")
		}
	})
}

func TestService_Stop(t *testing.T) {
	plan := &RunPlan{RunID: "1", Session: "team-1"}

	t.Run("success", func(t *testing.T) {
		term := &stubTerminal{}
		svc := New(term)
		svc.sleep = func(_ time.Duration) {}

		if err := svc.Stop(plan); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("terminate failure returns error", func(t *testing.T) {
		term := &stubTerminal{stopErr: errors.New("run plan not found")}
		svc := New(term)

		err := svc.Stop(plan)
		if err == nil {
			t.Fatal("expected error for failed termination")
		}
	})
}

func TestService_Send(t *testing.T) {
	plan := &RunPlan{
		RunID:   "1",
		Session: "team-1",
		Members: []MemberTerminal{{MemberID: 2, Name: "worker", Window: 1}},
	}

	t.Run("agent message includes sender name", func(t *testing.T) {
		term := &stubTerminal{}
		svc := New(term)

		if err := svc.Send(plan, int64Ptr(1), int64Ptr(2), "hello"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(term.sent) != 1 {
			t.Fatalf("expected 1 sent, got %d", len(term.sent))
		}
		want := "[Message from 1] hello"
		if term.sent[0] != want {
			t.Errorf("sent = %q, want %q", term.sent[0], want)
		}
	})

	t.Run("nil sender has no prefix", func(t *testing.T) {
		term := &stubTerminal{}
		svc := New(term)

		if err := svc.Send(plan, nil, int64Ptr(2), "do this"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if term.sent[0] != "do this" {
			t.Errorf("sent = %q, want %q", term.sent[0], "do this")
		}
	})

	t.Run("delivery failure returns error", func(t *testing.T) {
		term := &failTerminal{}
		svc := New(term)

		err := svc.Send(plan, int64Ptr(1), int64Ptr(99), "hello")
		if err == nil {
			t.Fatal("expected error for failed delivery")
		}
	})
}

type failTerminal struct{}

func (t *failTerminal) Terminate(_ *RunPlan) error { return nil }
func (t *failTerminal) Send(_ *RunPlan, _ int64, _ string) error {
	return errors.New("surface not found")
}
