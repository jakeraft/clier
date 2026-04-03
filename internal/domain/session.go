package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UserMemberID is the reserved member ID for the human user who started the session.
const UserMemberID = "00000000-0000-0000-0000-000000000000"

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
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	FromMemberID string    `json:"from_member_id"`
	ToMemberID   string    `json:"to_member_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewMessage(sessionID, fromMemberID, toMemberID, content string) (*Message, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("message session id must not be empty")
	}
	if strings.TrimSpace(toMemberID) == "" {
		return nil, errors.New("message recipient must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("message content must not be empty")
	}

	return &Message{
		ID:           uuid.NewString(),
		SessionID:    sessionID,
		FromMemberID: fromMemberID,
		ToMemberID:   toMemberID,
		Content:      content,
		CreatedAt:    time.Now(),
	}, nil
}
