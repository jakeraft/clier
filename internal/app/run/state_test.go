package run

import "testing"

func TestState_SaveLoadAndMutate(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	state := NewPlan("run-123", "team-run", nil)
	if err := state.AddMessage(int64Ptr(1), int64Ptr(2), "hello"); err != nil {
		t.Fatalf("add message: %v", err)
	}
	if err := state.AddNote(int64Ptr(2), "working"); err != nil {
		t.Fatalf("add note: %v", err)
	}
	state.MarkStopped()

	if err := SaveState(base, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	loaded, err := LoadState(base, state.RunID)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if loaded.RunID != state.RunID {
		t.Fatalf("RunID = %q, want %q", loaded.RunID, state.RunID)
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
