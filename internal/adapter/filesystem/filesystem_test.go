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
