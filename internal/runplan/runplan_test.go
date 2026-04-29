package runplan

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSaveLoadRoundtrip(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)

	plan := &Plan{
		RunID:       "test-run",
		SessionName: "clier-test-run",
		RunDir:      store.RunDir("test-run"),
		Namespace:   "jakeraft",
		TeamName:    "clier-qa-claude",
		Mounts: []Mount{
			{
				Name:       "jakeraft.clier-qa-claude",
				GitRepoURL: "https://github.com/jakeraft/clier-qa",
				GitSubpath: "teams/clier-qa-claude",
				LocalDir:   filepath.Join(store.MountsDir("test-run"), "jakeraft.clier-qa-claude"),
			},
		},
		Agents: []Agent{
			{
				ID:        "jakeraft.clier-qa-claude",
				Window:    0,
				Mount:     "jakeraft.clier-qa-claude",
				Cwd:       "jakeraft.clier-qa-claude/teams/clier-qa-claude",
				AbsCwd:    "/tmp/x",
				Command:   "CLIER_AGENT= claude",
				Args:      []string{"--append-system-prompt", "# Team Protocol\n"},
				AgentType: "claude",
			},
		},
		Status:    StatusRunning,
		StartedAt: time.Now().UTC().Truncate(time.Second),
	}

	if err := store.Save(plan); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := store.Load("test-run")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.RunID != plan.RunID {
		t.Errorf("RunID: got %q, want %q", loaded.RunID, plan.RunID)
	}
	if len(loaded.Agents) != 1 || loaded.Agents[0].ID != "jakeraft.clier-qa-claude" {
		t.Errorf("Agents: got %+v", loaded.Agents)
	}
	if loaded.Agents[0].Args[1] != "# Team Protocol\n" {
		t.Errorf("Agents[0].Args[1]: got %q", loaded.Agents[0].Args[1])
	}
}

func TestStoreLoadMissing(t *testing.T) {
	store := NewStore(t.TempDir())
	_, err := store.Load("ghost")
	if !errors.Is(err, ErrRunNotFound) {
		t.Errorf("Load missing: got %v, want ErrRunNotFound", err)
	}
}

func TestStoreListNewestFirst(t *testing.T) {
	store := NewStore(t.TempDir())
	older := time.Now().Add(-1 * time.Hour)
	newer := time.Now()

	if err := store.Save(&Plan{RunID: "older", RunDir: store.RunDir("older"), StartedAt: older}); err != nil {
		t.Fatalf("Save older: %v", err)
	}
	if err := store.Save(&Plan{RunID: "newer", RunDir: store.RunDir("newer"), StartedAt: newer}); err != nil {
		t.Fatalf("Save newer: %v", err)
	}

	plans, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("got %d plans, want 2", len(plans))
	}
	if plans[0].RunID != "newer" || plans[1].RunID != "older" {
		t.Errorf("order: got [%s, %s], want [newer, older]", plans[0].RunID, plans[1].RunID)
	}
}

func TestPurgeMountsLeavesPlanJSON(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	plan := &Plan{RunID: "rid", RunDir: store.RunDir("rid"), StartedAt: time.Now()}
	if err := store.Save(plan); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Drop a fake mount tree.
	mountsDir := store.MountsDir("rid")
	if err := makeFile(filepath.Join(mountsDir, "x", "y"), "data"); err != nil {
		t.Fatalf("makeFile: %v", err)
	}

	if err := store.PurgeMounts("rid"); err != nil {
		t.Fatalf("PurgeMounts: %v", err)
	}

	if _, err := store.Load("rid"); err != nil {
		t.Errorf("expected plan to still load after purge, got %v", err)
	}
}

func makeFile(path, contents string) error {
	if err := mkdirParent(path); err != nil {
		return err
	}
	return writeFile(path, []byte(contents))
}
