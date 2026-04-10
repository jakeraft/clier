# Clone Adapter Extraction Design

> Issue: https://github.com/jakeraft/clier/issues/38

## Goal

Extract filesystem and git I/O from `internal/app/workspace/` into explicit adapter/port boundaries so the app layer orchestrates *what* to materialize while adapters handle *how* to interact with the local filesystem and git.

After this refactoring, the workspace package should have **zero** direct `"os"` or `"os/exec"` imports (test files excluded).

## Port Definitions

File: `internal/app/workspace/port.go`

```go
package workspace

import "io/fs"

// FileMaterializer abstracts local filesystem operations.
type FileMaterializer interface {
    WriteFile(path string, content []byte) error
    ReadFile(path string) ([]byte, error)
    MkdirAll(path string) error
    Stat(path string) (fs.FileInfo, error)
    ReadDir(path string) ([]fs.DirEntry, error)
    MkdirTemp(pattern string) (string, error)
    RemoveAll(path string) error
}

// GitRepo abstracts git CLI operations.
type GitRepo interface {
    Clone(repoURL, targetDir string) error
    IsRepo(dir string) (bool, error)
    Origin(dir string) (string, error)
    Diff(pathA, pathB string) (string, bool, error)
}
```

Uses `io/fs` types instead of `os` types for clean abstraction.

## Adapter Implementations

### `internal/adapter/filesystem/filesystem.go`

`LocalFS` struct — thin wrapper around `os.*`:

| Method | Delegates to |
|--------|-------------|
| `WriteFile` | `os.MkdirAll(dir) + os.WriteFile(path, content, 0644)` |
| `ReadFile` | `os.ReadFile` |
| `MkdirAll` | `os.MkdirAll(path, 0755)` |
| `Stat` | `os.Stat` |
| `ReadDir` | `os.ReadDir` |
| `MkdirTemp` | `os.MkdirTemp("", pattern)` |
| `RemoveAll` | `os.RemoveAll` |

Note: `WriteFile` auto-creates parent directories. This eliminates the separate `MkdirAll` + `WriteFile` pattern currently used by `writeFile()` in writer.go.

### `internal/adapter/git/git.go`

`ExecGit` struct — wraps `exec.Command("git", ...)`:

| Method | Delegates to |
|--------|-------------|
| `Clone` | `git clone --depth 1 <url> <dir>` (with parent dir creation via `os.MkdirAll`) |
| `IsRepo` | `git rev-parse --show-toplevel` + path comparison |
| `Origin` | `git config --get remote.origin.url` |
| `Diff` | `git diff --no-index --no-color -- <pathA> <pathB>` |

