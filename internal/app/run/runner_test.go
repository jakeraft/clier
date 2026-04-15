package run

import (
	"errors"
	"os"
	"testing"
)

type stubLauncher struct {
	err error
}

func (l *stubLauncher) Launch(_ *RunPlan) error {
	return l.err
}

func TestRunnerRun_RemovesRunFileWhenLaunchFails(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	runner := NewRunner(&stubLauncher{err: errors.New("launch failed")})

	_, err := runner.Run(base, "run-123", "alpha", []MemberTerminal{{
		MemberID: 1,
		Name:     "leader",
	}})
	if err == nil {
		t.Fatal("expected launch failure")
	}

	if _, statErr := os.Stat(PlanPath(base, "run-123")); !os.IsNotExist(statErr) {
		t.Fatalf("run file should be removed on launch failure, got %v", statErr)
	}
}
