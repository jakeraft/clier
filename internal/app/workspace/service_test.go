package workspace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
)

func TestMaterializeResolvedTeam_TracksNestedTeamsRecursively(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()
	svc := NewService(nil, fs, nil)

	root := &api.ResolvedResource{
		OwnerName: "jakeraft",
		Name:      "platform",
		Version:   1,
		Snapshot:  []byte(`{"agent_type":"manager","children":[{"owner":"alice","name":"lead","version":2}]}`),
	}
	resourceMap := map[string]*api.ResolvedResource{
		"alice/lead": {
			OwnerName: "alice",
			Name:      "lead",
			Version:   2,
			Snapshot:  []byte(`{"agent_type":"manager","children":[{"owner":"bob","name":"coder","version":3}]}`),
		},
		"bob/coder": {
			OwnerName: "bob",
			Name:      "coder",
			Version:   3,
			Snapshot:  []byte(`{"agent_type":"codex","command":"codex"}`),
		},
	}

	manifest, err := svc.materializeResolvedTeam(base, root, resourceMap, nil)
	if err != nil {
		t.Fatalf("materializeResolvedTeam: %v", err)
	}

	wantChild := teamTrackedPath("alice", "lead")
	wantGrandchild := teamTrackedPath("bob", "coder")
	if _, ok := manifest.FindTrackedResource(wantChild); !ok {
		t.Fatalf("tracked resources should include %s", wantChild)
	}
	if _, ok := manifest.FindTrackedResource(wantGrandchild); !ok {
		t.Fatalf("tracked resources should include %s", wantGrandchild)
	}

	agentDir := filepath.Join(base, "bob", "coder")
	if _, err := os.Stat(filepath.Join(agentDir, "AGENTS.md")); err != nil {
		t.Fatalf("stat AGENTS.md: %v", err)
	}
}

func TestRemoveStaleManagedFiles_RemovesDroppedTrackedAndGeneratedFiles(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()
	svc := NewService(nil, fs, nil)

	oldTracked := teamTrackedPath("alice", "lead")
	oldGenerated := filepath.ToSlash(filepath.Join("bob", "coder", ".clier", "bob-coder-team-protocol.md"))

	if err := fs.EnsureFile(filepath.Join(base, filepath.FromSlash(oldTracked)), []byte("tracked")); err != nil {
		t.Fatalf("EnsureFile(tracked): %v", err)
	}
	if err := fs.EnsureFile(filepath.Join(base, filepath.FromSlash(oldGenerated)), []byte("generated")); err != nil {
		t.Fatalf("EnsureFile(generated): %v", err)
	}

	prev := &Manifest{
		TrackedResources: []TrackedResource{{
			LocalPath: oldTracked,
		}},
		GeneratedFiles: []string{oldGenerated},
	}
	next := &Manifest{}

	if err := svc.removeStaleManagedFiles(base, prev, next); err != nil {
		t.Fatalf("removeStaleManagedFiles: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(oldTracked))); !os.IsNotExist(err) {
		t.Fatalf("tracked file should be removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(oldGenerated))); !os.IsNotExist(err) {
		t.Fatalf("generated file should be removed, got %v", err)
	}
}

func TestApplyPushedResourceVersion_UpdatesReferencingTeamState(t *testing.T) {
	t.Parallel()

	rootProjection := TeamProjection{
		Name:      "root",
		AgentType: "claude",
		InstructionRef: &ResourceRefProjection{
			Owner:   "org",
			Name:    "prompt",
			Version: 1,
		},
		Children: []ChildProjection{{
			Owner:        "org",
			Name:         "child",
			ChildVersion: 1,
		}},
	}
	childProjection := TeamProjection{
		Name:      "child",
		AgentType: "codex",
	}
	manifest := &Manifest{
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "org",
			Name:          "root",
			LocalPath:     teamTrackedPath("org", "root"),
			RemoteVersion: intPtr(1),
		},
		Teams: []StoredTeamState{
			{Owner: "org", Name: "root", Version: 1, Projection: rootProjection},
			{Owner: "org", Name: "child", Version: 1, Projection: childProjection},
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "root",
				LocalPath:     teamTrackedPath("org", "root"),
				RemoteVersion: intPtr(1),
			},
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "child",
				LocalPath:     teamTrackedPath("org", "child"),
				RemoteVersion: intPtr(1),
			},
			{
				Kind:          string(api.KindInstruction),
				Owner:         "org",
				Name:          "prompt",
				LocalPath:     "root/CLAUDE.md",
				RemoteVersion: intPtr(1),
			},
		},
	}

	svc := &Service{}
	svc.applyPushedResourceVersion(manifest, TrackedResource{
		Kind:      string(api.KindInstruction),
		Owner:     "org",
		Name:      "prompt",
		LocalPath: "root/CLAUDE.md",
	}, 2)

	root, ok := manifest.FindTeam("org", "root")
	if !ok {
		t.Fatal("root team not found")
	}
	if root.Projection.InstructionRef == nil || root.Projection.InstructionRef.Version != 2 {
		t.Fatalf("instruction ref version = %+v, want 2", root.Projection.InstructionRef)
	}
	promptTracked, ok := manifest.FindTrackedResource("root/CLAUDE.md")
	if !ok || promptTracked.RemoteVersion == nil || *promptTracked.RemoteVersion != 2 {
		t.Fatalf("tracked instruction version = %+v, want 2", promptTracked)
	}

	svc.applyPushedResourceVersion(manifest, TrackedResource{
		Kind:      string(api.KindTeam),
		Owner:     "org",
		Name:      "child",
		LocalPath: teamTrackedPath("org", "child"),
	}, 3)

	child, ok := manifest.FindTeam("org", "child")
	if !ok {
		t.Fatal("child team not found")
	}
	if child.Version != 3 {
		t.Fatalf("child team version = %d, want 3", child.Version)
	}
	root, _ = manifest.FindTeam("org", "root")
	if len(root.Projection.Children) != 1 || root.Projection.Children[0].ChildVersion != 3 {
		t.Fatalf("root child version = %+v, want 3", root.Projection.Children)
	}
}

