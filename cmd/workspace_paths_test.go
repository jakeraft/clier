package cmd

import (
	"os"
	"path/filepath"
	"testing"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func TestDefaultWorkspaceBase(t *testing.T) {
	t.Parallel()

	base := "/tmp/clier"
	if got := defaultWorkspaceBase(base, workspaceTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}); got != filepath.Join(base, "jakeraft", "reviewer") {
		t.Fatalf("member workspace base = %q", got)
	}

	if got := defaultWorkspaceBase(base, workspaceTarget{
		Kind:  resourceKindTeam,
		Owner: "jakeraft",
		Name:  "todo-team",
	}); got != filepath.Join(base, "jakeraft", "todo-team") {
		t.Fatalf("team workspace base = %q", got)
	}
}

func TestShouldReuseWorkspaceRoot(t *testing.T) {
	t.Parallel()

	target := workspaceTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}
	meta := &appworkspace.Manifest{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}

	if !shouldReuseWorkspaceRoot(target, "/tmp/clier/jakeraft/reviewer", meta) {
		t.Fatalf("expected matching workspace root to be reused")
	}
	if shouldReuseWorkspaceRoot(target, "/tmp/clier/jakeraft/reviewer", &appworkspace.Manifest{
		Kind:  resourceKindMember,
		Owner: "other",
		Name:  "reviewer",
	}) {
		t.Fatalf("did not expect mismatched workspace manifest to be reused")
	}
}

func TestResolveWorkspaceCreateBase_FailsWhenTargetAlreadyExists(t *testing.T) {
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

	_, err := resolveWorkspaceCreateBase(workspaceTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	})
	if err == nil {
		t.Fatalf("expected existing destination to fail")
	}
}

func TestResolveWorkspaceCreateBase_FailsInsideExistingWorkspace(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appworkspace.SaveManifest(base, &appworkspace.Manifest{
		Kind:  resourceKindMember,
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

	_, err := resolveWorkspaceCreateBase(workspaceTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	})
	if err == nil {
		t.Fatalf("expected existing workspace root to fail")
	}
}

func TestRequireCurrentWorkspaceRoot_RequiresDirectWorkspaceOwner(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appworkspace.SaveManifest(base, &appworkspace.Manifest{
		Kind:  resourceKindMember,
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

	_, _, err := requireCurrentWorkspaceRoot(workspaceTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}, "`clier member run`")
	if err == nil {
		t.Fatalf("expected nested directory to fail")
	}
}

func TestRequireCurrentWorkspaceRoot_LoadsCurrentManifest(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appworkspace.SaveManifest(base, &appworkspace.Manifest{
		Kind:  resourceKindMember,
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

	gotBase, meta, err := requireCurrentWorkspaceRoot(workspaceTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}, "`clier member run`")
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
	if meta.Kind != resourceKindMember || meta.Owner != "jakeraft" || meta.Name != "reviewer" {
		t.Fatalf("unexpected workspace manifest: %+v", meta)
	}
}
