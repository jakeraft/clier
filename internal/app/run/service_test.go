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
func (t *stubTerminal) Send(_ *RunPlan, _ string, text string) error {
	t.sent = append(t.sent, text)
	return nil
}

type stubPlanStore struct {
	saved []*RunPlan
}

func (s *stubPlanStore) Save(plan *RunPlan) error {
	s.saved = append(s.saved, plan)
	return nil
}

func strPtr(v string) *string { return &v }

func TestService_Note(t *testing.T) {
	store := &stubPlanStore{}
	svc := New(&stubTerminal{}, store)
	plan := &RunPlan{RunID: "1", Session: "team-1"}

	t.Run("success", func(t *testing.T) {
		if err := svc.Note(plan, strPtr("worker"), "run done"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(plan.Notes) == 0 {
			t.Fatal("expected note to be recorded")
		}
		if len(store.saved) == 0 {
			t.Fatal("expected plan to be saved")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		err := svc.Note(plan, strPtr("worker"), "  ")
		if err == nil {
			t.Fatal("expected error for empty content")
		}
	})
}

func TestService_Stop(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		plan := &RunPlan{RunID: "1", Session: "team-1", Status: StatusRunning}
		term := &stubTerminal{}
		store := &stubPlanStore{}
		svc := New(term, store)
		svc.sleep = func(_ time.Duration) {}

		if err := svc.Stop(plan); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if plan.Status != StatusStopped {
			t.Errorf("status = %q, want %q", plan.Status, StatusStopped)
		}
		if len(store.saved) == 0 {
			t.Fatal("expected plan to be saved after stop")
		}
	})

	t.Run("terminate failure returns error", func(t *testing.T) {
		plan := &RunPlan{RunID: "1", Session: "team-1"}
		term := &stubTerminal{stopErr: errors.New("run plan not found")}
		svc := New(term, &stubPlanStore{})

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
		Agents:  []AgentTerminal{{Name: "worker", Window: 1}},
	}

	t.Run("agent message includes sender name and is recorded", func(t *testing.T) {
		term := &stubTerminal{}
		store := &stubPlanStore{}
		svc := New(term, store)

		if err := svc.Send(plan, strPtr("leader"), strPtr("worker"), "hello"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(term.sent) != 1 {
			t.Fatalf("expected 1 sent, got %d", len(term.sent))
		}
		want := "[Message from leader] hello"
		if term.sent[0] != want {
			t.Errorf("sent = %q, want %q", term.sent[0], want)
		}
		if len(plan.Messages) == 0 {
			t.Fatal("expected message to be recorded")
		}
		if len(store.saved) == 0 {
			t.Fatal("expected plan to be saved after send")
		}
	})

	t.Run("nil sender has no prefix", func(t *testing.T) {
		term := &stubTerminal{}
		svc := New(term, &stubPlanStore{})

		if err := svc.Send(plan, nil, strPtr("worker"), "do this"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if term.sent[0] != "do this" {
			t.Errorf("sent = %q, want %q", term.sent[0], "do this")
		}
	})

	t.Run("delivery failure returns error", func(t *testing.T) {
		term := &failTerminal{}
		svc := New(term, &stubPlanStore{})

		err := svc.Send(plan, strPtr("leader"), strPtr("unknown"), "hello")
		if err == nil {
			t.Fatal("expected error for failed delivery")
		}
	})
}

type failTerminal struct{}

func (t *failTerminal) Terminate(_ *RunPlan) error { return nil }
func (t *failTerminal) Send(_ *RunPlan, _ string, _ string) error {
	return errors.New("surface not found")
}
