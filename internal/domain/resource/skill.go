package resource

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Skill is a Claude Code skill that gets written to {workspace}/.claude/skills/{name}/SKILL.md.
// Maps 1:1 to the Claude Code skill system.
// Name is used as the folder name, so it must be a valid directory name
// (lowercase, hyphens, no spaces or special chars).
type Skill struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// validSkillName checks that the name is safe as a directory name.
// Allows lowercase letters, digits, and hyphens only.
var validSkillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func NewSkill(name, content string) (*Skill, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("skill name must not be empty")
	}
	if !validSkillName.MatchString(name) {
		return nil, errors.New("skill name must be lowercase with hyphens only (e.g. code-review)")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("skill content must not be empty")
	}

	now := time.Now()
	return &Skill{
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *Skill) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("skill name must not be empty")
		}
		if !validSkillName.MatchString(trimmed) {
			return errors.New("skill name must be lowercase with hyphens only (e.g. code-review)")
		}
		s.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("skill content must not be empty")
		}
		s.Content = trimmed
	}
	s.UpdatedAt = time.Now()
	return nil
}
