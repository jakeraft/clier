# Clone Adapter Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract all direct filesystem and git I/O from `internal/app/workspace/` into adapter/port boundaries so the app layer has zero `"os"` and `"os/exec"` imports.

**Architecture:** Two ports (`FileMaterializer`, `GitRepo`) defined in the workspace package, implemented by `LocalFS` and `ExecGit` adapters, injected via constructor into `Service` and `Writer`.

**Tech Stack:** Go 1.25, `io/fs` for interface types, `os` and `os/exec` only in adapter implementations.

---

### Task 1: Create Port Definitions

**Files:**
- Create: `internal/app/workspace/port.go`

- [ ] **Step 1: Create port.go with interface definitions**

```go
package workspace

import "io/fs"

// FileMaterializer abstracts local filesystem operations used during
// clone, pull, push, and status workflows.
type FileMaterializer interface {
	WriteFile(path string, content []byte) error
	ReadFile(path string) ([]byte, error)
	MkdirAll(path string) error
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	MkdirTemp(pattern string) (string, error)
	RemoveAll(path string) error
}

// GitRepo abstracts git CLI operations used during clone and diff workflows.
type GitRepo interface {
	Clone(repoURL, targetDir string) error
	IsRepo(dir string) (bool, error)
	Origin(dir string) (string, error)
	Diff(pathA, pathB string) (string, bool, error)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/jake_kakao/jakeraft/clier && go build ./internal/app/workspace/`
Expected: success (no consumers yet, just interface definitions)

- [ ] **Step 3: Commit**

```bash
git add internal/app/workspace/port.go
git commit -m "refactor: define FileMaterializer and GitRepo port interfaces"
```

---

### Task 2: Create Filesystem Adapter

**Files:**
- Create: `internal/adapter/filesystem/filesystem.go`
- Create: `internal/adapter/filesystem/filesystem_test.go`

- [ ] **Step 1: Write the test**

```go
package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFS_WriteAndReadFile(t *testing.T) {
	t.Parallel()
	fs := New()
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "file.txt")

	if err := fs.WriteFile(path, []byte("hello")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("content = %q, want %q", string(data), "hello")
	}
}

func TestLocalFS_WriteFileCreatesParentDirs(t *testing.T) {
	t.Parallel()
	fs := New()
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "file.txt")

	if err := fs.WriteFile(path, []byte("nested")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestLocalFS_StatAndReadDir(t *testing.T) {
	t.Parallel()
	lfs := New()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := lfs.Stat(dir)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected directory")
	}

	entries, err := lfs.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "a.txt" {
		t.Fatalf("unexpected entries: %v", entries)
	}
}

func TestLocalFS_MkdirTempAndRemoveAll(t *testing.T) {
	t.Parallel()
	lfs := New()

	dir, err := lfs.MkdirTemp("clier-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("temp dir not created: %v", err)
	}
	if err := lfs.RemoveAll(dir); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("temp dir not removed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/adapter/filesystem/`
Expected: FAIL — `New` not defined

- [ ] **Step 3: Write the implementation**

```go
package filesystem

import (
	"os"
	"path/filepath"
)

// LocalFS implements workspace.FileMaterializer using the local filesystem.
type LocalFS struct{}

func New() *LocalFS {
	return &LocalFS{}
}

func (f *LocalFS) WriteFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

func (f *LocalFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *LocalFS) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func (f *LocalFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (f *LocalFS) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (f *LocalFS) MkdirTemp(pattern string) (string, error) {
	return os.MkdirTemp("", pattern)
}

func (f *LocalFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/adapter/filesystem/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/filesystem/
git commit -m "refactor: add LocalFS filesystem adapter"
```

---

### Task 3: Create Git Adapter

**Files:**
- Create: `internal/adapter/git/git.go`
- Create: `internal/adapter/git/git_test.go`

- [ ] **Step 1: Write the test**

```go
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

	// Same file should report no changes.
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/adapter/git/`
Expected: FAIL — `New` not defined

