package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/api"
)

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

func parseTeamMemberSpecs(specs []string) ([]api.TeamMemberRequest, error) {
	members := make([]api.TeamMemberRequest, 0, len(specs))
	for _, spec := range specs {
		ref, err := parseOptionalResourceRefRequest(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid --member %q: %w", spec, err)
		}
		if ref == nil {
			return nil, fmt.Errorf("invalid --member %q, member ref must not be empty", spec)
		}
		members = append(members, api.TeamMemberRequest{
			MemberID:      ref.ID,
			MemberVersion: ref.Version,
		})
	}
	return members, nil
}

func parseTeamRelationSpecs(specs []string) ([]api.TeamRelationRequest, error) {
	relations := make([]api.TeamRelationRequest, 0, len(specs))
	for _, spec := range specs {
		parts := strings.SplitN(spec, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --relation %q, want <from-member-id>:<to-member-id>", spec)
		}
		from, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid from in %q: %w", spec, err)
		}
		to, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid to in %q: %w", spec, err)
		}
		relations = append(relations, api.TeamRelationRequest{
			From: from,
			To:   to,
		})
	}
	return relations, nil
}
