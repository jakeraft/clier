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
