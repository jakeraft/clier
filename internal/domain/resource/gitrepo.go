package resource

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type GitRepo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewGitRepo(name, url string) (*GitRepo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("git repo name must not be empty")
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, errors.New("git repo url must not be empty")
	}

	now := time.Now()
	return &GitRepo{
		ID:        uuid.NewString(),
		Name:      name,
		URL:       url,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *GitRepo) Update(name, url *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("git repo name must not be empty")
		}
		r.Name = trimmed
	}
	if url != nil {
		trimmed := strings.TrimSpace(*url)
		if trimmed == "" {
			return errors.New("git repo url must not be empty")
		}
		r.URL = trimmed
	}
	r.UpdatedAt = time.Now()
	return nil
}
