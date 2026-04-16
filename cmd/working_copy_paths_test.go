package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func TestDefaultCloneDir(t *testing.T) {
	t.Parallel()

	base := "/tmp/clier"
	if got := defaultCloneDir(base, resourceTarget{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "reviewer",
	}); got != filepath.Join(base, "jakeraft", "reviewer") {
		t.Fatalf("clone dir = %q", got)
	}

	if got := defaultCloneDir(base, resourceTarget{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "todo-team",
	}); got != filepath.Join(base, "jakeraft", "todo-team") {
		t.Fatalf("clone dir = %q", got)
	}
}

func TestResolveCloneBase_FailsWhenTargetAlreadyExists(t *testing.T) {
	base := t.TempDir()
	targetDir := filepath.Join(base, "jakeraft", "reviewer")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(base); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	_, err := resolveCloneBase(resourceTarget{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "reviewer",
	})
	if err == nil {
		t.Fatalf("expected existing destination to fail")
	}
}

func TestResolveCloneBase_FailsInsideExistingWorkingCopy(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appworkspace.SaveManifest(filesystem.New(), base, &appworkspace.Manifest{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "reviewer",
	}); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(base); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	_, err := resolveCloneBase(resourceTarget{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "todo-team",
	})
	if err == nil {
		t.Fatalf("expected existing working-copy root to fail")
	}
}
