package cmd

import (
	"os"
	"path/filepath"
	"testing"

	appclone "github.com/jakeraft/clier/internal/app/clone"
)

func TestDefaultCloneBase(t *testing.T) {
	t.Parallel()

	base := "/tmp/clier"
	if got := defaultCloneBase(base, cloneTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}); got != filepath.Join(base, "jakeraft", "reviewer") {
		t.Fatalf("member clone base = %q", got)
	}

	if got := defaultCloneBase(base, cloneTarget{
		Kind:  resourceKindTeam,
		Owner: "jakeraft",
		Name:  "todo-team",
	}); got != filepath.Join(base, "jakeraft", "todo-team") {
		t.Fatalf("team clone base = %q", got)
	}
}

func TestShouldReuseCloneRoot(t *testing.T) {
	t.Parallel()

	target := cloneTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}
	meta := &appclone.CloneMetadata{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}

	if !shouldReuseCloneRoot(target, "/tmp/clier/jakeraft/reviewer", meta) {
		t.Fatalf("expected matching clone root to be reused")
	}
	if shouldReuseCloneRoot(target, "/tmp/clier/jakeraft/reviewer", &appclone.CloneMetadata{
		Kind:  resourceKindMember,
		Owner: "other",
		Name:  "reviewer",
	}) {
		t.Fatalf("did not expect mismatched clone metadata to be reused")
	}
}

func TestResolveCloneCreateBase_FailsWhenTargetAlreadyExists(t *testing.T) {
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

	_, err := resolveCloneCreateBase(cloneTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	})
	if err == nil {
		t.Fatalf("expected existing destination to fail")
	}
}

func TestResolveCloneCreateBase_FailsInsideExistingClone(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appclone.SaveCloneMetadata(base, &appclone.CloneMetadata{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}); err != nil {
		t.Fatalf("SaveCloneMetadata: %v", err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(base); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	_, err := resolveCloneCreateBase(cloneTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	})
	if err == nil {
		t.Fatalf("expected existing clone root to fail")
	}
}

func TestRequireCurrentCloneRoot_RequiresDirectCloneOwner(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appclone.SaveCloneMetadata(base, &appclone.CloneMetadata{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}); err != nil {
		t.Fatalf("SaveCloneMetadata: %v", err)
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

	_, _, err := requireCurrentCloneRoot(cloneTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}, "`clier member run`")
	if err == nil {
		t.Fatalf("expected nested directory to fail")
	}
}

func TestRequireCurrentCloneRoot_LoadsCurrentCloneMetadata(t *testing.T) {
	base := filepath.Join(t.TempDir(), "jakeraft", "reviewer")
	if err := appclone.SaveCloneMetadata(base, &appclone.CloneMetadata{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}); err != nil {
		t.Fatalf("SaveCloneMetadata: %v", err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(base); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	gotBase, meta, err := requireCurrentCloneRoot(cloneTarget{
		Kind:  resourceKindMember,
		Owner: "jakeraft",
		Name:  "reviewer",
	}, "`clier member run`")
	if err != nil {
		t.Fatalf("requireCurrentCloneRoot: %v", err)
	}
	wantBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if gotBase != wantBase {
		t.Fatalf("base = %q, want %q", gotBase, wantBase)
	}
	if meta.Kind != resourceKindMember || meta.Owner != "jakeraft" || meta.Name != "reviewer" {
		t.Fatalf("unexpected clone metadata: %+v", meta)
	}
}
