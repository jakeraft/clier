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
	EnvironmentIDs  []string  `json:"environment_ids"`
	GitRepoID       string    `json:"git_repo_id"` // empty string means not set
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func NewMember(name, cliProfileID string, systemPromptIDs, environmentIDs []string, gitRepoID string) (*Member, error) {
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
	if environmentIDs == nil {
		environmentIDs = []string{}
	}

	now := time.Now()
	return &Member{
		ID:              uuid.NewString(),
		Name:            name,
		CliProfileID:    cliProfileID,
		SystemPromptIDs: systemPromptIDs,
		EnvironmentIDs:  environmentIDs,
		GitRepoID:       gitRepoID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (m *Member) Update(name, cliProfileID *string, systemPromptIDs, environmentIDs *[]string, gitRepoID *string) error {
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
	if environmentIDs != nil {
		m.EnvironmentIDs = *environmentIDs
	}
	if gitRepoID != nil {
		m.GitRepoID = *gitRepoID
	}
	m.UpdatedAt = time.Now()
	return nil
}
