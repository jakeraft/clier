package api

import (
	"encoding/json"
	"fmt"

	domainworkspace "github.com/jakeraft/clier/internal/domain/workspace"
)

type snapshotRef struct {
	RelType       string `json:"rel_type"`
	TargetName    string `json:"target_name"`
	TargetOwner   string `json:"target_owner"`
	TargetVersion int    `json:"target_version"`
}

type snapshotWithRefs struct {
	Refs []snapshotRef `json:"refs"`
}

func IsAbstractTeamAgentType(agentType string) bool {
	return agentType == "manager"
}

func decodeSnapshot[T any](snapshot json.RawMessage) (*T, error) {
	var s T
	return &s, json.Unmarshal(snapshot, &s)
}

func ContentFromResolved(r *ResolvedResource) (string, error) {
	spec, err := decodeSnapshot[ContentSpec](r.Snapshot)
	if err != nil {
		return "", fmt.Errorf("decode content snapshot for %s/%s: %w", r.OwnerName, r.Name, err)
	}
	return spec.Content, nil
}

func TeamProjectionFromResolved(r *ResolvedResource) (*domainworkspace.TeamProjection, error) {
	spec, err := decodeSnapshot[TeamSpec](r.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("decode team snapshot for %s/%s: %w", r.OwnerName, r.Name, err)
	}

	var refs snapshotWithRefs
	if err := json.Unmarshal(r.Snapshot, &refs); err != nil {
		return nil, fmt.Errorf("decode team refs for %s/%s: %w", r.OwnerName, r.Name, err)
	}

	projection := &domainworkspace.TeamProjection{
		Name:       r.Name,
		AgentType:  spec.AgentType,
		Command:    spec.Command,
		GitRepoURL: spec.GitRepoURL,
		Skills:     make([]domainworkspace.ResourceRef, 0),
		Children:   make([]domainworkspace.Child, 0, len(spec.Children)),
	}

	for _, child := range spec.Children {
		projection.Children = append(projection.Children, domainworkspace.Child{
			Owner:        child.Owner,
			Name:         child.Name,
			ChildVersion: child.Version,
		})
	}

	for _, ref := range refs.Refs {
		rp := domainworkspace.ResourceRef{
			Owner:   ref.TargetOwner,
			Name:    ref.TargetName,
			Version: ref.TargetVersion,
		}
		switch ref.RelType {
		case string(KindInstruction):
			projection.InstructionRef = &rp
		case string(KindClaudeSettings), string(KindCodexSettings):
			projection.SettingsRef = &rp
		case string(KindSkill):
			projection.Skills = append(projection.Skills, rp)
		}
	}

	return projection, nil
}
