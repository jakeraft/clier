package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/domain/resource"
)

type Member struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	AgentType        string    `json:"agent_type"`
	Command          string    `json:"command"`
	ClaudeMdID       *int64    `json:"claude_md_id"` // nil = not set (nullable FK)
	SkillIDs         []int64   `json:"skill_ids"`
	ClaudeSettingsID *int64    `json:"claude_settings_id"` // nil = not set (nullable FK)
	GitRepoURL       string    `json:"git_repo_url"`       // empty string = no repo
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func NewMember(name, agentType, command string,
	claudeMdID *int64, skillIDs []int64,
	claudeSettingsID *int64,
	gitRepoURL string) (*Member, error) {

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("member name must not be empty")
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, errors.New("member command must not be empty")
	}
	if skillIDs == nil {
		skillIDs = []int64{}
	}

	now := time.Now()
	return &Member{
		Name:             name,
		AgentType:        agentType,
		Command:          command,
		ClaudeMdID:       claudeMdID,
		SkillIDs:         skillIDs,
		ClaudeSettingsID: claudeSettingsID,
		GitRepoURL:       gitRepoURL,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (m *Member) Update(name, agentType, command *string,
	claudeMdID **int64, skillIDs *[]int64,
	claudeSettingsID **int64,
	gitRepoURL *string) error {

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("member name must not be empty")
		}
		m.Name = trimmed
	}
	if agentType != nil {
		m.AgentType = *agentType
	}
	if command != nil {
		trimmed := strings.TrimSpace(*command)
		if trimmed == "" {
			return errors.New("member command must not be empty")
		}
		m.Command = trimmed
	}
	if claudeMdID != nil {
		m.ClaudeMdID = *claudeMdID
	}
	if skillIDs != nil {
		m.SkillIDs = *skillIDs
	}
	if claudeSettingsID != nil {
		m.ClaudeSettingsID = *claudeSettingsID
	}
	if gitRepoURL != nil {
		m.GitRepoURL = *gitRepoURL
	}
	m.UpdatedAt = time.Now()
	return nil
}

// ResolvedMember is a Member spec with all referenced resources loaded.
// Produced by the resolve phase; consumed by the build phase to create MemberPlan.
type ResolvedMember struct {
	TeamMemberID   int64
	Name           string
	Command        string
	ClaudeMd       *resource.ClaudeMd
	Skills         []resource.Skill
	ClaudeSettings *resource.ClaudeSettings
	GitRepoURL     string
	Relations      MemberRelations
}