- [ ] **Step 3: Write the implementation**

Logic is extracted from the current `internal/app/workspace/git_repo.go` functions `gitClone`, `isGitRepo`, `gitOrigin` and the `git diff` call in `upstream.go`.

```go
package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecGit implements workspace.GitRepo using the git CLI.
type ExecGit struct{}

func New() *ExecGit {
	return &ExecGit{}
}

func (g *ExecGit) Clone(repoURL, targetDir string) error {
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return fmt.Errorf("create repo parent dir: %w", err)
	}
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, targetDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone %s: %w: %s", repoURL, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (g *ExecGit) IsRepo(dir string) (bool, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(strings.ToLower(string(out)), "not a git repository") {
			return false, nil
		}
		return false, fmt.Errorf("git rev-parse --show-toplevel: %w: %s", err, strings.TrimSpace(string(out)))
	}
	topLevel := strings.TrimSpace(string(out))
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false, fmt.Errorf("abs dir: %w", err)
	}
	topLevel, err = filepath.EvalSymlinks(topLevel)
	if err != nil {
		return false, fmt.Errorf("eval git top-level: %w", err)
	}
	absDir, err = filepath.EvalSymlinks(absDir)
	if err != nil {
		return false, fmt.Errorf("eval dir: %w", err)
	}
	return filepath.Clean(topLevel) == filepath.Clean(absDir), nil
}

func (g *ExecGit) Origin(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "config", "--get", "remote.origin.url")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git config remote.origin.url: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *ExecGit) Diff(pathA, pathB string) (string, bool, error) {
	cmd := exec.Command("git", "diff", "--no-index", "--no-color", "--", pathA, pathB)
	output, err := cmd.CombinedOutput()
	switch {
	case err == nil:
		return string(output), false, nil
	case diffExitCode(err) == 1:
		return string(output), true, nil
	default:
		return "", false, fmt.Errorf("git diff: %w", err)
	}
}

func diffExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/adapter/git/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/git/
git commit -m "refactor: add ExecGit adapter for git CLI operations"
```

---

### Task 4: Refactor manifest.go — Add fs Parameter

**Files:**
- Modify: `internal/app/workspace/manifest.go`

- [ ] **Step 1: Update all public functions to take `FileMaterializer` as first parameter**

Replace the entire `import` block and all functions that use `os.*`:

Replace `import` block:
```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)
```
with:
```go
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)
```

Replace `FindManifestPath`:
```go
func FindManifestPath(fs FileMaterializer, base string) (string, error) {
	path := ManifestPath(base)
	if _, err := fs.Stat(path); err == nil {
		return path, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat working-copy manifest: %w", err)
	}
	return "", os.ErrNotExist
}
```

Replace `FindManifestAbove`:
```go
func FindManifestAbove(fs FileMaterializer, start string) (string, *Manifest, error) {
	base, err := filepath.Abs(start)
	if err != nil {
		return "", nil, fmt.Errorf("resolve working-copy base: %w", err)
	}
	for dir := base; ; dir = filepath.Dir(dir) {
		if _, err := FindManifestPath(fs, dir); err == nil {
			manifest, loadErr := LoadManifest(fs, dir)
			if loadErr != nil {
				return "", nil, loadErr
			}
			return dir, manifest, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", nil, os.ErrNotExist
}
```

Replace `SaveManifest`:
```go
func SaveManifest(fs FileMaterializer, base string, manifest *Manifest) error {
	dir := filepath.Join(base, ".clier")
	if err := fs.MkdirAll(dir); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := fs.WriteFile(ManifestPath(base), data); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}
```

Replace `LoadManifest`:
```go
func LoadManifest(fs FileMaterializer, base string) (*Manifest, error) {
	path, err := FindManifestPath(fs, base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("read manifest: %w", err)
		}
		return nil, err
	}
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	return &manifest, nil
}
```

Note: `os` import still needed for `os.IsNotExist` and `os.ErrNotExist`. These are error checks, not I/O.

