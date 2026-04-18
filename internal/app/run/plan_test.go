package run

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	storerunplan "github.com/jakeraft/clier/internal/store/runplan"
)

func TestPlan_SaveLoadAndMutate(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	plan := NewPlan("run-123", "team-run", "/tmp/wc", nil)
	if err := plan.AddMessage(strPtr("leader"), strPtr("worker"), "hello"); err != nil {
		t.Fatalf("add message: %v", err)
	}
	if err := plan.AddNote(strPtr("worker"), "working"); err != nil {
		t.Fatalf("add note: %v", err)
	}
	plan.MarkStopped()

	if err := storerunplan.Save(base, plan.RunID, plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	loaded, err := storerunplan.Load(base, plan.RunID)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if loaded.RunID != plan.RunID {
		t.Fatalf("RunID = %q, want %q", loaded.RunID, plan.RunID)
	}
	if loaded.Status != StatusStopped {
		t.Fatalf("Status = %q, want %q", loaded.Status, StatusStopped)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("Messages = %d, want 1", len(loaded.Messages))
	}
	if len(loaded.Notes) != 1 {
		t.Fatalf("Notes = %d, want 1", len(loaded.Notes))
	}
	if loaded.StoppedAt == nil {
		t.Fatal("StoppedAt = nil, want timestamp")
	}
}

func TestListPlans_FailsOnCorruptPlan(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "broken.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write corrupt plan: %v", err)
	}

	_, err := storerunplan.List(base)
	if err == nil {
		t.Fatal("expected corrupt plan to fail scan")
	}
	if !strings.Contains(err.Error(), "broken.json") {
		t.Fatalf("error should mention broken file, got %v", err)
	}
}
