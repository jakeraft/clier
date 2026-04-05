package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskRunning TaskStatus = "running"
	TaskStopped TaskStatus = "stopped"
)

// Task is an execution instance of a Team on a local machine.
// Plan is built fresh at task start from the team's current state.
type Task struct {
	ID     string     `json:"id"`
	TeamID string     `json:"team_id"`
	Status TaskStatus `json:"status"`
	// Plan retains {{CLIER_*}} placeholders as built. Safe for name/ID lookups;
	// paths and commands require resolution before use.
	Plan      []MemberPlan `json:"plan"`
	CreatedAt time.Time    `json:"created_at"`
	StoppedAt *time.Time   `json:"stopped_at"`
}

func NewTask(id, teamID string) (*Task, error) {
	if id == "" {
		return nil, errors.New("task id must not be empty")
	}
	if teamID == "" {
		return nil, errors.New("team id must not be empty")
	}
	return &Task{
		ID:        id,
		TeamID:    teamID,
		Status:    TaskRunning,
		CreatedAt: time.Now(),
	}, nil
}

func (t *Task) Stop() {
	now := time.Now()
	t.Status = TaskStopped
	t.StoppedAt = &now
}

// Message represents an inter-member message within a task.
// FromTeamMemberID is nullable — empty when the sender is not a team member.
type Message struct {
	ID               string    `json:"id"`
	TaskID           string    `json:"task_id"`
	FromTeamMemberID string    `json:"from_team_member_id"`
	ToTeamMemberID   string    `json:"to_team_member_id"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}

func NewMessage(taskID, fromTeamMemberID, toTeamMemberID, content string) (*Message, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, errors.New("message task id must not be empty")
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
		TaskID:           taskID,
		FromTeamMemberID: fromTeamMemberID,
		ToTeamMemberID:   toTeamMemberID,
		Content:          content,
		CreatedAt:        time.Now(),
	}, nil
}

// Note is a progress entry posted by a team member within a task.
type Note struct {
	ID           string    `json:"id"`
	TaskID       string    `json:"task_id"`
	TeamMemberID string    `json:"team_member_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewNote(taskID, teamMemberID, content string) (*Note, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, errors.New("note task id must not be empty")
	}
	if strings.TrimSpace(teamMemberID) == "" {
		return nil, errors.New("note team member id must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("note content must not be empty")
	}

	return &Note{
		ID:           uuid.NewString(),
		TaskID:       taskID,
		TeamMemberID: teamMemberID,
		Content:      content,
		CreatedAt:    time.Now(),
	}, nil
}

// MemberPlan is the execution plan for a single team member, built from resolved resources.
// Binary, Model, Envs are NOT stored — they are already baked into Command.
// Relations are NOT stored — they are in Team.Relations and baked into the prompt.
//
// Plan retains {{CLIER_*}} placeholders; these are expanded at task start
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

// FileEntry is a config file to write to a member's workspace.
type FileEntry struct {
	Path    string `json:"path"`    // relative to memberspace
	Content string `json:"content"`
}