func TestPull_PreservesFirstRunAt(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	cloned := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	firstRun := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	pre := &Manifest{
		Kind:       string(api.KindTeam),
		Owner:      "org",
		Name:       "solo",
		ClonedAt:   cloned,
		FirstRunAt: &firstRun,
		RootResource: TrackedResource{
			Kind:      string(api.KindTeam),
			Owner:     "org",
			Name:      "solo",
			LocalPath: teamTrackedPath("org", "solo"),
			Editable:  true,
		},
	}
	if err := SaveManifest(fs, base, pre); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/orgs/org/teams/solo/resolve" {
			_ = json.NewEncoder(w).Encode(api.ResolveResponse{
				Root: api.ResolvedResource{
					Kind:      string(api.KindTeam),
					OwnerName: "org",
					Name:      "solo",
					Version:   1,
					Snapshot:  []byte(`{"agent_type":"claude","command":"claude"}`),
				},
				Resources: []api.ResolvedResource{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	svc := NewService(api.NewClient(server.URL, ""), fs, nil)
	pulled, err := svc.Pull(base, true)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}

	if pulled.FirstRunAt == nil {
		t.Fatal("Pull dropped FirstRunAt from returned manifest")
	}
	if !pulled.FirstRunAt.Equal(firstRun) {
		t.Fatalf("returned FirstRunAt = %v, want %v", pulled.FirstRunAt, firstRun)
	}
	if !pulled.ClonedAt.Equal(cloned) {
		t.Fatalf("returned ClonedAt = %v, want %v", pulled.ClonedAt, cloned)
	}

	reloaded, err := LoadManifest(fs, base)
	if err != nil {
		t.Fatalf("LoadManifest after pull: %v", err)
	}
	if reloaded.FirstRunAt == nil || !reloaded.FirstRunAt.Equal(firstRun) {
		t.Fatalf("persisted FirstRunAt = %v, want %v", reloaded.FirstRunAt, firstRun)
	}
}