- [ ] **Step 2: Verify compilation fails (callers not yet updated)**

Run: `cd /Users/jake_kakao/jakeraft/clier && go build ./... 2>&1 | head -20`
Expected: FAIL — callers pass wrong number of arguments

- [ ] **Step 3: Commit (partial — will fix callers in later tasks)**

```bash
git add internal/app/workspace/manifest.go
git commit -m "refactor: add FileMaterializer param to manifest functions"
```

---

### Task 5: Refactor projection.go — Add fs Parameter

**Files:**
- Modify: `internal/app/workspace/projection.go`

- [ ] **Step 1: Update all I/O functions to take `FileMaterializer`**

Replace `writeJSONProjection`:
```go
func writeJSONProjection(fs FileMaterializer, path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal projection: %w", err)
	}
	if err := fs.WriteFile(path, data); err != nil {
		return fmt.Errorf("write projection: %w", err)
	}
	return nil
}
```

Note: `MkdirAll` is no longer needed because `LocalFS.WriteFile` auto-creates parent dirs.

Replace `loadJSONProjection`:
```go
func loadJSONProjection(fs FileMaterializer, path string, payload any) error {
	data, err := fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read projection: %w", err)
	}
	if err := json.Unmarshal(data, payload); err != nil {
		return fmt.Errorf("unmarshal projection: %w", err)
	}
	return nil
}
```

Replace public wrappers:
```go
func WriteMemberProjection(fs FileMaterializer, path string, projection *MemberProjection) error {
	return writeJSONProjection(fs, path, projection)
}

func WriteTeamProjection(fs FileMaterializer, path string, projection *TeamProjection) error {
	return writeJSONProjection(fs, path, projection)
}

func LoadMemberProjection(fs FileMaterializer, path string) (*MemberProjection, error) {
	var projection MemberProjection
	if err := loadJSONProjection(fs, path, &projection); err != nil {
		return nil, err
	}
	return &projection, nil
}

func LoadTeamProjection(fs FileMaterializer, path string) (*TeamProjection, error) {
	var projection TeamProjection
	if err := loadJSONProjection(fs, path, &projection); err != nil {
		return nil, err
	}
	return &projection, nil
}
```

Remove `"os"` from the import block (no longer needed).

- [ ] **Step 2: Commit**

```bash
git add internal/app/workspace/projection.go
git commit -m "refactor: add FileMaterializer param to projection functions"
```

---

### Task 6: Refactor git_repo.go — Move Git Logic to Adapter

**Files:**
- Modify: `internal/app/workspace/git_repo.go`

- [ ] **Step 1: Remove git CLI functions, keep orchestration with port params**

Replace the entire file content with:

```go
package workspace

import (
	"fmt"
	"os"
)

func ensureRepoDir(fs FileMaterializer, git GitRepo, repoURL, repoDir string) error {
	if repoURL == "" {
		return fs.MkdirAll(repoDir)
	}

	info, err := fs.Stat(repoDir)
	if err != nil {
		if os.IsNotExist(err) {
			return git.Clone(repoURL, repoDir)
		}
		return fmt.Errorf("stat repo dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repo path %s exists and is not a directory", repoDir)
	}

	entries, err := fs.ReadDir(repoDir)
	if err != nil {
		return fmt.Errorf("read repo dir: %w", err)
	}
	if len(entries) == 0 {
		return git.Clone(repoURL, repoDir)
	}

	isRepo, err := git.IsRepo(repoDir)
	if err != nil {
		return fmt.Errorf("check git repo: %w", err)
	}
	if isRepo {
		originURL, err := git.Origin(repoDir)
		if err != nil {
			return fmt.Errorf("read git origin: %w", err)
		}
		if originURL != repoURL {
			return fmt.Errorf("repo dir %s already tracks %s, not %s", repoDir, originURL, repoURL)
		}
		return nil
	}

	return fmt.Errorf("repo dir %s already exists and is not a git repo", repoDir)
}

func IsMaterializedRoot(fs FileMaterializer, git GitRepo, repoURL, root string) (bool, error) {
	info, err := fs.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return false, nil
	}

	if repoURL != "" {
		return git.IsRepo(root)
	}

	entries, err := fs.ReadDir(root)
	if err != nil {
		return false, fmt.Errorf("read root: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() != ".clier" {
			return true, nil
		}
	}
	return false, nil
}
```

