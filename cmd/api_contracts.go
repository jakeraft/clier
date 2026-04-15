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
		return nil, fmt.Errorf("invalid resource ref %q: want <owner/name>@<version>", raw)
	}
	owner, name, err := parseOwnerName(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid resource ref %q: %w", raw, err)
	}
	version, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid resource version in %q: %w", raw, err)
	}
	return &api.ResourceRefRequest{Owner: owner, Name: name, Version: version}, nil
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
			Owner:         ref.Owner,
			Name:          ref.Name,
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
			return nil, fmt.Errorf("invalid --relation %q, want <owner/name>:<owner/name>", spec)
		}
		fromOwner, fromName, err := parseOwnerName(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid from in %q: %w", spec, err)
		}
		toOwner, toName, err := parseOwnerName(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid to in %q: %w", spec, err)
		}
		relations = append(relations, api.TeamRelationRequest{
			From: api.ResourceIdentifier{Owner: fromOwner, Name: fromName},
			To:   api.ResourceIdentifier{Owner: toOwner, Name: toName},
		})
	}
	return relations, nil
}
