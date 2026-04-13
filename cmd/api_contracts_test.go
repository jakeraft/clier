package cmd

import (
	"strings"
	"testing"
)

func TestParseTeamMemberSpecs(t *testing.T) {
	t.Parallel()

	got, err := parseTeamMemberSpecs([]string{"101@3", "202@5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].MemberID != 101 || got[0].MemberVersion != 3 {
		t.Fatalf("first member = %+v", got[0])
	}
}

func TestParseTeamMemberSpecs_EmptyMemberRefReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parseTeamMemberSpecs([]string{""})
	if err == nil {
		t.Fatal("expected error for empty member ref")
	}
	if !strings.Contains(err.Error(), "member ref must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseTeamRelationSpecs(t *testing.T) {
	t.Parallel()

	got, err := parseTeamRelationSpecs([]string{"100:200", "200:300"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[1].From != 200 || got[1].To != 300 {
		t.Fatalf("second relation = %+v", got[1])
	}
}