Remove `"os/exec"` from imports entirely. `"os"` stays only for `os.IsNotExist`.

- [ ] **Step 2: Commit**

```bash
git add internal/app/workspace/git_repo.go
git commit -m "refactor: move git CLI logic to adapter, keep orchestration"
```

---

### Task 7: Refactor writer.go — Inject fs and git

**Files:**
- Modify: `internal/app/workspace/writer.go`

- [ ] **Step 1: Update Writer struct, constructor, and all methods**

Replace `Writer` struct and constructor:
```go
type Writer struct {
	client *api.Client
	owner  string
	fs     FileMaterializer
	git    GitRepo
}

func NewWriter(client *api.Client, owner string, fs FileMaterializer, git GitRepo) *Writer {
	return &Writer{client: client, owner: owner, fs: fs, git: git}
}
```

Convert `writeFile` to Writer method:
```go
func (w *Writer) writeFile(path, content string) error {
	return w.fs.WriteFile(path, []byte(content))
}
```

Note: `MkdirAll` is no longer needed — `LocalFS.WriteFile` auto-creates parent dirs.

Convert `writeLocalSettings` to Writer method:
```go
func (w *Writer) writeLocalSettings(base string, profile domain.AgentProfile) error {
	if profile.LocalSettingsFile == "" {
		return nil
	}
	content, err := localSettingsContent(profile)
	if err != nil {
		return err
	}
	return w.writeFile(filepath.Join(base, profile.SettingsDir, profile.LocalSettingsFile), content)
}
```

Convert `writeWorkLogProtocol` to Writer method:
```go
func (w *Writer) writeWorkLogProtocol(base string) error {
	return w.writeFile(filepath.Join(base, ".clier", workLogProtocolFileName), BuildAgentFacingWorkLogProtocol())
}
```

Update `materializeMemberFilesFromResponse` — change `ensureRepoDir` call and method calls:

Replace:
```go
	if err := ensureRepoDir(member.GitRepoURL, base); err != nil {
```
with:
```go
	if err := ensureRepoDir(w.fs, w.git, member.GitRepoURL, base); err != nil {
```

Replace all `writeFile(` calls with `w.writeFile(`, `writeLocalSettings(` with `w.writeLocalSettings(`, `writeWorkLogProtocol(` with `w.writeWorkLogProtocol(`.

Update `MaterializeTeamFiles` — replace `writeFile(protocolPath, protocol)` with `w.writeFile(protocolPath, protocol)`.

The `"os"` import remains only for `os.UserHomeDir()` in `localSettingsContent`. All file I/O now goes through `w.fs`.

- [ ] **Step 2: Commit**

```bash
git add internal/app/workspace/writer.go
git commit -m "refactor: inject FileMaterializer and GitRepo into Writer"
```

---

### Task 8: Refactor service.go — Inject fs and git, Convert Standalone Functions

**Files:**
- Modify: `internal/app/workspace/service.go`

- [ ] **Step 1: Update Service struct and constructor**

Replace:
```go
type Service struct {
	client *api.Client
}
```
with:
```go
type Service struct {
	client *api.Client
	fs     FileMaterializer
	git    GitRepo
}
```

Replace:
```go
func NewService(client *api.Client) *Service {
	return &Service{client: client}
}
```
with:
```go
func NewService(client *api.Client, fs FileMaterializer, git GitRepo) *Service {
	return &Service{client: client, fs: fs, git: git}
}
```

- [ ] **Step 2: Convert standalone functions to Service methods**

