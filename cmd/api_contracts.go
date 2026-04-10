package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/api"
)

func parseOptionalInt64(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid int64 %q: %w", raw, err)
	}
	return &v, nil
}

func parseOptionalResourceRefRequest(raw string) (*api.ResourceRefRequest, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.SplitN(raw, "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resource ref %q: want <id>@<version>", raw)
	}
	id, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid resource id in %q: %w", raw, err)
	}
	version, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid resource version in %q: %w", raw, err)
	}
	return &api.ResourceRefRequest{ID: id, Version: version}, nil
}

func resourceRefRequests(refs []api.ResourceRef) []api.ResourceRefRequest {
	if len(refs) == 0 {
		return nil
	}
	requests := make([]api.ResourceRefRequest, 0, len(refs))
	for _, ref := range refs {
		requests = append(requests, api.ResourceRefRequest{ID: ref.ID, Version: ref.Version})
	}
	return requests
}

func parseTeamMemberSpecs(specs []string) ([]api.TeamMemberRequest, error) {
	members := make([]api.TeamMemberRequest, 0, len(specs))
	for _, spec := range specs {
		parts := strings.SplitN(spec, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --member %q, want <member-id>@<version>:<name>", spec)
		}
		memberRef, err := parseOptionalResourceRefRequest(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid member ref in %q: %w", spec, err)
		}
		name := strings.TrimSpace(parts[1])
		if name == "" {
			return nil, fmt.Errorf("invalid --member %q, name must not be empty", spec)
		}
		members = append(members, api.TeamMemberRequest{
			Member: api.MemberRefRequest{
				ID:      memberRef.ID,
				Version: memberRef.Version,
			},
			Name: name,
		})
	}
	return members, nil
}

func parseTeamRelationSpecs(specs []string) ([]api.TeamRelationRequest, error) {
	relations := make([]api.TeamRelationRequest, 0, len(specs))
	for _, spec := range specs {
		parts := strings.SplitN(spec, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --relation %q, want <from-index>:<to-index>", spec)
		}
		fromIndex, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid from-index in %q: %w", spec, err)
		}
		toIndex, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid to-index in %q: %w", spec, err)
		}
		relations = append(relations, api.TeamRelationRequest{
			FromIndex: fromIndex,
			ToIndex:   toIndex,
		})
	}
	return relations, nil
}

func teamMutationRequestFromResponse(team *api.TeamResponse) (*api.TeamMutationRequest, error) {
	memberIndexByID := make(map[int64]int, len(team.TeamMembers))
	members := make([]api.TeamMemberRequest, 0, len(team.TeamMembers))
	for i, tm := range team.TeamMembers {
		memberIndexByID[tm.ID] = i
		members = append(members, api.TeamMemberRequest{
			Member: api.MemberRefRequest{
				ID:      tm.Member.ID,
				Version: tm.Member.Version,
			},
			Name: tm.Name,
		})
	}

	relations := make([]api.TeamRelationRequest, 0, len(team.Relations))
	for _, rel := range team.Relations {
		fromIndex, ok := memberIndexByID[rel.FromTeamMemberID]
		if !ok {
			return nil, fmt.Errorf("team relation references unknown from_team_member_id %d", rel.FromTeamMemberID)
		}
		toIndex, ok := memberIndexByID[rel.ToTeamMemberID]
		if !ok {
			return nil, fmt.Errorf("team relation references unknown to_team_member_id %d", rel.ToTeamMemberID)
		}
		relations = append(relations, api.TeamRelationRequest{
			FromIndex: fromIndex,
			ToIndex:   toIndex,
		})
	}

	var rootIndex *int
	if team.RootTeamMemberID != nil {
		idx, ok := memberIndexByID[*team.RootTeamMemberID]
		if !ok {
			return nil, fmt.Errorf("root_team_member_id %d not found in team_members", *team.RootTeamMemberID)
		}
		rootIndex = &idx
	}

	return &api.TeamMutationRequest{
		Name:        team.Name,
		TeamMembers: members,
		Relations:   relations,
		RootIndex:   rootIndex,
	}, nil
}
