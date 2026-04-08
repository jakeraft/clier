package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain/resource"
)

type Member struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	AgentType        string    `json:"agent_type"`
	Model            string    `json:"model"`
	Args             []string  `json:"args"`
	ClaudeMdID       string    `json:"claude_md_id"`        // empty string = not set (nullable FK)
	SkillIDs         []string  `json:"skill_ids"`
	ClaudeSettingsID string    `json:"claude_settings_id"`  // empty string = not set (nullable FK)
	ClaudeJsonID     string    `json:"claude_json_id"`      // empty string = not set (nullable FK)
	GitRepoURL       string    `json:"git_repo_url"`        // empty string = no repo
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func NewMember(name, agentType, model string, args []string,
	claudeMdID string, skillIDs []string,
	claudeSettingsID, claudeJsonID string,
	gitRepoURL string) (*Member, error) {

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("member name must not be empty")
	}
	agentType = strings.TrimSpace(agentType)
	if agentType == "" {
		agentType = "claude"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, errors.New("member model must not be empty")
	}
	if args == nil {
		args = []string{}
	}
	if skillIDs == nil {
		skillIDs = []string{}
	}

	now := time.Now()
	return &Member{
		ID:               uuid.NewString(),
		Name:             name,
		AgentType:        agentType,
		Model:            model,
		Args:             args,
		ClaudeMdID:       claudeMdID,
		SkillIDs:         skillIDs,
		ClaudeSettingsID: claudeSettingsID,
		ClaudeJsonID:     claudeJsonID,
		GitRepoURL:       gitRepoURL,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (m *Member) Update(name, agentType, model *string, args *[]string,
	claudeMdID *string, skillIDs *[]string,
	claudeSettingsID, claudeJsonID *string,
	gitRepoURL *string) error {

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("member name must not be empty")
		}
		m.Name = trimmed
	}
	if agentType != nil {
		trimmed := strings.TrimSpace(*agentType)
		if trimmed != "" {
			m.AgentType = trimmed
		}
	}
	if model != nil {
		trimmed := strings.TrimSpace(*model)
		if trimmed == "" {
			return errors.New("member model must not be empty")
		}
		m.Model = trimmed
	}
	if args != nil {
		m.Args = *args
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
	if claudeJsonID != nil {
		m.ClaudeJsonID = *claudeJsonID
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
	TeamMemberID   string
	Name           string
	AgentType      string
	Model          string
	Args           []string
	ClaudeMd       *resource.ClaudeMd
	Skills         []resource.Skill
	ClaudeSettings *resource.ClaudeSettings
	ClaudeJson     *resource.ClaudeJson
	GitRepoURL     string
	Relations      MemberRelations
}
