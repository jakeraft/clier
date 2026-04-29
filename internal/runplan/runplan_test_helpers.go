package runplan

import (
	"os"
	"path/filepath"
)

// Test-only helpers kept in a non-_test.go file so they can be reused by
// any future cross-package tests without copying.

func mkdirParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
