package workspace

import (
	"os"
	"path/filepath"
	"strings"
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
				Name: "reviewer",
				Members: []TeamMemberRuntimeMetadata{{
					Name:      "reviewer",
					Owner:     "jakeraft",
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
	if loaded.Runtime.Team.Members[0].Owner != "jakeraft" {
		t.Fatalf("member owner = %q, want %q", loaded.Runtime.Team.Members[0].Owner, "jakeraft")
	}
}

func TestLoadManifest_RejectsOutdatedFormat(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	// Write a manifest with format 0 (simulating a legacy clone).
	data := []byte(`{"format":0,"kind":"member","owner":"jakeraft","name":"reviewer"}`)
	if err := fs.EnsureFile(ManifestPath(base), data); err != nil {
		t.Fatalf("write legacy manifest: %v", err)
	}

	_, err := LoadManifest(fs, base)
	if err == nil {
		t.Fatal("expected error for outdated manifest format")
	}
	if !strings.Contains(err.Error(), "outdated") {
		t.Fatalf("error should mention outdated: %v", err)
	}
}

func TestLoadManifest_RejectsNewerFormat(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	// Write a manifest with a future format version.
	data := []byte(`{"format":999,"kind":"member","owner":"jakeraft","name":"reviewer"}`)
	if err := fs.EnsureFile(ManifestPath(base), data); err != nil {
		t.Fatalf("write future manifest: %v", err)
	}

	_, err := LoadManifest(fs, base)
	if err == nil {
		t.Fatal("expected error for newer manifest format")
	}
	if !strings.Contains(err.Error(), "upgrade") {
		t.Fatalf("error should suggest upgrade: %v", err)
	}
}

func TestLoadManifest_RequiresManifestPath(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if _, err := LoadManifest(filesystem.New(), base); err == nil {
		t.Fatalf("expected manifest lookup to fail without manifest.json")
	}
}
