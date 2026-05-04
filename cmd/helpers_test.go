package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRequireOneArg_missingMessageHumanizesLabel(t *testing.T) {
	cmd := &cobra.Command{Use: "view <run-id>"}
	err := requireOneArg("<run-id>")(cmd, nil)
	if err == nil {
		t.Fatal("missing arg should error")
	}
	got := err.Error()
	if !strings.Contains(got, "<run-id> is required") {
		t.Errorf("missing message should name the label, got %q", got)
	}
	if !strings.Contains(got, "Usage:") {
		t.Errorf("missing message should embed usage hint, got %q", got)
	}
}

func TestRequireOneArg_extraSurfacesCount(t *testing.T) {
	cmd := &cobra.Command{Use: "stop <run-id>"}
	err := requireOneArg("<run-id>")(cmd, []string{"a", "b"})
	if err == nil {
		t.Fatal("extra args should error")
	}
	if !strings.Contains(err.Error(), "got 2") {
		t.Errorf("extra-arg error should report the count, got %q", err.Error())
	}
}

func TestRequireOneArg_oneArgPasses(t *testing.T) {
	cmd := &cobra.Command{Use: "view <run-id>"}
	if err := requireOneArg("<run-id>")(cmd, []string{"x"}); err != nil {
		t.Errorf("single arg should pass, got %v", err)
	}
}

func TestReadContent_emptyArgRejected(t *testing.T) {
	if _, err := readContent([]string{""}); err == nil {
		t.Fatal("empty arg must reject before any downstream lookup")
	}
}

func TestReadContent_whitespaceOnlyArgRejected(t *testing.T) {
	if _, err := readContent([]string{"   \t\n  "}); err == nil {
		t.Fatal("whitespace-only arg must reject (same contract as stdin path)")
	}
}

func TestReadContent_nonEmptyArgPasses(t *testing.T) {
	got, err := readContent([]string{"hello"})
	if err != nil {
		t.Fatalf("non-empty arg should pass: %v", err)
	}
	if got != "hello" {
		t.Errorf("content: got %q want %q", got, "hello")
	}
}
