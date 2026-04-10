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
