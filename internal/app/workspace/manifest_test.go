package workspace

import (
	"errors"
	"os"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
	"github.com/jakeraft/clier/internal/domain"
	storemanifest "github.com/jakeraft/clier/internal/store/manifest"
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
			LocalPath:     teamTrackedPath("jakeraft", "dev-squad"),
			RemoteVersion: &resourceVersion,
			BaseHash:      "abc123",
			Editable:      true,
		},
		TrackedResources: []TrackedResource{{
			Kind:          string(api.KindSkill),
			Owner:         "jakeraft",
			Name:          "reviewer",
			LocalPath:     "lead/.claude/skills/jakeraft.reviewer/SKILL.md",
			RemoteVersion: &resourceVersion,
			BaseHash:      "def456",
			Editable:      true,
		}},
	}

	if err := storemanifest.Save(filesystem.New(), base, meta); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	path := storemanifest.Path(base)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat manifest file: %v", err)
	}

	loaded, err := storemanifest.Load(filesystem.New(), base)
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

func TestManifest_AgentTeamClone(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	resourceVersion := 1
	meta := &Manifest{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "reviewer",
		RootResource: TrackedResource{
			Kind:      string(api.KindTeam),
			Owner:     "jakeraft",
			Name:      "reviewer",
			LocalPath: teamTrackedPath("jakeraft", "reviewer"),
			Editable:  true,
		},
		TrackedResources: []TrackedResource{{
			Kind:          string(api.KindTeam),
			Owner:         "jakeraft",
			Name:          "reviewer",
			LocalPath:     teamTrackedPath("jakeraft", "reviewer"),
			RemoteVersion: &resourceVersion,
			Editable:      true,
		}},
	}

	if err := storemanifest.Save(filesystem.New(), base, meta); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	loaded, err := storemanifest.Load(filesystem.New(), base)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if loaded.Kind != string(api.KindTeam) {
		t.Fatalf("Kind = %q, want %q", loaded.Kind, string(api.KindTeam))
	}
	if loaded.RootResource.LocalPath != teamTrackedPath("jakeraft", "reviewer") {
		t.Fatalf("root local path = %q, want %q", loaded.RootResource.LocalPath, teamTrackedPath("jakeraft", "reviewer"))
	}
	if len(loaded.TrackedResources) != 1 {
		t.Fatalf("expected 1 tracked resource, got %d", len(loaded.TrackedResources))
	}
	if loaded.TrackedResources[0].Owner != "jakeraft" {
		t.Fatalf("tracked owner = %q, want %q", loaded.TrackedResources[0].Owner, "jakeraft")
	}
}

func TestManifest_CompositeTeamClone(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	rootVersion := 2
	childVersion := 1
	meta := &Manifest{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "dev-squad",
		RootResource: TrackedResource{
			Kind:          string(api.KindTeam),
			Owner:         "jakeraft",
			Name:          "dev-squad",
			LocalPath:     teamTrackedPath("jakeraft", "dev-squad"),
			RemoteVersion: &rootVersion,
			Editable:      true,
		},
		TrackedResources: []TrackedResource{
			{
				Kind:          string(api.KindTeam),
				Owner:         "jakeraft",
				Name:          "dev-squad",
				LocalPath:     teamTrackedPath("jakeraft", "dev-squad"),
				RemoteVersion: &rootVersion,
				Editable:      true,
			},
			{
				Kind:          string(api.KindTeam),
				Owner:         "jakeraft",
				Name:          "reviewer",
				LocalPath:     teamTrackedPath("jakeraft", "reviewer"),
				RemoteVersion: &childVersion,
				Editable:      true,
			},
		},
	}

	if err := storemanifest.Save(filesystem.New(), base, meta); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	loaded, err := storemanifest.Load(filesystem.New(), base)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if loaded.Kind != string(api.KindTeam) {
		t.Fatalf("Kind = %q, want %q", loaded.Kind, string(api.KindTeam))
	}
	if len(loaded.TrackedResources) != 2 {
		t.Fatalf("expected 2 tracked resources, got %d", len(loaded.TrackedResources))
	}
	if loaded.TrackedResources[1].LocalPath != teamTrackedPath("jakeraft", "reviewer") {
		t.Fatalf("child local path = %q, want %q", loaded.TrackedResources[1].LocalPath, teamTrackedPath("jakeraft", "reviewer"))
	}
}

func TestLoadManifest_RejectsOutdatedFormat(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	// Write a manifest with format 0 (simulating a legacy clone).
	data := []byte(`{"format":0,"kind":"team","owner":"jakeraft","name":"reviewer"}`)
	if err := fs.EnsureFile(storemanifest.Path(base), data); err != nil {
		t.Fatalf("write legacy manifest: %v", err)
	}

	_, err := storemanifest.Load(fs, base)
	if err == nil {
		t.Fatal("expected error for outdated manifest format")
	}
	var f *domain.Fault
	if !errors.As(err, &f) || f.Kind != domain.KindManifestIncompatible {
		t.Fatalf("expected KindManifestIncompatible, got %v", err)
	}
	if f.Subject["hint"] != "re-clone with 'clier clone'" {
		t.Fatalf("hint should mention re-clone, got %q", f.Subject["hint"])
	}
}

func TestLoadManifest_RejectsNewerFormat(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fs := filesystem.New()

	// Write a manifest with a future format version.
	data := []byte(`{"format":999,"kind":"team","owner":"jakeraft","name":"reviewer"}`)
	if err := fs.EnsureFile(storemanifest.Path(base), data); err != nil {
		t.Fatalf("write future manifest: %v", err)
	}

	_, err := storemanifest.Load(fs, base)
	if err == nil {
		t.Fatal("expected error for newer manifest format")
	}
	var f *domain.Fault
	if !errors.As(err, &f) || f.Kind != domain.KindManifestIncompatible {
		t.Fatalf("expected KindManifestIncompatible, got %v", err)
	}
	if f.Subject["hint"] != "upgrade clier" {
		t.Fatalf("hint should suggest upgrade, got %q", f.Subject["hint"])
	}
}

func TestLoadManifest_RequiresManifestPath(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if _, err := storemanifest.Load(filesystem.New(), base); err == nil {
		t.Fatalf("expected manifest lookup to fail without state.json")
	}
}