Convert `fileHash`:
```go
func (s *Service) fileHash(path string) (string, error) {
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
```

Convert `populateBaseHashes`:
```go
func (s *Service) populateBaseHashes(base string, tracked []TrackedResource) error {
	for i := range tracked {
		sum, err := s.fileHash(filepath.Join(base, filepath.FromSlash(tracked[i].LocalPath)))
		if err != nil {
			return err
		}
		tracked[i].BaseHash = sum
	}
	return nil
}
```

Convert `ModifiedTrackedResources` (changes from public standalone to public method):
```go
func (s *Service) ModifiedTrackedResources(base string) ([]TrackedResource, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}

	var modified []TrackedResource
	for _, resource := range manifest.TrackedResources {
		sum, err := s.fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
		if err != nil {
			return nil, err
		}
		if sum != resource.BaseHash {
			modified = append(modified, resource)
		}
	}
	return modified, nil
}
```

Convert `trackedStatuses`:
```go
func (s *Service) trackedStatuses(base string, manifest *Manifest) ([]TrackedStatus, int, error) {
	statuses := make([]TrackedStatus, 0, len(manifest.TrackedResources))
	modifiedCount := 0
	for _, resource := range manifest.TrackedResources {
		sum, err := s.fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
		if err != nil {
			return nil, 0, err
		}
		local := "clean"
		if sum != resource.BaseHash {
			local = "modified"
			modifiedCount++
		}
		statuses = append(statuses, TrackedStatus{
			Kind:  resource.Kind,
			Owner: resource.Owner,
			Name:  resource.Name,
			Path:  resource.LocalPath,
			Local: local,
		})
	}
	slices.SortFunc(statuses, func(a, b TrackedStatus) int {
		return strings.Compare(a.Path, b.Path)
	})
	return statuses, modifiedCount, nil
}
```

Convert `runSummary`:
```go
func (s *Service) runSummary(base string) (RunStatusSummary, error) {
	dir := filepath.Join(base, ".clier")
	entries, err := s.fs.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return RunStatusSummary{}, nil
		}
		return RunStatusSummary{}, fmt.Errorf("read runtime dir: %w", err)
	}
	var summary RunStatusSummary
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || name == ManifestFile {
			continue
		}
		plan, err := apprun.LoadPlanFromPath(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		summary.Total++
		if plan.Status == apprun.StatusRunning {
			summary.Running++
		} else {
			summary.Stopped++
		}
	}
	return summary, nil
}
```

Convert `serverClaudeMdContent`:
```go
func (s *Service) serverClaudeMdContent(base string, manifest *Manifest, resource TrackedResource) (string, error) {
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
	}
	content := string(data)
	clean := filepath.ToSlash(filepath.Clean(resource.LocalPath))
	if clean == filepath.ToSlash("CLAUDE.md") {
		return StripMemberClaudeMdPrelude(content), nil
	}
	if manifest.Runtime != nil && manifest.Runtime.Team != nil {
		for _, member := range manifest.Runtime.Team.Members {
			memberPath := filepath.ToSlash(filepath.Join(member.Name, "CLAUDE.md"))
			if clean == memberPath {
				return StripTeamClaudeMdPrelude(member.Name, content), nil
			}
		}
	}
	return content, nil
}
```

- [ ] **Step 3: Update all Service methods to use s.fs**

In `Pull`:
```go
func (s *Service) Pull(base string, force bool) (*Manifest, error) {
	manifest, err := LoadManifest(s.fs, base)
	// ...
	if err := SaveManifest(s.fs, base, pulled); err != nil {
```

In `pullTarget` — change `ModifiedTrackedResources(base)` to `s.ModifiedTrackedResources(base)`.

In `Status`:
```go
func (s *Service) Status(base string) (*Status, error) {
	manifest, err := LoadManifest(s.fs, base)
	// ...
	tracked, modifiedCount, err := s.trackedStatuses(base, manifest)
	// ...
	runs, err := s.runSummary(base)
```