Logic moved from current `git_repo.go` (`gitClone`, `isGitRepo`, `gitOrigin`) and `upstream.go` (`renderProjectionDiff`'s git diff call).

## Service Changes

### Struct & Constructor

```go
type Service struct {
    client *api.Client
    fs     FileMaterializer
    git    GitRepo
}

func NewService(client *api.Client, fs FileMaterializer, git GitRepo) *Service
```

### Standalone functions → Service methods

These functions are only called from Service methods, so they become methods with access to `s.fs`:

| Current (standalone) | After (Service method) |
|----------------------|----------------------|
| `populateBaseHashes(base, tracked)` | `s.populateBaseHashes(base, tracked)` |
| `fileHash(path)` | `s.fileHash(path)` — uses `s.fs.ReadFile` + `sha256` |
| `trackedStatuses(base, manifest)` | `s.trackedStatuses(base, manifest)` |
| `runSummary(base)` | `s.runSummary(base)` — uses `s.fs.ReadDir` |
| `ModifiedTrackedResources(base)` | `s.ModifiedTrackedResources(base)` |
| `serverClaudeMdContent(base, manifest, resource)` | `s.serverClaudeMdContent(base, manifest, resource)` |

### Direct `os.*` replacements in Push

`Push` method reads local files for claude-settings and skill resources (`os.ReadFile` at lines 267, 285). These become `s.fs.ReadFile`.

## Writer Changes

### Struct & Constructor

```go
type Writer struct {
    client *api.Client
    owner  string
    fs     FileMaterializer
    git    GitRepo
}

func NewWriter(client *api.Client, owner string, fs FileMaterializer, git GitRepo) *Writer
```

Service creates Writer with its own fs/git:

```go
writer := NewWriter(s.client, owner, s.fs, s.git)
```

### Standalone functions → Writer methods

| Current (standalone) | After (Writer method) |
|----------------------|----------------------|
| `writeFile(path, content)` | `w.writeFile(path, content)` — uses `w.fs.WriteFile` |
| `writeLocalSettings(base, profile)` | `w.writeLocalSettings(base, profile)` |
| `writeWorkLogProtocol(base)` | `w.writeWorkLogProtocol(base)` |

`localSettingsContent(profile)` stays standalone — it only uses `os.UserHomeDir()` which is an environment query, not file I/O.

## git_repo.go Changes

| Function | Change |
|----------|--------|
| `gitClone` | Deleted — logic moves to `adapter/git/ExecGit.Clone` |
| `isGitRepo` | Deleted — logic moves to `adapter/git/ExecGit.IsRepo` |
| `gitOrigin` | Deleted — logic moves to `adapter/git/ExecGit.Origin` |
| `ensureRepoDir(repoURL, repoDir)` | Becomes Writer method `w.ensureRepoDir` — uses `w.fs` + `w.git` |
| `IsMaterializedRoot(repoURL, root)` | Signature: `IsMaterializedRoot(fs FileMaterializer, git GitRepo, repoURL, root string)` |

## manifest.go Changes

Public functions get `fs FileMaterializer` as first parameter:

```go
func SaveManifest(fs FileMaterializer, base string, manifest *Manifest) error
func LoadManifest(fs FileMaterializer, base string) (*Manifest, error)
func FindManifestPath(fs FileMaterializer, base string) (string, error)
func FindManifestAbove(fs FileMaterializer, start string) (string, *Manifest, error)
```

Called from both Service methods (with `s.fs`) and cmd/ (with concrete adapter instance).

## projection.go Changes

Public functions get `fs FileMaterializer` as first parameter:

```go
func WriteMemberProjection(fs FileMaterializer, path string, projection *MemberProjection) error
func WriteTeamProjection(fs FileMaterializer, path string, projection *TeamProjection) error
func LoadMemberProjection(fs FileMaterializer, path string) (*MemberProjection, error)
func LoadTeamProjection(fs FileMaterializer, path string) (*TeamProjection, error)
```

Internal `writeJSONProjection` and `loadJSONProjection` also take `fs` parameter.

## upstream.go Changes

- `FetchUpstream`: `LoadManifest`/`SaveManifest`/`WriteProjection` calls pass `s.fs`
- `MergeFetchedUpstream`: direct `os.ReadFile`/`os.WriteFile` → `s.fs.ReadFile`/`s.fs.WriteFile`
- `renderProjectionDiff`: fully migrated:
  - `os.Stat` → `s.fs.Stat`
  - `os.MkdirTemp` → `s.fs.MkdirTemp`
  - `os.RemoveAll` → `s.fs.RemoveAll`
  - `copyFile` → `s.fs.ReadFile` + `s.fs.WriteFile`
  - `exec.Command("git", "diff", ...)` → `s.git.Diff`
- `renderProjectionDiff` becomes Service method `s.renderProjectionDiff`
- `copyFile` standalone function deleted (inlined via fs adapter calls)

## cmd/ Wiring Changes

### Adapter creation helpers

```go
// cmd/helpers.go (or similar)
func newFileMaterializer() *filesystem.LocalFS { return filesystem.New() }
func newGitRepo() *git.ExecGit { return git.New() }
```

### NewService calls (7 files)

```go
// clone.go, status.go, push.go, diff.go, fetch.go, merge.go, pull.go
svc := appworkspace.NewService(newAPIClient(), newFileMaterializer(), newGitRepo())
```

### Standalone function calls

```go
// FindManifestAbove — helpers.go, status.go, diff.go, push.go, fetch.go, merge.go, pull.go, working_copy_paths.go
fs := newFileMaterializer()
copyRoot, manifest, err := appworkspace.FindManifestAbove(fs, base)

// IsMaterializedRoot — working_copy_validation.go
fs := newFileMaterializer()
gitRepo := newGitRepo()
materialized, err := appworkspace.IsMaterializedRoot(fs, gitRepo, member.GitRepoURL, base)

// LoadMemberProjection — run.go
fs := newFileMaterializer()
memberProjection, err := appworkspace.LoadMemberProjection(fs, path)
```

### Test files

`cmd/working_copy_paths_test.go`, `cmd/helpers_test.go`: `SaveManifest` calls get `filesystem.New()` as first arg.

## File Change Summary

| Category | File | Change |
|----------|------|--------|
| New | `internal/app/workspace/port.go` | Interface definitions |
| New | `internal/adapter/filesystem/filesystem.go` | LocalFS implementation |
| New | `internal/adapter/git/git.go` | ExecGit implementation |
| Modify | `internal/app/workspace/service.go` | fs/git injection, standalone→method |
| Modify | `internal/app/workspace/writer.go` | fs/git injection, standalone→method |
| Modify | `internal/app/workspace/manifest.go` | fs parameter added |
| Modify | `internal/app/workspace/projection.go` | fs parameter added |
| Modify | `internal/app/workspace/git_repo.go` | Git logic→adapter, orchestration remains |
| Modify | `internal/app/workspace/upstream.go` | s.fs/s.git usage, renderProjectionDiff migration |
| Modify | `cmd/clone.go` | Wiring |
| Modify | `cmd/status.go` | Wiring |
| Modify | `cmd/push.go` | Wiring |
| Modify | `cmd/diff.go` | Wiring |
| Modify | `cmd/fetch.go` | Wiring |
| Modify | `cmd/merge.go` | Wiring |
| Modify | `cmd/pull.go` | Wiring |
| Modify | `cmd/helpers.go` | Adapter helpers + FindManifestAbove |
| Modify | `cmd/run.go` | LoadMemberProjection |
| Modify | `cmd/working_copy_validation.go` | IsMaterializedRoot |
| Modify | `cmd/working_copy_paths.go` | FindManifestAbove |
| Modify | `cmd/` test files | SaveManifest signature |

## Out of Scope

- `os.UserHomeDir()` in writer.go — environment query, not file I/O
- API client abstraction — already in `adapter/api/`
