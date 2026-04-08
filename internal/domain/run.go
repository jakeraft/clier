package domain

import (
	"errors"
	"strings"
	"time"
)

type RunStatus string

const (
	RunRunning RunStatus = "running"
	RunStopped RunStatus = "stopped"
)

// Run is an execution record of a Member or Team.
// Execution plan is saved locally to .clier/{RUN_ID}.json.
// Run stores only status and communication history (Messages, Notes).
type Run struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	UserID    int64     `json:"user_id"`
	TeamID    *int64    `json:"team_id"`    // nullable
	MemberID  *int64    `json:"member_id"`  // nullable
	Status    RunStatus `json:"status"`
	StartedAt time.Time `json:"started_at"`
	StoppedAt *time.Time `json:"stopped_at"`
}

func NewRun(id int64, name string, teamID *int64, memberID *int64) (*Run, error) {
	if id == 0 {
		return nil, errors.New("run id must not be empty")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("run name must not be empty")
	}
	return &Run{
		ID:        id,
		Name:      name,
		TeamID:    teamID,
		MemberID:  memberID,
		Status:    RunRunning,
		StartedAt: time.Now(),
	}, nil
}

// RunName generates a run name from team name and run ID.
func RunName(teamName, runID string) string {
	name := strings.NewReplacer(".", "-", ":", "-", " ", "-").Replace(teamName)
	if len(name) > 20 {
		name = name[:20]
	}
	short := runID
	if len(short) > 8 {
		short = short[:8]
	}
	return name + "-" + short
}

func (r *Run) Stop() {
	now := time.Now()
	r.Status = RunStopped
	r.StoppedAt = &now
}

// Message represents an inter-member message within a run.
// FromTeamMemberID is zero when the sender is not a team member.
type Message struct {
	ID               int64     `json:"id"`
	RunID            int64     `json:"run_id"`
	FromTeamMemberID int64     `json:"from_team_member_id"`
	ToTeamMemberID   int64     `json:"to_team_member_id"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}

func NewMessage(runID int64, fromTeamMemberID, toTeamMemberID int64, content string) (*Message, error) {
	if runID == 0 {
		return nil, errors.New("message run id must not be empty")
	}
	if toTeamMemberID == 0 {
		return nil, errors.New("message recipient must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("message content must not be empty")
	}

	return &Message{
		RunID:            runID,
		FromTeamMemberID: fromTeamMemberID,
		ToTeamMemberID:   toTeamMemberID,
		Content:          content,
		CreatedAt:        time.Now(),
	}, nil
}

// Note is a progress entry posted by a team member within a run.
type Note struct {
	ID           int64     `json:"id"`
	RunID        int64     `json:"run_id"`
	TeamMemberID int64     `json:"team_member_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewNote(runID int64, teamMemberID int64, content string) (*Note, error) {
	if runID == 0 {
		return nil, errors.New("note run id must not be empty")
	}
	if teamMemberID == 0 {
		return nil, errors.New("note team member id must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("note content must not be empty")
	}

	return &Note{
		RunID:        runID,
		TeamMemberID: teamMemberID,
		Content:      content,
		CreatedAt:    time.Now(),
	}, nil
}

// MemberPlan is the execution plan for a single team member, built from resolved resources.
// Binary, Model, Envs are NOT stored — they are already baked into Command.
// Relations are NOT stored — they are in Team.Relations and baked into the prompt.
// All paths in MemberPlan are absolute and concrete, built at Start() time.
type MemberPlan struct {
	TeamMemberID int64         `json:"team_member_id"`
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
	GitRepoURL  string      `json:"git_repo_url"`
}

// FileEntry is a config file to write to a member's workspace.
type FileEntry struct {
	Path    string `json:"path"`    // relative to memberspace
	Content string `json:"content"`
}
