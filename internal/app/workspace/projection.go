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

// TeamProjection is the unified projection for both leaf and composite teams.
// Leaf team: has Command/AgentType/refs, no Children.
// Composite team: has Children, may also have Command/AgentType.
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

// IsLeaf returns true if this team has no children (i.e. is a runnable agent).
func (p *TeamProjection) IsLeaf() bool {
	return len(p.Children) == 0
}

const TeamProjectionFile = "team.json"

func TeamProjectionPath(base string) string {
	return filepath.Join(base, ".clier", TeamProjectionFile)
}

func ChildTeamProjectionPath(base, childName string) string {
	return filepath.Join(base, ".clier", "teams", sanitizeRepoDirName(childName)+".json")
}

func TeamProjectionLocalPath() string {
	return filepath.ToSlash(filepath.Join(".clier", "team.json"))
}

func ChildTeamProjectionLocalPath(childName string) string {
	return filepath.ToSlash(filepath.Join(".clier", "teams", sanitizeRepoDirName(childName)+".json"))
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
