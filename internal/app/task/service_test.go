package task

import (
	"context"
	"errors"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

type stubStore struct {
	task    *domain.Task
	team    *domain.Team
	updates []*domain.Update
	msgs    []*domain.Message
}

func (s *stubStore) CreateTask(_ context.Context, t *domain.Task) error { return nil }
func (s *stubStore) GetTask(_ context.Context, id string) (domain.Task, error) {
	if s.task != nil && s.task.ID == id {
		return *s.task, nil
	}
	return domain.Task{}, errors.New("task not found")
}
func (s *stubStore) UpdateTaskStatus(_ context.Context, _ *domain.Task) error { return nil }
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
func (s *stubStore) CreateUpdate(_ context.Context, u *domain.Update) error {
	s.updates = append(s.updates, u)
	return nil
}
func (s *stubStore) GetMember(_ context.Context, _ string) (domain.Member, error) {
	return domain.Member{}, errors.New("not implemented")
}
func (s *stubStore) GetCliProfile(_ context.Context, _ string) (resource.CliProfile, error) {
	return resource.CliProfile{}, errors.New("not implemented")
}
func (s *stubStore) GetSystemPrompt(_ context.Context, _ string) (resource.SystemPrompt, error) {
	return resource.SystemPrompt{}, errors.New("not implemented")
}
func (s *stubStore) GetEnv(_ context.Context, _ string) (resource.Env, error) {
	return resource.Env{}, errors.New("not implemented")
}
func (s *stubStore) GetGitRepo(_ context.Context, _ string) (resource.GitRepo, error) {
	return resource.GitRepo{}, errors.New("not implemented")
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

func TestService_Update(t *testing.T) {
	tk := &domain.Task{ID: "s-1", TeamID: "t-1", Status: domain.TaskRunning}
	store := &stubStore{task: tk}
	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "", "")

	t.Run("success", func(t *testing.T) {
		store.updates = nil
		if err := svc.Update(context.Background(), "s-1", "member-1", "task done"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(store.updates) != 1 {
			t.Fatalf("expected 1 update, got %d", len(store.updates))
		}
		if store.updates[0].Content != "task done" {
			t.Errorf("Content = %q, want %q", store.updates[0].Content, "task done")
		}
		if store.updates[0].TeamMemberID != "member-1" {
			t.Errorf("TeamMemberID = %q, want %q", store.updates[0].TeamMemberID, "member-1")
		}
	})

	t.Run("task not found", func(t *testing.T) {
		err := svc.Update(context.Background(), "unknown", "member-1", "hello")
		if err == nil {
			t.Fatal("expected error for unknown task")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		err := svc.Update(context.Background(), "s-1", "member-1", "  ")
		if err == nil {
			t.Fatal("expected error for empty content")
		}
	})
}

func TestService_Send(t *testing.T) {
	tk := &domain.Task{
		ID:     "s-1",
		TeamID: "t-1",
		Status: domain.TaskRunning,
		Plan: []domain.MemberPlan{
			{TeamMemberID: "m-1", MemberName: "alice"},
			{TeamMemberID: "m-2", MemberName: "bob"},
		},
	}
	team := &domain.Team{ID: "t-1"}

	t.Run("agent message includes sender name", func(t *testing.T) {
		store := &stubStore{task: tk, team: team}
		term := &stubTerminal{}
		svc := New(store, term, &stubWorkspace{}, "", "")

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
		store := &stubStore{task: tk, team: team}
		term := &stubTerminal{}
		svc := New(store, term, &stubWorkspace{}, "", "")

		if err := svc.Send(context.Background(), "s-1", "", "m-2", "do this"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if term.sent[0] != "do this" {
			t.Errorf("sent = %q, want %q", term.sent[0], "do this")
		}
	})

	t.Run("delivery failure prevents save", func(t *testing.T) {
		store := &stubStore{task: tk, team: team}
		term := &failTerminal{}
		svc := New(store, term, &stubWorkspace{}, "", "")

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
