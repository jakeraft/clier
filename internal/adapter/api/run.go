package api

import (
	"fmt"
	"time"

	"github.com/jakeraft/clier/internal/domain"
)

// RunResponse is the server's JSON representation of a Run.
type RunResponse struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	TeamID    string             `json:"team_id"`
	Status    domain.RunStatus   `json:"status"`
	Plan      []domain.MemberPlan `json:"plan"`
	StartedAt time.Time          `json:"started_at"`
	StoppedAt *time.Time         `json:"stopped_at"`
}

// MessageResponse is the server's JSON representation of a Message.
type MessageResponse struct {
	ID               string    `json:"id"`
	RunID            string    `json:"run_id"`
	FromTeamMemberID string    `json:"from_team_member_id"`
	ToTeamMemberID   string    `json:"to_team_member_id"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}

// NoteResponse is the server's JSON representation of a Note.
type NoteResponse struct {
	ID           string    `json:"id"`
	RunID        string    `json:"run_id"`
	TeamMemberID string    `json:"team_member_id"`
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
	return c.patch(fmt.Sprintf("/api/v1/runs/%s/status", id), body, nil)
}

func (c *Client) DeleteRun(id string) error {
	return c.delete(fmt.Sprintf("/api/v1/runs/%s", id))
}

func (c *Client) AddMessage(runID string, body any) (*MessageResponse, error) {
	var r MessageResponse
	return &r, c.post(fmt.Sprintf("/api/v1/runs/%s/messages", runID), body, &r)
}

func (c *Client) ListMessages(runID string) ([]MessageResponse, error) {
	var r []MessageResponse
	return r, c.get(fmt.Sprintf("/api/v1/runs/%s/messages", runID), &r)
}

func (c *Client) AddNote(runID string, body any) (*NoteResponse, error) {
	var r NoteResponse
	return &r, c.post(fmt.Sprintf("/api/v1/runs/%s/notes", runID), body, &r)
}

func (c *Client) ListNotes(runID string) ([]NoteResponse, error) {
	var r []NoteResponse
	return r, c.get(fmt.Sprintf("/api/v1/runs/%s/notes", runID), &r)
}

// SaveTerminalRefs persists terminal refs for a run member.
func (c *Client) SaveTerminalRefs(runID, memberID string, refs map[string]string) error {
	body := map[string]any{
		"team_member_id": memberID,
		"refs":           refs,
	}
	return c.put(fmt.Sprintf("/api/v1/runs/%s/terminal-refs/%s", runID, memberID), body, nil)
}

// GetTerminalRefs retrieves terminal refs for a specific member.
func (c *Client) GetTerminalRefs(runID, memberID string) (map[string]string, error) {
	var r map[string]string
	return r, c.get(fmt.Sprintf("/api/v1/runs/%s/terminal-refs/%s", runID, memberID), &r)
}

// GetRunTerminalRefs retrieves terminal refs for the entire run.
func (c *Client) GetRunTerminalRefs(runID string) (map[string]string, error) {
	var r map[string]string
	return r, c.get(fmt.Sprintf("/api/v1/runs/%s/terminal-refs", runID), &r)
}

// DeleteTerminalRefs deletes all terminal refs for a run.
func (c *Client) DeleteTerminalRefs(runID string) error {
	return c.delete(fmt.Sprintf("/api/v1/runs/%s/terminal-refs", runID))
}
