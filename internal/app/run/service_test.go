package run

import (
	"context"
	"errors"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

type stubStore struct {
	run   *domain.Run
	notes []*domain.Note
	msgs  []*domain.Message
}

func (s *stubStore) CreateRun(_ context.Context, r *domain.Run) error { return nil }
func (s *stubStore) GetRun(_ context.Context, id int64) (domain.Run, error) {
	if s.run != nil && s.run.ID == id {
		return *s.run, nil
	}
	return domain.Run{}, errors.New("run not found")
}
func (s *stubStore) UpdateRunStatus(_ context.Context, _ *domain.Run) error { return nil }
func (s *stubStore) CreateMessage(_ context.Context, msg *domain.Message) error {
	s.msgs = append(s.msgs, msg)
	return nil
}
func (s *stubStore) CreateNote(_ context.Context, n *domain.Note) error {
	s.notes = append(s.notes, n)
	return nil
}

type stubTerminal struct {
	sent []string
}

func (t *stubTerminal) Launch(_, _ string, _ []domain.MemberPlan) error { return nil }
func (t *stubTerminal) Terminate(_ string) error                        { return nil }
func (t *stubTerminal) Send(_, _, text string) error {
	t.sent = append(t.sent, text)
	return nil
}
func (t *stubTerminal) Attach(_ string, _ *string) error { return nil }

func int64Ptr(v int64) *int64 { return &v }

func TestService_Note(t *testing.T) {
	r := &domain.Run{ID: 1, TeamID: int64Ptr(1), Status: domain.RunRunning}
	store := &stubStore{run: r}
	svc := New(store, &stubTerminal{})

	t.Run("success", func(t *testing.T) {
		store.notes = nil
		if err := svc.Note(context.Background(), 1, int64Ptr(7), "run done"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(store.notes) != 1 {
			t.Fatalf("expected 1 note, got %d", len(store.notes))
		}
		if store.notes[0].Content != "run done" {
			t.Errorf("Content = %q, want %q", store.notes[0].Content, "run done")
		}
		if store.notes[0].TeamMemberID == nil || *store.notes[0].TeamMemberID != 7 {
			t.Errorf("TeamMemberID = %v, want 7", store.notes[0].TeamMemberID)
		}
	})

	t.Run("run not found", func(t *testing.T) {
		err := svc.Note(context.Background(), 9999, int64Ptr(7), "hello")
		if err == nil {
			t.Fatal("expected error for unknown run")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		err := svc.Note(context.Background(), 1, int64Ptr(7), "  ")
		if err == nil {
			t.Fatal("expected error for empty content")
		}
	})
}

func TestService_Send(t *testing.T) {
	r := &domain.Run{
		ID:     1,
		TeamID: int64Ptr(1),
		Status: domain.RunRunning,
	}

	t.Run("agent message includes sender name", func(t *testing.T) {
		store := &stubStore{run: r}
		term := &stubTerminal{}
		svc := New(store, term)

		if err := svc.Send(context.Background(), 1, int64Ptr(1), int64Ptr(2), "hello"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(term.sent) != 1 {
			t.Fatalf("expected 1 sent, got %d", len(term.sent))
		}
		want := "[Message from 1] hello"
		if term.sent[0] != want {
			t.Errorf("sent = %q, want %q", term.sent[0], want)
		}
		if len(store.msgs) != 1 {
			t.Fatalf("expected 1 message saved, got %d", len(store.msgs))
		}
	})

	t.Run("nil sender has no prefix", func(t *testing.T) {
		store := &stubStore{run: r}
		term := &stubTerminal{}
		svc := New(store, term)

		if err := svc.Send(context.Background(), 1, nil, int64Ptr(2), "do this"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if term.sent[0] != "do this" {
			t.Errorf("sent = %q, want %q", term.sent[0], "do this")
		}
	})

	t.Run("delivery failure prevents save", func(t *testing.T) {
		store := &stubStore{run: r}
		term := &failTerminal{}
		svc := New(store, term)

		err := svc.Send(context.Background(), 1, int64Ptr(1), int64Ptr(99), "hello")
		if err == nil {
			t.Fatal("expected error for failed delivery")
		}
		if len(store.msgs) != 0 {
			t.Errorf("expected 0 messages saved after delivery failure, got %d", len(store.msgs))
		}
	})
}

type failTerminal struct{}

func (t *failTerminal) Launch(_, _ string, _ []domain.MemberPlan) error { return nil }
func (t *failTerminal) Terminate(_ string) error                        { return nil }
func (t *failTerminal) Send(_, _, _ string) error                       { return errors.New("surface not found") }
func (t *failTerminal) Attach(_ string, _ *string) error               { return nil }
