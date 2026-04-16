package cmd

import (
	"strings"
	"testing"
)

func TestParseChildRefSpecs(t *testing.T) {
	t.Parallel()

	got, err := parseChildRefSpecs([]string{"alice/worker@3", "bob/runner@5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Owner != "alice" || got[0].Name != "worker" || got[0].ChildVersion != 3 {
		t.Fatalf("first child = %+v", got[0])
	}
}

func TestParseChildRefSpecs_EmptyRefReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parseChildRefSpecs([]string{""})
	if err == nil {
		t.Fatal("expected error for empty child ref")
	}
	if !strings.Contains(err.Error(), "child ref must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}
