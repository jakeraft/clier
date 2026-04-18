package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
	storemanifest "github.com/jakeraft/clier/internal/store/manifest"
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

	agentDir := filepath.Join(base, "bob.coder")
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
	oldGenerated := filepath.ToSlash(filepath.Join("bob.coder", ".clier", "team-protocol.md"))

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
	if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(oldTracked))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("tracked file should be removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(oldGenerated))); !errors.Is(err, os.ErrNotExist) {
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

func TestPull_ReportsChangedResources(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	rootProjection := TeamProjection{
		Name:      "solo",
		AgentType: "claude",
		InstructionRef: &ResourceRefProjection{
			Owner:   "org",
			Name:    "prompt",
			Version: 1,
		},
	}
	pre := &Manifest{
		Kind:     string(api.KindTeam),
		Owner:    "org",
		Name:     "solo",
		ClonedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "org",
			Name:          "solo",
			LocalPath:     teamTrackedPath("org", "solo"),
			RemoteVersion: intPtr(1),
			BaseHash:      hashTeamProjectionForTest(t, rootProjection),
			Editable:      true,
		},
		Teams: []StoredTeamState{
			{Owner: "org", Name: "solo", Version: 1, LocalDir: "org.solo", Projection: rootProjection},
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "solo",
				LocalPath:     teamTrackedPath("org", "solo"),
				RemoteVersion: intPtr(1),
				BaseHash:      hashTeamProjectionForTest(t, rootProjection),
				Editable:      true,
			},
			{
				Kind:          string(api.KindInstruction),
				AgentType:     "claude",
				Owner:         "org",
				Name:          "prompt",
				LocalPath:     "org.solo/CLAUDE.md",
				RemoteVersion: intPtr(1),
				Editable:      true,
			},
		},
	}
	if err := storemanifest.Save(fs, base, pre); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	if err := fs.EnsureFile(filepath.Join(base, "org.solo", "CLAUDE.md"), []byte("hello")); err != nil {
		t.Fatalf("EnsureFile: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/orgs/org/teams/solo/resolve" {
			_ = json.NewEncoder(w).Encode(api.ResolveResponse{
				Root: api.ResolvedResource{
					Kind:      string(api.KindTeam),
					OwnerName: "org",
					Name:      "solo",
					Version:   2,
					Snapshot:  []byte(`{"agent_type":"claude","command":"claude","refs":[{"rel_type":"instruction","target_owner":"org","target_name":"prompt","target_version":2}]}`),
				},
				Resources: []api.ResolvedResource{
					{
						Kind:      string(api.KindInstruction),
						OwnerName: "org",
						Name:      "prompt",
						Version:   2,
						Snapshot:  []byte(`{"content":"updated"}`),
					},
				},
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

	if pulled.Status != PullStatusPulled {
		t.Fatalf("Pull status = %q, want pulled", pulled.Status)
	}
	if len(pulled.Resources) != 2 {
		t.Fatalf("Pull resources len = %d, want 2 (%+v)", len(pulled.Resources), pulled.Resources)
	}

	got := map[string]PullResourceChange{}
	for _, change := range pulled.Resources {
		got[change.Name] = change
	}
	if got["solo"].From == nil || *got["solo"].From != 1 || got["solo"].To == nil || *got["solo"].To != 2 {
		t.Fatalf("solo change = %+v, want 1->2", got["solo"])
	}
	if got["prompt"].From == nil || *got["prompt"].From != 1 || got["prompt"].To == nil || *got["prompt"].To != 2 {
		t.Fatalf("prompt change = %+v, want 1->2", got["prompt"])
	}
}

func TestFetch_PreviewsChangesWithoutWriting(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	rootProjection := TeamProjection{
		Name:      "solo",
		AgentType: "claude",
		InstructionRef: &ResourceRefProjection{
			Owner:   "org",
			Name:    "prompt",
			Version: 1,
		},
	}
	pre := &Manifest{
		Kind:     string(api.KindTeam),
		Owner:    "org",
		Name:     "solo",
		ClonedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "org",
			Name:          "solo",
			LocalPath:     teamTrackedPath("org", "solo"),
			RemoteVersion: intPtr(1),
			BaseHash:      hashTeamProjectionForTest(t, rootProjection),
			Editable:      true,
		},
		Teams: []StoredTeamState{
			{Owner: "org", Name: "solo", Version: 1, LocalDir: "org.solo", Projection: rootProjection},
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "solo",
				LocalPath:     teamTrackedPath("org", "solo"),
				RemoteVersion: intPtr(1),
				BaseHash:      hashTeamProjectionForTest(t, rootProjection),
				Editable:      true,
			},
			{
				Kind:          string(api.KindInstruction),
				AgentType:     "claude",
				Owner:         "org",
				Name:          "prompt",
				LocalPath:     "org.solo/CLAUDE.md",
				RemoteVersion: intPtr(1),
				BaseHash:      hashStringForTest("hello"),
				Editable:      true,
			},
		},
	}
	if err := storemanifest.Save(fs, base, pre); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	if err := fs.EnsureFile(filepath.Join(base, "org.solo", "CLAUDE.md"), []byte("hello")); err != nil {
		t.Fatalf("EnsureFile: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/orgs/org/teams/solo/resolve" {
			_ = json.NewEncoder(w).Encode(api.ResolveResponse{
				Root: api.ResolvedResource{
					Kind:      string(api.KindTeam),
					OwnerName: "org",
					Name:      "solo",
					Version:   2,
					Snapshot:  []byte(`{"agent_type":"claude","command":"claude","refs":[{"rel_type":"instruction","target_owner":"org","target_name":"prompt","target_version":2}]}`),
				},
				Resources: []api.ResolvedResource{
					{
						Kind:      string(api.KindInstruction),
						OwnerName: "org",
						Name:      "prompt",
						Version:   2,
						Snapshot:  []byte(`{"content":"updated"}`),
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	svc := NewService(api.NewClient(server.URL, ""), fs, nil)
	result, err := svc.Fetch(base)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if result.Status != FetchStatusUpdatesAvailable {
		t.Fatalf("Fetch status = %q, want updates_available", result.Status)
	}
	if len(result.Resources) != 2 {
		t.Fatalf("Fetch resources len = %d, want 2 (%+v)", len(result.Resources), result.Resources)
	}

	data, err := fs.ReadFile(filepath.Join(base, "org.solo", "CLAUDE.md"))
	if err != nil {
		t.Fatalf("ReadFile after fetch: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("local file content = %q, want unchanged hello", string(data))
	}

	reloaded, err := storemanifest.Load(fs, base)
	if err != nil {
		t.Fatalf("LoadManifest after fetch: %v", err)
	}
	if reloaded.RootResource.RemoteVersion == nil || *reloaded.RootResource.RemoteVersion != 1 {
		t.Fatalf("saved root version = %v, want unchanged 1", reloaded.RootResource.RemoteVersion)
	}
	promptTracked, ok := reloaded.FindTrackedResource("org.solo/CLAUDE.md")
	if !ok || promptTracked.RemoteVersion == nil || *promptTracked.RemoteVersion != 1 {
		t.Fatalf("saved prompt tracked = %+v, want unchanged version 1", promptTracked)
	}
}

func TestPush_ReportsDirectAndCascadeUpdates(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	rootProjection := TeamProjection{
		Name:      "solo",
		AgentType: "claude",
		Command:   "claude",
		InstructionRef: &ResourceRefProjection{
			Owner:   "org",
			Name:    "prompt",
			Version: 1,
		},
	}
	pre := &Manifest{
		Kind:     string(api.KindTeam),
		Owner:    "org",
		Name:     "solo",
		ClonedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "org",
			Name:          "solo",
			LocalPath:     teamTrackedPath("org", "solo"),
			RemoteVersion: intPtr(1),
			BaseHash:      hashTeamProjectionForTest(t, rootProjection),
			Editable:      true,
		},
		Teams: []StoredTeamState{
			{Owner: "org", Name: "solo", Version: 1, LocalDir: "org.solo", Projection: rootProjection},
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "solo",
				LocalPath:     teamTrackedPath("org", "solo"),
				RemoteVersion: intPtr(1),
				BaseHash:      hashTeamProjectionForTest(t, rootProjection),
				Editable:      true,
			},
			{
				Kind:          string(api.KindInstruction),
				AgentType:     "claude",
				Owner:         "org",
				Name:          "prompt",
				LocalPath:     "org.solo/CLAUDE.md",
				RemoteVersion: intPtr(1),
				BaseHash:      hashStringForTest("old prompt"),
				Editable:      true,
			},
		},
	}
	if err := storemanifest.Save(fs, base, pre); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	if err := fs.EnsureFile(filepath.Join(base, "org.solo", "CLAUDE.md"), []byte("updated prompt")); err != nil {
		t.Fatalf("EnsureFile: %v", err)
	}

	promptVersion := 1
	teamVersion := 1

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/orgs/org/resources/prompt":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Kind: string(api.KindInstruction),
				Metadata: api.ResourceMetadata{
					Name:          "prompt",
					OwnerName:     "org",
					LatestVersion: promptVersion,
				},
			})
		case "/api/v1/orgs/org/resources/solo":
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Kind: string(api.KindTeam),
				Metadata: api.ResourceMetadata{
					Name:          "solo",
					OwnerName:     "org",
					LatestVersion: teamVersion,
				},
			})
		case "/api/v1/orgs/org/instructions/prompt":
			var body api.ContentWriteRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode prompt update: %v", err)
			}
			if body.Content != "updated prompt" {
				t.Fatalf("prompt content = %q, want updated prompt", body.Content)
			}
			promptVersion = 2
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Kind: string(api.KindInstruction),
				Metadata: api.ResourceMetadata{
					Name:          "prompt",
					OwnerName:     "org",
					LatestVersion: promptVersion,
				},
			})
		case "/api/v1/orgs/org/teams/solo":
			var body api.TeamWriteRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode team update: %v", err)
			}
			if body.Instruction == nil || body.Instruction.Version != 2 {
				t.Fatalf("team instruction ref = %+v, want version 2", body.Instruction)
			}
			teamVersion = 2
			_ = json.NewEncoder(w).Encode(api.ResourceResponse{
				Kind: string(api.KindTeam),
				Metadata: api.ResourceMetadata{
					Name:          "solo",
					OwnerName:     "org",
					LatestVersion: teamVersion,
				},
			})
		case "/api/v1/orgs/org/teams/solo/resolve":
			_ = json.NewEncoder(w).Encode(api.ResolveResponse{
				Root: api.ResolvedResource{
					Kind:      string(api.KindTeam),
					OwnerName: "org",
					Name:      "solo",
					Version:   teamVersion,
					Snapshot:  []byte(`{"agent_type":"claude","command":"claude","refs":[{"rel_type":"instruction","target_owner":"org","target_name":"prompt","target_version":2}]}`),
				},
				Resources: []api.ResolvedResource{
					{
						Kind:      string(api.KindInstruction),
						OwnerName: "org",
						Name:      "prompt",
						Version:   promptVersion,
						Snapshot:  []byte(`{"content":"updated prompt"}`),
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := NewService(api.NewClient(server.URL, ""), fs, nil)
	result, err := svc.Push(base)
	if err != nil {
		t.Fatalf("Push: %v", err)
	}

	if result.Status != PushStatusPushed {
		t.Fatalf("Push status = %q, want pushed", result.Status)
	}
	if len(result.Pushed) != 2 {
		t.Fatalf("Push pushed len = %d, want 2 (%+v)", len(result.Pushed), result.Pushed)
	}

	got := map[string]PushResourceChange{}
	for _, change := range result.Pushed {
		got[change.Name] = change
	}

	prompt := got["prompt"]
	if prompt.Kind != string(api.KindInstruction) || prompt.Owner != "org" || prompt.Reason != PushReasonLocalEdit {
		t.Fatalf("prompt push = %+v", prompt)
	}
	if prompt.From == nil || *prompt.From != 1 || prompt.To == nil || *prompt.To != 2 {
		t.Fatalf("prompt versions = %+v, want 1->2", prompt)
	}

	solo := got["solo"]
	if solo.Kind != string(api.KindTeam) || solo.Owner != "org" || solo.Reason != PushReasonRefCascade {
		t.Fatalf("solo push = %+v", solo)
	}
	if solo.From == nil || *solo.From != 1 || solo.To == nil || *solo.To != 2 {
		t.Fatalf("solo versions = %+v, want 1->2", solo)
	}

	reloaded, err := storemanifest.Load(fs, base)
	if err != nil {
		t.Fatalf("LoadManifest after push: %v", err)
	}
	root, ok := reloaded.FindTeam("org", "solo")
	if !ok || root.Version != 2 {
		t.Fatalf("reloaded root team = %+v, want version 2", root)
	}
	promptTracked, ok := reloaded.FindTrackedResource("org.solo/CLAUDE.md")
	if !ok || promptTracked.RemoteVersion == nil || *promptTracked.RemoteVersion != 2 {
		t.Fatalf("reloaded prompt tracked = %+v, want version 2", promptTracked)
	}
}

func hashStringForTest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func TestCloneVersion_UsesVersionedResolveEndpoint(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/orgs/org/teams/solo/versions/7/resolve" {
			_ = json.NewEncoder(w).Encode(api.ResolveResponse{
				Root: api.ResolvedResource{
					Kind:      string(api.KindTeam),
					OwnerName: "org",
					Name:      "solo",
					Version:   7,
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
	manifest, err := svc.CloneVersion(base, "org", "solo", 7)
	if err != nil {
		t.Fatalf("CloneVersion: %v", err)
	}

	if manifest.RootResource.RemoteVersion == nil || *manifest.RootResource.RemoteVersion != 7 {
		t.Fatalf("root version = %v, want 7", manifest.RootResource.RemoteVersion)
	}
	reloaded, err := storemanifest.Load(fs, base)
	if err != nil {
		t.Fatalf("LoadManifest after clone: %v", err)
	}
	if reloaded.RootResource.RemoteVersion == nil || *reloaded.RootResource.RemoteVersion != 7 {
		t.Fatalf("saved root version = %v, want 7", reloaded.RootResource.RemoteVersion)
	}
}

func TestPush_NoChangesReturnsEmptyPushedList(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	rootProjection := TeamProjection{Name: "solo", AgentType: "claude"}
	manifest := &Manifest{
		Kind:     string(api.KindTeam),
		Owner:    "org",
		Name:     "solo",
		ClonedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "org",
			Name:          "solo",
			LocalPath:     teamTrackedPath("org", "solo"),
			RemoteVersion: intPtr(1),
			BaseHash:      hashTeamProjectionForTest(t, rootProjection),
			Editable:      true,
		},
		Teams: []StoredTeamState{
			{Owner: "org", Name: "solo", Version: 1, Projection: rootProjection},
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "org",
				Name:          "solo",
				LocalPath:     teamTrackedPath("org", "solo"),
				RemoteVersion: intPtr(1),
				BaseHash:      hashTeamProjectionForTest(t, rootProjection),
				Editable:      true,
			},
		},
	}
	if err := storemanifest.Save(fs, base, manifest); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	svc := NewService(nil, fs, nil)
	result, err := svc.Push(base)
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if result.Status != PushStatusNoChanges {
		t.Fatalf("Push status = %q, want %q", result.Status, PushStatusNoChanges)
	}
	if result.Pushed == nil || len(result.Pushed) != 0 {
		t.Fatalf("Push pushed = %+v, want empty slice", result.Pushed)
	}
}
