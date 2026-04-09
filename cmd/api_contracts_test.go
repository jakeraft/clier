package cmd

import (
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
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

	got, err := parseTeamMemberSpecs([]string{"101:lead", "202:worker"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].MemberID != 101 || got[0].Name != "lead" {
		t.Fatalf("first member = %+v", got[0])
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

func TestTeamMutationRequestFromResponse(t *testing.T) {
	t.Parallel()

	rootID := int64(11)
	team := &api.TeamResponse{
		Name:             "dev-squad",
		RootTeamMemberID: &rootID,
		TeamMembers: []api.TeamMemberResponse{
			{ID: 11, Name: "lead", Member: api.MemberRef{ResourceRef: api.ResourceRef{ID: 101}}},
			{ID: 22, Name: "worker", Member: api.MemberRef{ResourceRef: api.ResourceRef{ID: 202}}},
		},
		Relations: []api.TeamRelationResponse{
			{FromTeamMemberID: 11, ToTeamMemberID: 22},
		},
	}

	got, err := teamMutationRequestFromResponse(team)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "dev-squad" {
		t.Fatalf("Name = %q, want dev-squad", got.Name)
	}
	if got.RootIndex == nil || *got.RootIndex != 0 {
		t.Fatalf("RootIndex = %v, want 0", got.RootIndex)
	}
	if len(got.TeamMembers) != 2 || got.TeamMembers[1].MemberID != 202 {
		t.Fatalf("TeamMembers = %+v", got.TeamMembers)
	}
	if len(got.Relations) != 1 || got.Relations[0].FromIndex != 0 || got.Relations[0].ToIndex != 1 {
		t.Fatalf("Relations = %+v", got.Relations)
	}
}
