package cmd

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
)

func TestParseTeamMemberSpecs(t *testing.T) {
	t.Parallel()

	got, err := parseTeamMemberSpecs([]string{"alice/worker@3", "bob/runner@5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Owner != "alice" || got[0].Name != "worker" || got[0].MemberVersion != 3 {
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

	got, err := parseTeamRelationSpecs([]string{"alice/leader:bob/worker", "bob/worker:carol/runner"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	expected := api.TeamRelationRequest{
		From: api.ResourceIdentifier{Owner: "bob", Name: "worker"},
		To:   api.ResourceIdentifier{Owner: "carol", Name: "runner"},
	}
	if got[1].From != expected.From || got[1].To != expected.To {
		t.Fatalf("second relation = %+v", got[1])
	}
}
