package workspace

import (
	"encoding/json"
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
)

type versionedContentSnapshot struct {
	Content string `json:"content"`
}

type memberSnapshot struct {
	AgentType      string                  `json:"agent_type"`
	Command        string                  `json:"command"`
	GitRepoURL     string                  `json:"git_repo_url"`
	ClaudeMd       *ResourceRefProjection  `json:"claude_md,omitempty"`
	ClaudeSettings *ResourceRefProjection  `json:"claude_settings,omitempty"`
	Skills         []ResourceRefProjection `json:"skills,omitempty"`
}

func loadVersionedContent(raw json.RawMessage) (string, error) {
	var snapshot versionedContentSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return "", fmt.Errorf("unmarshal versioned content: %w", err)
	}
	return snapshot.Content, nil
}

func loadMemberSnapshot(raw json.RawMessage) (*memberSnapshot, error) {
	var snapshot memberSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal member snapshot: %w", err)
	}
	return &snapshot, nil
}

func memberResponseFromSnapshot(owner, name string, version int, snapshot *memberSnapshot) *api.MemberResponse {
	latestVersion := version
	response := &api.MemberResponse{
		Name:          name,
		OwnerLogin:    owner,
		LatestVersion: &latestVersion,
		AgentType:     snapshot.AgentType,
		Command:       snapshot.Command,
		GitRepoURL:    snapshot.GitRepoURL,
		Skills:        make([]api.ResourceRef, 0, len(snapshot.Skills)),
	}
	if snapshot.ClaudeMd != nil {
		response.ClaudeMd = &api.ResourceRef{
			Owner:   snapshot.ClaudeMd.Owner,
			Name:    snapshot.ClaudeMd.Name,
			Version: snapshot.ClaudeMd.Version,
		}
	}
	if snapshot.ClaudeSettings != nil {
		response.ClaudeSettings = &api.ResourceRef{
			Owner:   snapshot.ClaudeSettings.Owner,
			Name:    snapshot.ClaudeSettings.Name,
			Version: snapshot.ClaudeSettings.Version,
		}
	}
	for _, skill := range snapshot.Skills {
		response.Skills = append(response.Skills, api.ResourceRef{
			Owner:   skill.Owner,
			Name:    skill.Name,
			Version: skill.Version,
		})
	}
	return response
}
