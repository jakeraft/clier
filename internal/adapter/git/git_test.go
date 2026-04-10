package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExecGit_CloneAndIsRepoAndOrigin(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	repoURL := newTestRepo(t)
	g := New()

	targetDir := filepath.Join(t.TempDir(), "clone")
	if err := g.Clone(repoURL, targetDir); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	isRepo, err := g.IsRepo(targetDir)
	if err != nil {
		t.Fatalf("IsRepo: %v", err)
	}
	if !isRepo {
		t.Fatalf("expected directory to be a git repo")
	}

	origin, err := g.Origin(targetDir)
	if err != nil {
		t.Fatalf("Origin: %v", err)
	}
	if origin != repoURL {
		t.Fatalf("origin = %q, want %q", origin, repoURL)
	}
}

func TestExecGit_IsRepo_ReturnsFalseForNonRepo(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	g := New()
	dir := t.TempDir()
	isRepo, err := g.IsRepo(dir)
	if err != nil {
		t.Fatalf("IsRepo: %v", err)
	}
	if isRepo {
		t.Fatalf("expected non-repo directory to return false")
	}
}

func TestExecGit_Diff(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	g := New()
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.json")
	fileB := filepath.Join(dir, "b.json")
	if err := os.WriteFile(fileA, []byte(`{"name":"alpha"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileB, []byte(`{"name":"beta"}`), 0644); err != nil {
		t.Fatal(err)
	}

	diff, hasChanges, err := g.Diff(fileA, fileB)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !hasChanges {
		t.Fatalf("expected changes between different files")
	}
	if diff == "" {
		t.Fatalf("expected non-empty diff output")
	}

	_, hasChanges, err = g.Diff(fileA, fileA)
	if err != nil {
		t.Fatalf("Diff same: %v", err)
	}
	if hasChanges {
		t.Fatalf("expected no changes for identical files")
	}
}

func newTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	remoteDir := filepath.Join(root, "remote.git")
	runGit(t, root, "init", "--bare", remoteDir)
	runGit(t, root, "init", sourceDir)
	runGit(t, sourceDir, "config", "user.name", "Test")
	runGit(t, sourceDir, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, sourceDir, "add", "README.md")
	runGit(t, sourceDir, "commit", "-m", "initial commit")
	runGit(t, sourceDir, "remote", "add", "origin", remoteDir)
	runGit(t, sourceDir, "push", "origin", "HEAD")
	return remoteDir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}
