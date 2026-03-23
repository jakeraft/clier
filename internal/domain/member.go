package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Member struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	CliProfileID    string    `json:"cli_profile_id"`
	SystemPromptIDs []string  `json:"system_prompt_ids"`
	EnvIDs          []string  `json:"env_ids"`
	GitRepoID       string    `json:"git_repo_id"` // empty string means not set
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func NewMember(name, cliProfileID string, systemPromptIDs []string, gitRepoID string, envIDs []string) (*Member, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("member name must not be empty")
	}
	cliProfileID = strings.TrimSpace(cliProfileID)
	if cliProfileID == "" {
		return nil, errors.New("member cli profile id must not be empty")
	}

	if systemPromptIDs == nil {
		systemPromptIDs = []string{}
	}
	if envIDs == nil {
		envIDs = []string{}
	}

	now := time.Now()
	return &Member{
		ID:              uuid.NewString(),
		Name:            name,
		CliProfileID:    cliProfileID,
		SystemPromptIDs: systemPromptIDs,
		EnvIDs:          envIDs,
		GitRepoID:       gitRepoID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (m *Member) Update(name, cliProfileID *string, systemPromptIDs *[]string, gitRepoID *string, envIDs *[]string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("member name must not be empty")
		}
		m.Name = trimmed
	}
	if cliProfileID != nil {
		trimmed := strings.TrimSpace(*cliProfileID)
		if trimmed == "" {
			return errors.New("member cli profile id must not be empty")
		}
		m.CliProfileID = trimmed
	}
	if systemPromptIDs != nil {
		m.SystemPromptIDs = *systemPromptIDs
	}
	if gitRepoID != nil {
		m.GitRepoID = *gitRepoID
	}
	if envIDs != nil {
		m.EnvIDs = *envIDs
	}
	m.UpdatedAt = time.Now()
	return nil
}
