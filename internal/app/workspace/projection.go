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

type MemberProjection struct {
	Name           string                  `json:"name"`
	Command        string                  `json:"command"`
	GitRepoURL     string                  `json:"git_repo_url,omitempty"`
	ClaudeMd       *ResourceRefProjection  `json:"claude_md,omitempty"`
	ClaudeSettings *ResourceRefProjection  `json:"claude_settings,omitempty"`
	Skills         []ResourceRefProjection `json:"skills,omitempty"`
}

type TeamProjection struct {
	Name      string                   `json:"name"`
	Members   []TeamMemberProjection   `json:"members"`
	Relations []TeamRelationProjection `json:"relations,omitempty"`
}

type TeamMemberProjection struct {
	MemberID      int64                 `json:"member_id"`
	MemberVersion int                   `json:"member_version"`
	Name          string                `json:"name"`
	Member        ResourceRefProjection `json:"member"`
}

type TeamRelationProjection struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

const TeamProjectionFile = "team.json"

func TeamProjectionPath(base string) string {
	return filepath.Join(base, ".clier", TeamProjectionFile)
}

func TeamMemberProjectionPath(base, memberName string) string {
	return filepath.Join(base, ".clier", "members", sanitizeRepoDirName(memberName)+".json")
}

func TeamProjectionLocalPath() string {
	return filepath.ToSlash(filepath.Join(".clier", "team.json"))
}

func TeamMemberProjectionLocalPath(memberName string) string {
	return filepath.ToSlash(filepath.Join(".clier", "members", sanitizeRepoDirName(memberName)+".json"))
}

func WriteMemberProjection(fs FileMaterializer, path string, projection *MemberProjection) error {
	return writeJSONProjection(fs, path, projection)
}

func WriteTeamProjection(fs FileMaterializer, path string, projection *TeamProjection) error {
	return writeJSONProjection(fs, path, projection)
}

func LoadMemberProjection(fs FileMaterializer, path string) (*MemberProjection, error) {
	var projection MemberProjection
	if err := loadJSONProjection(fs, path, &projection); err != nil {
		return nil, err
	}
	return &projection, nil
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
