package run

import (
	"context"
	"errors"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

type stubStore struct {
	run   *domain.Run
	team  *domain.Team
	notes []*domain.Note
	msgs  []*domain.Message
}

func (s *stubStore) CreateRun(_ context.Context, r *domain.Run) error { return nil }
func (s *stubStore) GetRun(_ context.Context, id string) (domain.Run, error) {
	if s.run != nil && s.run.ID == id {
		return *s.run, nil
	}
	return domain.Run{}, errors.New("run not found")
}
func (s *stubStore) UpdateRunStatus(_ context.Context, _ *domain.Run) error { return nil }
func (s *stubStore) GetTeam(_ context.Context, id string) (domain.Team, error) {
	if s.team != nil && s.team.ID == id {
		return *s.team, nil
	}
	return domain.Team{}, errors.New("team not found")
}
func (s *stubStore) CreateMessage(_ context.Context, msg *domain.Message) error {
	s.msgs = append(s.msgs, msg)
	return nil
}
func (s *stubStore) CreateNote(_ context.Context, n *domain.Note) error {
	s.notes = append(s.notes, n)
	return nil
}
func (s *stubStore) GetMember(_ context.Context, _ string) (domain.Member, error) {
	return domain.Member{}, errors.New("not implemented")
}
func (s *stubStore) GetClaudeMd(_ context.Context, _ string) (resource.ClaudeMd, error) {
	return resource.ClaudeMd{}, errors.New("not implemented")
}
func (s *stubStore) GetSkill(_ context.Context, _ string) (resource.Skill, error) {
	return resource.Skill{}, errors.New("not implemented")
}
func (s *stubStore) GetClaudeSettings(_ context.Context, _ string) (resource.ClaudeSettings, error) {
	return resource.ClaudeSettings{}, errors.New("not implemented")
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

type stubWorkspace struct{}

func (w *stubWorkspace) Prepare(_ context.Context, _ []domain.MemberPlan) error { return nil }
func (w *stubWorkspace) Cleanup(_ string) error                                 { return nil }

func TestService_Note(t *testing.T) {
	r := &domain.Run{ID: "s-1", TeamID: "t-1", Status: domain.RunRunning}
	store := &stubStore{run: r}
	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "", "", nil)

	t.Run("success", func(t *testing.T) {
		store.notes = nil
		if err := svc.Note(context.Background(), "s-1", "member-1", "run done"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(store.notes) != 1 {
			t.Fatalf("expected 1 note, got %d", len(store.notes))
		}
		if store.notes[0].Content != "run done" {
			t.Errorf("Content = %q, want %q", store.notes[0].Content, "run done")
		}
		if store.notes[0].TeamMemberID != "member-1" {
			t.Errorf("TeamMemberID = %q, want %q", store.notes[0].TeamMemberID, "member-1")
		}
	})

	t.Run("run not found", func(t *testing.T) {
		err := svc.Note(context.Background(), "unknown", "member-1", "hello")
		if err == nil {
			t.Fatal("expected error for unknown run")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		err := svc.Note(context.Background(), "s-1", "member-1", "  ")
		if err == nil {
			t.Fatal("expected error for empty content")
		}
	})
}

func TestService_Send(t *testing.T) {
	r := &domain.Run{
		ID:     "s-1",
		TeamID: "t-1",
		Status: domain.RunRunning,
		Plan: []domain.MemberPlan{
			{TeamMemberID: "m-1", MemberName: "alice"},
			{TeamMemberID: "m-2", MemberName: "bob"},
		},
	}
	team := &domain.Team{ID: "t-1"}

	t.Run("agent message includes sender name", func(t *testing.T) {
		store := &stubStore{run: r, team: team}
		term := &stubTerminal{}
		svc := New(store, term, &stubWorkspace{}, "", "", nil)

		if err := svc.Send(context.Background(), "s-1", "m-1", "m-2", "hello"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(term.sent) != 1 {
			t.Fatalf("expected 1 sent, got %d", len(term.sent))
		}
		want := "[Message from alice] hello"
		if term.sent[0] != want {
			t.Errorf("sent = %q, want %q", term.sent[0], want)
		}
		if len(store.msgs) != 1 {
			t.Fatalf("expected 1 message saved, got %d", len(store.msgs))
		}
	})

	t.Run("empty sender has no prefix", func(t *testing.T) {
		store := &stubStore{run: r, team: team}
		term := &stubTerminal{}
		svc := New(store, term, &stubWorkspace{}, "", "", nil)

		if err := svc.Send(context.Background(), "s-1", "", "m-2", "do this"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if term.sent[0] != "do this" {
			t.Errorf("sent = %q, want %q", term.sent[0], "do this")
		}
	})

	t.Run("delivery failure prevents save", func(t *testing.T) {
		store := &stubStore{run: r, team: team}
		term := &failTerminal{}
		svc := New(store, term, &stubWorkspace{}, "", "", nil)

		err := svc.Send(context.Background(), "s-1", "m-1", "bad-member", "hello")
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