In `Push`:
- Change `LoadManifest(base)` to `LoadManifest(s.fs, base)`
- Change `ModifiedTrackedResources(base)` to `s.ModifiedTrackedResources(base)`
- Change `LoadMemberProjection(...)` to `LoadMemberProjection(s.fs, ...)`
- Change `LoadTeamProjection(...)` to `LoadTeamProjection(s.fs, ...)`
- Change `serverClaudeMdContent(...)` to `s.serverClaudeMdContent(...)`
- Change `os.ReadFile(...)` (lines 267, 285) to `s.fs.ReadFile(...)`

In `materializeMember`:
- Change `NewWriter(s.client, owner)` to `NewWriter(s.client, owner, s.fs, s.git)`
- Change `WriteMemberProjection(...)` to `WriteMemberProjection(s.fs, ...)`
- Change `populateBaseHashes(...)` to `s.populateBaseHashes(...)`
- Change `SaveManifest(base, manifest)` to `SaveManifest(s.fs, base, manifest)`

In `materializeTeam`:
- Change `NewWriter(s.client, owner)` to `NewWriter(s.client, owner, s.fs, s.git)`
- Change `WriteTeamProjection(...)` to `WriteTeamProjection(s.fs, ...)`
- Change `WriteMemberProjection(...)` to `WriteMemberProjection(s.fs, ...)`
- Change `populateBaseHashes(...)` to `s.populateBaseHashes(...)`
- Change `SaveManifest(base, manifest)` to `SaveManifest(s.fs, base, manifest)`

- [ ] **Step 4: Remove `"os"` from imports**

The `"os"` import should only remain if `os.IsNotExist` is still used (in `runSummary`). Remove if unused.

- [ ] **Step 5: Commit**

```bash
git add internal/app/workspace/service.go
git commit -m "refactor: inject FileMaterializer and GitRepo into Service"
```

---

### Task 9: Refactor upstream.go — Use Service Adapters

**Files:**
- Modify: `internal/app/workspace/upstream.go`

- [ ] **Step 1: Update FetchUpstream**

Change `LoadManifest(base)` to `LoadManifest(s.fs, base)` and `SaveManifest(base, manifest)` to `SaveManifest(s.fs, base, manifest)`.

- [ ] **Step 2: Update writeFetchedUpstreamProjection**

Change `WriteMemberProjection(UpstreamProjectionPath(base), projection)` to `WriteMemberProjection(s.fs, UpstreamProjectionPath(base), projection)`.
Same for `WriteTeamProjection`.

- [ ] **Step 3: Update MergeFetchedUpstream**

Replace direct `os.ReadFile` / `os.WriteFile`:
```go
func (s *Service) MergeFetchedUpstream(base string) (*MergeUpstreamResult, error) {
	manifest, err := LoadManifest(s.fs, base)
	// ...
	modified, err := s.ModifiedTrackedResources(base)
	// ...
	data, err := s.fs.ReadFile(UpstreamProjectionPath(base))
	if err != nil {
		return nil, fmt.Errorf("read fetched upstream projection: %w", err)
	}
	if err := s.fs.WriteFile(localPath, data); err != nil {
		return nil, fmt.Errorf("write merged projection: %w", err)
	}
```

- [ ] **Step 4: Convert renderProjectionDiff to Service method**

