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
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "reviewer",
	}); got != filepath.Join(base, "jakeraft", "reviewer") {
		t.Fatalf("member clone dir = %q", got)
	}

	if got := defaultCloneDir(base, resourceTarget{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "todo-team",
	}); got != filepath.Join(base, "jakeraft", "todo-team") {
		t.Fatalf("team clone dir = %q", got)
	}
}

func TestMatchesWorkingCopyTarget(t *testing.T) {
	t.Parallel()

	target := resourceTarget{
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "reviewer",
	}
	meta := &appworkspace.Manifest{
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "reviewer",
	}

	if !matchesWorkingCopyTarget(target, "/tmp/clier/jakeraft/reviewer", meta) {
		t.Fatalf("expected matching working-copy root to be reused")
	}
	if matchesWorkingCopyTarget(target, "/tmp/clier/jakeraft/reviewer", &appworkspace.Manifest{
		Kind:  string(api.KindMember),
		Owner: "other",
		Name:  "reviewer",
	}) {
		t.Fatalf("did not expect mismatched manifest to be reused")
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
		Kind:  string(api.KindMember),
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
		Kind:  string(api.KindMember),
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

func TestResolveCloneBase_FailsAtExistingTargetWorkingCopy(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appworkspace.SaveManifest(filesystem.New(), base, &appworkspace.Manifest{
		Kind:  string(api.KindMember),
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
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "reviewer",
	})
	if err == nil {
		t.Fatalf("expected existing working-copy root to fail")
	}
}

func TestRequireCurrentCopyRoot_RequiresDirectOwner(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appworkspace.SaveManifest(filesystem.New(), base, &appworkspace.Manifest{
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "reviewer",
	}); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	nested := filepath.Join(base, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	_, _, err := requireCurrentCopyRoot(resourceTarget{
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "reviewer",
	}, "`clier run start`")
	if err == nil {
		t.Fatalf("expected nested directory to fail")
	}
}

func TestRequireCurrentCopyRoot_LoadsCurrentManifest(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appworkspace.SaveManifest(filesystem.New(), base, &appworkspace.Manifest{
		Kind:  string(api.KindMember),
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

	gotBase, meta, err := requireCurrentCopyRoot(resourceTarget{
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "reviewer",
	}, "`clier run start`")
	if err != nil {
		t.Fatalf("requireCurrentWorkspaceRoot: %v", err)
	}
	wantBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if gotBase != wantBase {
		t.Fatalf("base = %q, want %q", gotBase, wantBase)
	}
	if meta.Kind != string(api.KindMember) || meta.Owner != "jakeraft" || meta.Name != "reviewer" {
		t.Fatalf("unexpected manifest: %+v", meta)
	}
}
