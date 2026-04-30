package runplan

import (
	"errors"
	"os"
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
		TeamName:    "hello-clier",
		Agents: []Agent{
			{
				ID:           "jakeraft.hello-clier",
				Window:       0,
				AbsCwd:       "/tmp/x",
				GitRepoURL:   "https://github.com/jakeraft/hello-clier",
				GitSubpath:   "",
				GitDest:      "jakeraft.hello-clier",
				ProtocolDest: "protocols/jakeraft.hello-clier.md",
				Command:      "claude --setting-sources project",
				Args:         []string{"--append-system-prompt-file", "../protocols/jakeraft.hello-clier.md"},
				AgentType:    "claude",
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
	if len(loaded.Agents) != 1 || loaded.Agents[0].ID != "jakeraft.hello-clier" {
		t.Errorf("Agents: got %+v", loaded.Agents)
	}
	if loaded.Agents[0].Args[1] != "../protocols/jakeraft.hello-clier.md" {
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

// PurgeRunArtifacts wipes each agent's git clone destination + protocol
// file, leaving run.json (so retrospection keeps working).
func TestPurgeRunArtifactsLeavesPlanJSON(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	plan := &Plan{
		RunID:     "rid",
		RunDir:    store.RunDir("rid"),
		StartedAt: time.Now(),
		Agents: []Agent{
			{
				ID:           "ns.team",
				GitDest:      "ns.team",
				ProtocolDest: "protocols/ns.team.md",
			},
		},
	}
	if err := store.Save(plan); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Drop a fake clone tree + protocol file.
	cloneFile := filepath.Join(plan.RunDir, "ns.team", "src", "main.go")
	protoFile := filepath.Join(plan.RunDir, "protocols", "ns.team.md")
	if err := makeFile(cloneFile, "package main"); err != nil {
		t.Fatalf("makeFile clone: %v", err)
	}
	if err := makeFile(protoFile, "# protocol"); err != nil {
		t.Fatalf("makeFile protocol: %v", err)
	}

	if err := store.PurgeRunArtifacts(plan); err != nil {
		t.Fatalf("PurgeRunArtifacts: %v", err)
	}

	if _, err := os.Stat(cloneFile); !os.IsNotExist(err) {
		t.Errorf("clone tree should be gone, stat err: %v", err)
	}
	if _, err := os.Stat(protoFile); !os.IsNotExist(err) {
		t.Errorf("protocol file should be gone, stat err: %v", err)
	}
	if _, err := store.Load("rid"); err != nil {
		t.Errorf("expected plan to still load after purge, got %v", err)
	}
}

func makeFile(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o644)
}