```go
func (s *Service) renderProjectionDiff(localPath, upstreamPath string) (string, bool, error) {
	if _, err := s.fs.Stat(upstreamPath); err != nil {
		if os.IsNotExist(err) {
			return "", false, errors.New("no fetched upstream snapshot; run `clier fetch upstream` first")
		}
		return "", false, fmt.Errorf("stat fetched upstream projection: %w", err)
	}

	tempDir, err := s.fs.MkdirTemp("clier-upstream-diff-*")
	if err != nil {
		return "", false, fmt.Errorf("create temp diff dir: %w", err)
	}
	defer func() {
		_ = s.fs.RemoveAll(tempDir)
	}()

	localTempPath := filepath.Join(tempDir, "local.json")
	upstreamTempPath := filepath.Join(tempDir, "upstream.json")

	localData, err := s.fs.ReadFile(localPath)
	if err != nil {
		return "", false, fmt.Errorf("read file %s: %w", localPath, err)
	}
	if err := s.fs.WriteFile(localTempPath, localData); err != nil {
		return "", false, fmt.Errorf("write file %s: %w", localTempPath, err)
	}
	upstreamData, err := s.fs.ReadFile(upstreamPath)
	if err != nil {
		return "", false, fmt.Errorf("read file %s: %w", upstreamPath, err)
	}
	if err := s.fs.WriteFile(upstreamTempPath, upstreamData); err != nil {
		return "", false, fmt.Errorf("write file %s: %w", upstreamTempPath, err)
	}

	return s.git.Diff(localTempPath, upstreamTempPath)
}
```

Update `DiffFetchedUpstream` to call `s.renderProjectionDiff(...)` and use `LoadManifest(s.fs, base)`.

Delete the standalone `copyFile` and `diffCommandExitCode` functions (logic moved to adapter and service method).

- [ ] **Step 5: Remove `"os/exec"` from imports, keep `"os"` only if `os.IsNotExist` is used**

- [ ] **Step 6: Commit**

```bash
git add internal/app/workspace/upstream.go
git commit -m "refactor: migrate upstream.go to use FileMaterializer and GitRepo"
```

---

### Task 10: Update cmd/ Wiring

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/clone.go`
- Modify: `cmd/status.go`
- Modify: `cmd/push.go`
- Modify: `cmd/pull.go`
- Modify: `cmd/diff.go`
- Modify: `cmd/fetch.go`
- Modify: `cmd/merge.go`
- Modify: `cmd/helpers.go`
- Modify: `cmd/run.go`
- Modify: `cmd/working_copy_validation.go`
- Modify: `cmd/working_copy_paths.go`

- [ ] **Step 1: Add adapter factory helpers to cmd/root.go**

Add imports:
```go
"github.com/jakeraft/clier/internal/adapter/filesystem"
adaptergit "github.com/jakeraft/clier/internal/adapter/git"
```

Add functions:
```go
func newFileMaterializer() *filesystem.LocalFS {
	return filesystem.New()
}

