package api

import (
	"fmt"
	"time"
)

// RunResponse is the server's JSON representation of a Run.
type RunResponse struct {
	ID        int64             `json:"id"`
	UserID    int64             `json:"user_id"`
	Name      string            `json:"name"`
	TeamID    *int64            `json:"team_id,omitempty"`
	MemberID  *int64            `json:"member_id,omitempty"`
	Status    string            `json:"status"`
	Messages  []MessageResponse `json:"messages"`
	Notes     []NoteResponse    `json:"notes"`
	StartedAt time.Time         `json:"started_at"`
	StoppedAt *time.Time        `json:"stopped_at,omitempty"`
}

// MessageResponse is the server's JSON representation of a Message.
type MessageResponse struct {
	ID               string    `json:"id"`
	RunID            string    `json:"run_id"`
	FromTeamMemberID int64     `json:"from_team_member_id"`
	ToTeamMemberID   int64     `json:"to_team_member_id"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}

// NoteResponse is the server's JSON representation of a Note.
type NoteResponse struct {
	ID           string    `json:"id"`
	RunID        string    `json:"run_id"`
	TeamMemberID int64     `json:"team_member_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

func (c *Client) CreateRun(body any) (*RunResponse, error) {
	var r RunResponse
	return &r, c.post("/api/v1/runs", body, &r)
}

func (c *Client) GetRun(id string) (*RunResponse, error) {
	var r RunResponse
	return &r, c.get(fmt.Sprintf("/api/v1/runs/%s", id), &r)
}

func (c *Client) ListRuns() ([]RunResponse, error) {
	var r []RunResponse
	return r, c.get("/api/v1/runs", &r)
}

func (c *Client) UpdateRunStatus(id string, body any) error {
	return c.patch(fmt.Sprintf("/api/v1/runs/%s", id), body, nil)
}

func (c *Client) DeleteRun(id string) error {
	return c.delete(fmt.Sprintf("/api/v1/runs/%s", id))
}

func (c *Client) AddMessage(runID string, body any) (*MessageResponse, error) {
	var r MessageResponse
	return &r, c.post(fmt.Sprintf("/api/v1/runs/%s/messages", runID), body, &r)
}

func (c *Client) AddNote(runID string, body any) (*NoteResponse, error) {
	var r NoteResponse
	return &r, c.post(fmt.Sprintf("/api/v1/runs/%s/notes", runID), body, &r)
}
