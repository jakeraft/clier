package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Member struct {
	ID              string
	Name            string
	CliProfileID    string
	SystemPromptIDs []string
	EnvironmentIDs  []string
	GitRepoID       string // empty string means not set
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewMember(name, cliProfileID string, systemPromptIDs, environmentIDs []string, gitRepoID string) (*Member, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("member name must not be empty")
	}
	cliProfileID = strings.TrimSpace(cliProfileID)
	if cliProfileID == "" {
		return nil, fmt.Errorf("member cli profile id must not be empty")
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
			return fmt.Errorf("member name must not be empty")
		}
		m.Name = trimmed
	}
	if cliProfileID != nil {
		trimmed := strings.TrimSpace(*cliProfileID)
		if trimmed == "" {
			return fmt.Errorf("member cli profile id must not be empty")
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
