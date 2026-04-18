package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandsDoNotConstructAPIClientsOutsideRoot(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob cmd files: %v", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || file == "root.go" {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(data), "newAPIClient(") {
			t.Fatalf("%s should not construct API clients directly", file)
		}
	}
}

func TestCommandsDoNotBuildSuccessPayloadsWithRawMaps(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob cmd files: %v", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(data), "present.Success(cmd.OutOrStdout(), map[") {
			t.Fatalf("%s should route success payloads through cmd/view, not raw maps", file)
		}
	}
}

func TestCommandsDoNotImportStorePackages(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob cmd files: %v", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(data), `"github.com/jakeraft/clier/internal/store/`) {
			t.Fatalf("%s should depend on app boundaries, not store packages", file)
		}
	}
}
