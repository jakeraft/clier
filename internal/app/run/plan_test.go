package run

import "testing"

func TestPlan_SaveLoadAndMutate(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	plan := NewPlan("run-123", "team-run", nil)
	if err := plan.AddMessage(int64Ptr(1), int64Ptr(2), "hello"); err != nil {
		t.Fatalf("add message: %v", err)
	}
	if err := plan.AddNote(int64Ptr(2), "working"); err != nil {
		t.Fatalf("add note: %v", err)
	}
	plan.MarkStopped()

	if err := SavePlan(base, plan.RunID, plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	loaded, err := LoadPlan(base, plan.RunID)
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
