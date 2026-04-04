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

// Session is an execution instance of a Team's plan on a local machine.
type Session struct {
	ID        string        `json:"id"`
	TeamID    string        `json:"team_id"`
	Status    SessionStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	StoppedAt *time.Time    `json:"stopped_at"`
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
