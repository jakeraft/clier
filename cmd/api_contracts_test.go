package cmd

import (
	"errors"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestParseOptionalResourceRefRequest_OrgOwner(t *testing.T) {
	t.Parallel()

	got, err := parseOptionalResourceRefRequest("@clier/hello-codex@7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("got nil ref, want parsed ref")
	}
	if got.Owner != "@clier" || got.Name != "hello-codex" || got.Version != 7 {
		t.Fatalf("parsed ref = %+v", *got)
	}
}

func TestParseOptionalResourceRefRequest_MissingVersionReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parseOptionalResourceRefRequest("@clier/hello-codex@")
	if err == nil {
		t.Fatal("expected error for missing version")
	}
	var f *domain.Fault
	if !errors.As(err, &f) || f.Kind != domain.KindInvalidResourceRef {
		t.Fatalf("expected KindInvalidResourceRef, got %v", err)
	}
}

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

func TestParseChildRefSpecs_OrgOwner(t *testing.T) {
	t.Parallel()

	got, err := parseChildRefSpecs([]string{"@clier/hello-codex@1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Owner != "@clier" || got[0].Name != "hello-codex" || got[0].ChildVersion != 1 {
		t.Fatalf("child = %+v", got[0])
	}
}

func TestParseChildRefSpecs_EmptyRefReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parseChildRefSpecs([]string{""})
	if err == nil {
		t.Fatal("expected error for empty child ref")
	}
	var f *domain.Fault
	if !errors.As(err, &f) || f.Kind != domain.KindInvalidArgument {
		t.Fatalf("expected KindInvalidArgument, got %v", err)
	}
}
