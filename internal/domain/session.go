package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SessionStatus string

const (
	SessionRunning SessionStatus = "running"
	SessionStopped SessionStatus = "stopped"
)

// Session is an execution instance of a Team on a local machine.
// Plan is built fresh at session start from the team's current state.
type Session struct {
	ID     string        `json:"id"`
	TeamID string        `json:"team_id"`
	Status SessionStatus `json:"status"`
	// Plan retains {{CLIER_*}} placeholders as built. Safe for name/ID lookups;
	// paths and commands require resolution before use.
	Plan      []MemberPlan `json:"plan"`
	CreatedAt time.Time    `json:"created_at"`
	StoppedAt *time.Time   `json:"stopped_at"`
}

func NewSession(id, teamID string) (*Session, error) {
	if id == "" {
		return nil, errors.New("session id must not be empty")
	}
	if teamID == "" {
		return nil, errors.New("team id must not be empty")
	}
	return &Session{
		ID:        id,
		TeamID:    teamID,
		Status:    SessionRunning,
		CreatedAt: time.Now(),
	}, nil
}

func (s *Session) Stop() {
	now := time.Now()
	s.Status = SessionStopped
	s.StoppedAt = &now
}

// Message represents an inter-member message within a session.
// FromTeamMemberID is nullable — empty when the sender is not a team member.
type Message struct {
	ID               string    `json:"id"`
	SessionID        string    `json:"session_id"`
	FromTeamMemberID string    `json:"from_team_member_id"`
	ToTeamMemberID   string    `json:"to_team_member_id"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}

func NewMessage(sessionID, fromTeamMemberID, toTeamMemberID, content string) (*Message, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("message session id must not be empty")
	}
	if strings.TrimSpace(toTeamMemberID) == "" {
		return nil, errors.New("message recipient must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("message content must not be empty")
	}

	return &Message{
		ID:               uuid.NewString(),
		SessionID:        sessionID,
		FromTeamMemberID: fromTeamMemberID,
		ToTeamMemberID:   toTeamMemberID,
		Content:          content,
		CreatedAt:        time.Now(),
	}, nil
}

// Log is a self-recorded entry by a team member within a session.
type Log struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	TeamMemberID string    `json:"team_member_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewLog(sessionID, teamMemberID, content string) (*Log, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("log session id must not be empty")
	}
	if strings.TrimSpace(teamMemberID) == "" {
		return nil, errors.New("log team member id must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("log content must not be empty")
	}

	return &Log{
		ID:           uuid.NewString(),
		SessionID:    sessionID,
		TeamMemberID: teamMemberID,
		Content:      content,
		CreatedAt:    time.Now(),
	}, nil
}

// MemberPlan is a fully-resolved execution plan for a single team member.
// Binary, Model, Envs are NOT stored — they are already resolved into Command.
// Relations are NOT stored — they are in Team.Relations and baked into the prompt.
//
// Plan retains {{CLIER_*}} placeholders; these are expanded at session start
// into concrete paths. The stored plan is safe for name/ID lookups but should
// not be used to reconstruct the workspace without re-expanding placeholders.
type MemberPlan struct {
	TeamMemberID string        `json:"team_member_id"`
	MemberName   string        `json:"member_name"`
	Terminal     TerminalPlan  `json:"terminal"`
	Workspace    WorkspacePlan `json:"workspace"`
}

// TerminalPlan holds the shell command that launches the member agent.
type TerminalPlan struct {
	Command string `json:"command"`
}

// WorkspacePlan holds the filesystem setup for a member's isolated environment.
type WorkspacePlan struct {
	Memberspace string      `json:"memberspace"`
	Files       []FileEntry `json:"files"`
	GitRepo     *GitRepoRef `json:"git_repo"`
}

type GitRepoRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// FileEntry is a resolved config file to write to a member's workspace.
type FileEntry struct {
	Path    string `json:"path"`    // relative to memberspace
	Content string `json:"content"`
}
