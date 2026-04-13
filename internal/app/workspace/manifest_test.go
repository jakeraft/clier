package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
)

func TestSaveManifest(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	resourceVersion := 3
	fetchedVersion := 7
	fetchedAt := timeRef(t)
	meta := &Manifest{
		Kind:     string(api.KindTeam),
		Owner:    "jakeraft",
		Name:     "dev-squad",
		Upstream: &UpstreamMetadata{Kind: string(api.KindTeam), Owner: "origin", Name: "dev-squad", FetchedVersion: &fetchedVersion, FetchedAt: fetchedAt},
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
	if loaded.Upstream == nil || loaded.Upstream.Owner != "origin" || loaded.Upstream.FetchedVersion == nil || *loaded.Upstream.FetchedVersion != fetchedVersion {
		t.Fatalf("loaded upstream mismatch: %#v", loaded.Upstream)
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

func TestLoadManifest_RequiresManifestPath(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if _, err := LoadManifest(filesystem.New(), base); err == nil {
		t.Fatalf("expected manifest lookup to fail without manifest.json")
	}
}

func timeRef(t *testing.T) *time.Time {
	t.Helper()
	now := time.Now().UTC()
	return &now
}
