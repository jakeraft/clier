package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
)

func TestSaveManifest(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	resourceVersion := 3
	meta := &Manifest{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "dev-squad",
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "jakeraft",
			Name:          "dev-squad",
			LocalPath:     ".clier/team.json",
			RemoteVersion: &resourceVersion,
			BaseHash:      "abc123",
			Editable:      true,
		},
		TrackedResources: []TrackedResource{{
			Kind:          string(api.KindSkill),
			Owner:         "jakeraft",
			Name:          "reviewer",
			LocalPath:     "lead/.claude/skills/reviewer/SKILL.md",
			RemoteVersion: &resourceVersion,
			BaseHash:      "def456",
			Editable:      true,
		}},
	}

	if err := SaveManifest(filesystem.New(), base, meta); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	path := filepath.Join(base, ".clier", ManifestFile)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat manifest file: %v", err)
	}

	loaded, err := LoadManifest(filesystem.New(), base)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if loaded.Kind != meta.Kind || loaded.Owner != meta.Owner || loaded.Name != meta.Name {
		t.Fatalf("loaded manifest mismatch: %#v", loaded)
	}
	if loaded.RootResource.LocalPath != meta.RootResource.LocalPath {
		t.Fatalf("loaded root resource mismatch: %#v", loaded.RootResource)
	}
	if len(loaded.TrackedResources) != 1 {
		t.Fatalf("expected 1 tracked resource, got %d", len(loaded.TrackedResources))
	}
	if loaded.TrackedResources[0].LocalPath != meta.TrackedResources[0].LocalPath {
		t.Fatalf("loaded tracked resource local path mismatch: %#v", loaded.TrackedResources[0])
	}
}

func TestManifest_MemberCloneUsesTeamRuntime(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	resourceVersion := 1
	meta := &Manifest{
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "reviewer",
		RootResource: TrackedResource{
			Kind:      string(api.KindMember),
			Owner:     "jakeraft",
			Name:      "reviewer",
			LocalPath: TeamMemberProjectionLocalPath("reviewer"),
			Editable:  true,
		},
		TrackedResources: []TrackedResource{{
			Kind:          string(api.KindMember),
			Owner:         "jakeraft",
			Name:          "reviewer",
			LocalPath:     TeamMemberProjectionLocalPath("reviewer"),
			RemoteVersion: &resourceVersion,
			Editable:      true,
		}},
		Runtime: &RuntimeMetadata{
			Team: &TeamRuntimeMetadata{
				ID:   0,
				Name: "reviewer",
				Members: []TeamMemberRuntimeMetadata{{
					MemberID:  42,
					Name:      "reviewer",
					AgentType: "claude",
					Command:   "claude",
				}},
			},
		},
	}

	if err := SaveManifest(filesystem.New(), base, meta); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	loaded, err := LoadManifest(filesystem.New(), base)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if loaded.Kind != string(api.KindMember) {
		t.Fatalf("Kind = %q, want %q", loaded.Kind, string(api.KindMember))
	}
	if loaded.Runtime == nil || loaded.Runtime.Team == nil {
		t.Fatal("expected Team runtime metadata")
	}
	if len(loaded.Runtime.Team.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(loaded.Runtime.Team.Members))
	}
	if loaded.Runtime.Team.Members[0].Name != "reviewer" {
		t.Fatalf("member name = %q, want %q", loaded.Runtime.Team.Members[0].Name, "reviewer")
	}
	if loaded.Runtime.Team.ID != 0 {
		t.Fatalf("team ID = %d, want 0 for member clone", loaded.Runtime.Team.ID)
	}
}

func TestLoadManifest_RequiresManifestPath(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if _, err := LoadManifest(filesystem.New(), base); err == nil {
		t.Fatalf("expected manifest lookup to fail without manifest.json")
	}
}
