package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func TestReadContent_FromArgs(t *testing.T) {
	got, err := readContent([]string{"hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("got %q, want %q", got, "hello world")
	}
}

func TestReadContent_FromStdin(t *testing.T) {
	r, w, _ := os.Pipe()
	_, _ = w.WriteString("from stdin\n")
	_ = w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	got, err := readContent(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "from stdin" {
		t.Fatalf("got %q, want %q", got, "from stdin")
	}
}

func TestReadContent_DashMeansStdin(t *testing.T) {
	r, w, _ := os.Pipe()
	_, _ = w.WriteString("via dash\n")
	_ = w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	got, err := readContent([]string{"-"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "via dash" {
		t.Fatalf("got %q, want %q", got, "via dash")
	}
}

func TestReadContent_EmptyStdinError(t *testing.T) {
	r, w, _ := os.Pipe()
	_ = w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	_, err := readContent(nil)
	if err == nil {
		t.Fatal("expected error for empty stdin")
	}
}

func TestFirstRunHint_NilFirstRunAt_ReturnsHintAndMark(t *testing.T) {
	manifest := &appworkspace.Manifest{}

	hint := appworkspace.MarkFirstRun(manifest, "20260417T010203-deadbeef", func() time.Time {
		return time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)
	})

	if hint == "" {
		t.Fatal("expected non-empty hint when FirstRunAt is nil")
	}
	if !strings.Contains(hint, "20260417T010203-deadbeef") {
		t.Fatalf("hint should reference the runID, got: %s", hint)
	}
	if manifest.FirstRunAt == nil {
		t.Fatal("expected mark timestamp when FirstRunAt is nil")
	}
}

func TestFirstRunHint_AlreadyMarked_ReturnsEmpty(t *testing.T) {
	already := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	manifest := &appworkspace.Manifest{FirstRunAt: &already}

	hint := appworkspace.MarkFirstRun(manifest, "any-run-id", time.Now)

	if hint != "" {
		t.Fatalf("expected empty hint when FirstRunAt is set, got: %s", hint)
	}
	if !manifest.FirstRunAt.Equal(already) {
		t.Fatalf("manifest FirstRunAt mutated: got %v, want %v", manifest.FirstRunAt, already)
	}
}
