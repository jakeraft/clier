package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID           string
	SprintID     string
	FromMemberID string
	ToMemberID   string
	Content      string
	CreatedAt    time.Time
}

func NewMessage(sprintID, fromMemberID, toMemberID, content string) (*Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("message content must not be empty")
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
