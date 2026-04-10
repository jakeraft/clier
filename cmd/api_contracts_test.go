package cmd

import (
	"strings"
	"testing"
)

func TestParseOptionalInt64(t *testing.T) {
	t.Parallel()

	got, err := parseOptionalInt64("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("got %v, want nil", *got)
	}

	got, err = parseOptionalInt64("42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || *got != 42 {
		t.Fatalf("got %v, want 42", got)
	}
}

func TestParseTeamMemberSpecs(t *testing.T) {
	t.Parallel()

	got, err := parseTeamMemberSpecs([]string{"101@3:lead", "202@5:worker"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Member.ID != 101 || got[0].Member.Version != 3 || got[0].Name != "lead" {
		t.Fatalf("first member = %+v", got[0])
	}
}

func TestParseTeamMemberSpecs_EmptyMemberRefReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parseTeamMemberSpecs([]string{":lead"})
	if err == nil {
		t.Fatal("expected error for empty member ref")
	}
	if !strings.Contains(err.Error(), "member ref must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseTeamRelationSpecs(t *testing.T) {
	t.Parallel()

	got, err := parseTeamRelationSpecs([]string{"0:1", "1:2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[1].FromIndex != 1 || got[1].ToIndex != 2 {
		t.Fatalf("second relation = %+v", got[1])
	}
}

