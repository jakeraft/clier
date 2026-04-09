package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveManifest(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	version := 7
	resourceVersion := 3
	meta := &Manifest{
		Kind:          "team",
		Owner:         "jakeraft",
		Name:          "dev-squad",
		Materializer:  "local-git",
		LatestVersion: &version,
		Resources: []ResourceManifest{{
			Kind:          "skill",
			Owner:         "jakeraft",
			Name:          "reviewer",
			LocalPath:     "lead/.claude/skills/reviewer/SKILL.md",
			LatestVersion: &resourceVersion,
		}},
	}

	if err := SaveManifest(base, meta); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	path := filepath.Join(base, ".clier", WorkspaceMetadataFile)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat metadata file: %v", err)
	}

	loaded, err := LoadManifest(base)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if loaded.Kind != meta.Kind || loaded.Owner != meta.Owner || loaded.Name != meta.Name {
		t.Fatalf("loaded metadata mismatch: %#v", loaded)
	}
	if loaded.Materializer != meta.Materializer {
		t.Fatalf("loaded materializer mismatch: %#v", loaded.Materializer)
	}
	if loaded.LatestVersion == nil || *loaded.LatestVersion != version {
		t.Fatalf("loaded latest version mismatch: %#v", loaded.LatestVersion)
	}
	if len(loaded.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(loaded.Resources))
	}
	if loaded.Resources[0].LocalPath != meta.Resources[0].LocalPath {
		t.Fatalf("loaded resource local path mismatch: %#v", loaded.Resources[0])
	}
}

func TestLoadManifest_RequiresWorkspaceMetadataPath(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if _, err := LoadManifest(base); err == nil {
		t.Fatalf("expected workspace metadata lookup to fail without workspace.json")
	}
}
