package workspace

import (
	"io/fs"

	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
)

// FileMaterializer abstracts local filesystem operations used during
// clone, pull, push, and status workflows.
type FileMaterializer interface {
	// EnsureFile creates parent directories as needed and writes content to path.
	EnsureFile(path string, content []byte) error
	ReadFile(path string) ([]byte, error)
	MkdirAll(path string) error
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	MkdirTemp(pattern string) (string, error)
	Rename(oldPath, newPath string) error
	RemoveAll(path string) error
}

// GitRepo abstracts git CLI operations used during clone and diff workflows.
type GitRepo interface {
	Clone(repoURL, targetDir string) error
	IsRepo(dir string) (bool, error)
	Origin(dir string) (string, error)
	Diff(pathA, pathB string) (string, bool, error)
}

// RemoteWorkspaceClient is the thin-client boundary to clier-server for
// workspace orchestration. The workspace domain only depends on
// resource reads, team resolve, and remote writes needed to sync a
// local working copy.
type RemoteWorkspaceClient interface {
	GetResource(owner, name string) (*remoteapi.ResourceResponse, error)
	ResolveTeam(owner, name string) (*remoteapi.ResolveResponse, error)
	ResolveTeamVersion(owner, name string, version int) (*remoteapi.ResolveResponse, error)
	UpdateResource(kind remoteapi.ResourceKind, owner, name string, body any) (*remoteapi.ResourceResponse, error)
}
