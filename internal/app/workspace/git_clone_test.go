package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureRepoDir_WithoutRepoURLCreatesDirectory(t *testing.T) {
	t.Parallel()

	repoDir := filepath.Join(t.TempDir(), "clier_todo")
	if err := ensureRepoDir("", repoDir); err != nil {
		t.Fatalf("ensureRepoDir: %v", err)
	}

	info, err := os.Stat(repoDir)
	if err != nil {
		t.Fatalf("stat repo dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("repo dir should be a directory")
	}
}

func TestEnsureRepoDir_ClonesRepoAndReusesSameOrigin(t *testing.T) {
	t.Parallel()

	repoURL := newTestRepo(t)
	repoDir := filepath.Join(t.TempDir(), "clier_todo")

	if err := ensureRepoDir(repoURL, repoDir); err != nil {
		t.Fatalf("ensureRepoDir initial clone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		t.Fatalf("stat .git: %v", err)
	}

	if err := ensureRepoDir(repoURL, repoDir); err != nil {
		t.Fatalf("ensureRepoDir second clone: %v", err)
	}
}

func TestEnsureRepoDir_RejectsDifferentOrigin(t *testing.T) {
	t.Parallel()

	firstRepoURL := newTestRepo(t)
	secondRepoURL := newTestRepo(t)
	repoDir := filepath.Join(t.TempDir(), "clier_todo")

	if err := ensureRepoDir(firstRepoURL, repoDir); err != nil {
		t.Fatalf("ensureRepoDir initial clone: %v", err)
	}
	if err := ensureRepoDir(secondRepoURL, repoDir); err == nil {
		t.Fatalf("ensureRepoDir should reject a different origin")
	}
}

func TestEnsureRepoDir_RejectsNonGitDirectory(t *testing.T) {
	t.Parallel()

	repoURL := newTestRepo(t)
	repoDir := filepath.Join(t.TempDir(), "clier_todo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("mkdir repo dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("local"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}

	if err := ensureRepoDir(repoURL, repoDir); err == nil {
		t.Fatalf("ensureRepoDir should reject a non-git directory")
	}
}

func TestEnsureRepoDir_DoesNotTreatParentRepositoryAsTargetRepository(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	runGit(t, parent, "init")
	runGit(t, parent, "config", "user.name", "Codex")
	runGit(t, parent, "config", "user.email", "codex@example.com")

	repoURL := newTestRepo(t)
	target := filepath.Join(parent, "child")
	if err := os.MkdirAll(filepath.Join(target, ".clier"), 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	err := ensureRepoDir(repoURL, target)
	if err == nil {
		t.Fatalf("ensureRepoDir should reject a non-git child directory inside another repository")
	}
	if !strings.Contains(err.Error(), "is not a git repo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newTestRepo(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	remoteDir := filepath.Join(root, "remote.git")
	runGit(t, root, "init", "--bare", remoteDir)
	runGit(t, root, "init", sourceDir)
	runGit(t, sourceDir, "config", "user.name", "Codex")
	runGit(t, sourceDir, "config", "user.email", "codex@example.com")

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