func newGitRepo() *adaptergit.ExecGit {
	return adaptergit.New()
}
```

Note: import alias `adaptergit` avoids conflict with standard `"git"` references.

- [ ] **Step 2: Update all NewService calls (7 files)**

In `clone.go`, `status.go`, `push.go`, `pull.go`, `diff.go`, `fetch.go`, `merge.go`:

Replace:
```go
svc := appworkspace.NewService(newAPIClient())
```
with:
```go
svc := appworkspace.NewService(newAPIClient(), newFileMaterializer(), newGitRepo())
```

In `clone.go` specifically:
```go
svc := appworkspace.NewService(client, newFileMaterializer(), newGitRepo())
```

- [ ] **Step 3: Update FindManifestAbove calls**

In `helpers.go` (`resolveRuntimeDir`):
```go
copyRoot, _, err := appworkspace.FindManifestAbove(newFileMaterializer(), base)
```

In `status.go`:
```go
copyRoot, _, err := appworkspace.FindManifestAbove(newFileMaterializer(), base)
```

Same pattern for `push.go`, `pull.go`, `diff.go`, `fetch.go`, `merge.go`.

In `working_copy_paths.go` (`resolveCloneBase`):
```go
if copyRoot, _, err := appworkspace.FindManifestAbove(newFileMaterializer(), base); err == nil {
```

In `working_copy_paths.go` (`requireCurrentCopyRootKind`):
```go
copyRoot, manifest, err := appworkspace.FindManifestAbove(newFileMaterializer(), base)
```

- [ ] **Step 4: Update IsMaterializedRoot call**

In `working_copy_validation.go`:
```go
materialized, err := appworkspace.IsMaterializedRoot(newFileMaterializer(), newGitRepo(), member.GitRepoURL, base)
```

- [ ] **Step 5: Update LoadMemberProjection calls**

In `run.go`:
```go
memberProjection, err := appworkspace.LoadMemberProjection(newFileMaterializer(), appworkspace.MemberProjectionPath(copyRoot))
```

And for team members:
```go
memberProjection, err := appworkspace.LoadMemberProjection(newFileMaterializer(), appworkspace.TeamMemberProjectionPath(copyRoot, member.Name))
```

- [ ] **Step 6: Commit**

```bash
git add cmd/
git commit -m "refactor: wire filesystem and git adapters in cmd/ layer"
```

---

### Task 11: Update Tests

**Files:**
- Modify: `cmd/helpers_test.go`
- Modify: `cmd/working_copy_paths_test.go`
- Modify: `internal/app/workspace/git_repo_test.go`
- Modify: `internal/app/workspace/manifest_test.go`
- Modify: `internal/app/workspace/upstream_test.go`
- Modify: `internal/app/workspace/writer_test.go`

- [ ] **Step 1: Update cmd/ tests — SaveManifest calls**

In `helpers_test.go` and `working_copy_paths_test.go`, add import:
```go
"github.com/jakeraft/clier/internal/adapter/filesystem"
```

Replace all `appworkspace.SaveManifest(base, ...)` with:
```go
appworkspace.SaveManifest(filesystem.New(), base, ...)
```

- [ ] **Step 2: Update workspace tests — git_repo_test.go**

Update `ensureRepoDir` calls to pass `fs` and `git` params:
```go
func TestEnsureRepoDir_WithoutRepoURLCreatesDirectory(t *testing.T) {
	t.Parallel()
	fs := filesystem.New()
	git := adaptergit.New()

	repoDir := filepath.Join(t.TempDir(), "clier_todo")
	if err := ensureRepoDir(fs, git, "", repoDir); err != nil {
		t.Fatalf("ensureRepoDir: %v", err)
	}
	// ...
}
```

Add imports:
```go
"github.com/jakeraft/clier/internal/adapter/filesystem"
adaptergit "github.com/jakeraft/clier/internal/adapter/git"
```

Update all test functions similarly. Remove `"os/exec"` import from this test file if it's no longer needed (the `newTestRepo` and `runGit` helpers may still use it — keep if so).

- [ ] **Step 3: Update workspace tests — manifest_test.go, upstream_test.go, writer_test.go**

Add `filesystem.New()` as first param to any `SaveManifest`, `LoadManifest`, `FindManifestAbove`, `WriteMemberProjection`, `LoadMemberProjection` calls in these test files. Check each file and update accordingly.

- [ ] **Step 4: Run all tests**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: update tests for adapter injection"
```

---

### Task 12: Verify No os/os/exec Imports in Workspace Package

**Files:** (read-only verification)

- [ ] **Step 1: Check imports**

Run: `cd /Users/jake_kakao/jakeraft/clier && grep -rn '"os"' internal/app/workspace/*.go | grep -v _test.go`

Expected: only `os.IsNotExist` / `os.ErrNotExist` error checks (manifest.go, git_repo.go, upstream.go) and `os.UserHomeDir()` (writer.go) remain. No `os.ReadFile`, `os.WriteFile`, `os.MkdirAll`, `os.Stat`, `os.ReadDir` calls.

Run: `cd /Users/jake_kakao/jakeraft/clier && grep -rn '"os/exec"' internal/app/workspace/*.go | grep -v _test.go`

Expected: zero matches.

- [ ] **Step 2: Run full test suite**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./...`
Expected: all PASS

- [ ] **Step 3: Build**

Run: `cd /Users/jake_kakao/jakeraft/clier && go build ./...`
Expected: success

- [ ] **Step 4: Final commit if any cleanup was needed**

```bash
git add -A
git commit -m "refactor: complete clone adapter extraction (closes #38)"
```
