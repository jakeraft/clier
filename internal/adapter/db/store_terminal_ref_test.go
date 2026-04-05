package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRefStore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	// Seed a task so FK constraint passes.
	_, err = store.db.ExecContext(context.Background(),
		"INSERT INTO teams (id, name, root_team_member_id, created_at, updated_at) VALUES ('t1','team','root',0,0)")
	if err != nil {
		t.Fatalf("seed team: %v", err)
	}
	_, err = store.db.ExecContext(context.Background(),
		"INSERT INTO tasks (id, team_id, status, plan, created_at) VALUES ('s1','t1','running','[]',0)")
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}

	ctx := context.Background()

	t.Run("SaveAndGet", func(t *testing.T) {
		refs := map[string]string{"session": "team-s1", "window": "0"}
		if err := store.SaveRefs(ctx, "s1", "m1", refs); err != nil {
			t.Fatalf("SaveRefs: %v", err)
		}
		got, err := store.GetRefs(ctx, "s1", "m1")
		if err != nil {
			t.Fatalf("GetRefs: %v", err)
		}
		if got["session"] != "team-s1" || got["window"] != "0" {
			t.Errorf("GetRefs = %v, want session=team-s1 window=0", got)
		}
	})

	t.Run("GetTaskRefs", func(t *testing.T) {
		got, err := store.GetTaskRefs(ctx, "s1")
		if err != nil {
			t.Fatalf("GetTaskRefs: %v", err)
		}
		if got["session"] != "team-s1" {
			t.Errorf("GetTaskRefs session = %q, want team-s1", got["session"])
		}
	})

	t.Run("Upsert", func(t *testing.T) {
		refs := map[string]string{"session": "team-s1", "window": "5"}
		if err := store.SaveRefs(ctx, "s1", "m1", refs); err != nil {
			t.Fatalf("SaveRefs upsert: %v", err)
		}
		got, err := store.GetRefs(ctx, "s1", "m1")
		if err != nil {
			t.Fatalf("GetRefs: %v", err)
		}
		if got["window"] != "5" {
			t.Errorf("window = %q after upsert, want 5", got["window"])
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if err := store.DeleteRefs(ctx, "s1"); err != nil {
			t.Fatalf("DeleteRefs: %v", err)
		}
		_, err := store.GetRefs(ctx, "s1", "m1")
		if err == nil {
			t.Error("expected error after delete, got nil")
		}
	})
}
