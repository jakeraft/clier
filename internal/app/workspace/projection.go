package workspace

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

type ResourceRefProjection struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

// TeamProjection describes a team. Command/AgentType/refs apply when the
// team itself runs as an agent; Children references nested teams.
type TeamProjection struct {
	Name           string                  `json:"name"`
	AgentType      string                  `json:"agent_type,omitempty"`
	Command        string                  `json:"command,omitempty"`
	GitRepoURL     string                  `json:"git_repo_url,omitempty"`
	InstructionRef *ResourceRefProjection  `json:"instruction_ref,omitempty"`
	SettingsRef    *ResourceRefProjection  `json:"settings_ref,omitempty"`
	Skills         []ResourceRefProjection `json:"skills,omitempty"`
	Children       []ChildProjection       `json:"children,omitempty"`
}

// ChildProjection is a reference to a child team at a pinned version.
type ChildProjection struct {
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	ChildVersion int    `json:"child_version"`
}

const TeamProjectionFile = "team.json"

func TeamProjectionPath(base string) string {
	return filepath.Join(base, ".clier", TeamProjectionFile)
}

func TeamProjectionLocalPath() string {
	return filepath.ToSlash(filepath.Join(".clier", "team.json"))
}

func WriteTeamProjection(fs FileMaterializer, path string, projection *TeamProjection) error {
	return writeJSONProjection(fs, path, projection)
}

func LoadTeamProjection(fs FileMaterializer, path string) (*TeamProjection, error) {
	var projection TeamProjection
	if err := loadJSONProjection(fs, path, &projection); err != nil {
		return nil, err
	}
	return &projection, nil
}

func writeJSONProjection(fs FileMaterializer, path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal projection: %w", err)
	}
	if err := fs.EnsureFile(path, data); err != nil {
		return fmt.Errorf("write projection: %w", err)
	}
	return nil
}

func loadJSONProjection(fs FileMaterializer, path string, payload any) error {
	data, err := fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read projection: %w", err)
	}
	if err := json.Unmarshal(data, payload); err != nil {
		return fmt.Errorf("unmarshal projection: %w", err)
	}
	return nil
}
