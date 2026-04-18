package run

import (
	"errors"
	"testing"
)

type stubLauncher struct {
	err error
}

func (l *stubLauncher) Launch(_ *RunPlan) error {
	return l.err
}

type stubRunnerStore struct {
	saved   []*RunPlan
	deleted []string
}

func (s *stubRunnerStore) Save(plan *RunPlan) error {
	s.saved = append(s.saved, plan)
	return nil
}

func (s *stubRunnerStore) Delete(runID string) error {
	s.deleted = append(s.deleted, runID)
	return nil
}

func TestRunnerRun_RemovesRunFileWhenLaunchFails(t *testing.T) {
	t.Parallel()

	store := &stubRunnerStore{}
	runner := NewRunner(&stubLauncher{err: errors.New("launch failed")}, store)

	_, err := runner.Run("/tmp/wc", "run-123", "alpha", []AgentTerminal{{
		Name: "leader",
	}})
	if err == nil {
		t.Fatal("expected launch failure")
	}
	if len(store.deleted) != 1 || store.deleted[0] != "run-123" {
		t.Fatalf("Delete called with %v, want [run-123]", store.deleted)
	}
}
