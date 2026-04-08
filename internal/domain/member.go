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
	Command          string    `json:"command"`
	ClaudeMdID       string    `json:"claude_md_id"`        // empty string = not set (nullable FK)
	SkillIDs         []string  `json:"skill_ids"`
	ClaudeSettingsID string    `json:"claude_settings_id"`  // empty string = not set (nullable FK)
	GitRepoURL       string    `json:"git_repo_url"`        // empty string = no repo
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func NewMember(name, command string,
	claudeMdID string, skillIDs []string,
	claudeSettingsID string,
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
		skillIDs = []string{}
	}

	now := time.Now()
	return &Member{
		ID:               uuid.NewString(),
		Name:             name,
		Command:          command,
		ClaudeMdID:       claudeMdID,
		SkillIDs:         skillIDs,
		ClaudeSettingsID: claudeSettingsID,
		GitRepoURL:       gitRepoURL,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (m *Member) Update(name, command *string,
	claudeMdID *string, skillIDs *[]string,
	claudeSettingsID *string,
	gitRepoURL *string) error {

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("member name must not be empty")
		}
		m.Name = trimmed
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
	TeamMemberID   string
	Name           string
	Command        string
	ClaudeMd       *resource.ClaudeMd
	Skills         []resource.Skill
	ClaudeSettings *resource.ClaudeSettings
	GitRepoURL     string
	Relations      MemberRelations
}
