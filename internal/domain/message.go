package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID           string    `json:"id"`
	SprintID     string    `json:"sprint_id"`
	FromMemberID string    `json:"from_member_id"`
	ToMemberID   string    `json:"to_member_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewMessage(sprintID, fromMemberID, toMemberID, content string) (*Message, error) {
	if strings.TrimSpace(sprintID) == "" {
		return nil, errors.New("message sprint id must not be empty")
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
		SprintID:     sprintID,
		FromMemberID: fromMemberID,
		ToMemberID:   toMemberID,
		Content:      content,
		CreatedAt:    time.Now(),
	}, nil
}
