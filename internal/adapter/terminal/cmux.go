package terminal

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jakeraft/clier/internal/app/sprint"
)

type CmuxTerminal struct {
	binary string
	db     *sql.DB
}

func NewCmuxTerminal(db *sql.DB) *CmuxTerminal {
	return &CmuxTerminal{binary: "cmux", db: db}
}

func (c *CmuxTerminal) Launch(sprintID, sprintName string, members []sprint.MemberSpec) error {
	if len(members) == 0 {
		return fmt.Errorf("no members to launch")
	}

	wsRef, err := c.createWorkspace(sprintName)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	// Cleanup workspace and saved surfaces on any subsequent failure
	success := false
	defer func() {
		if !success {
			_, _ = c.run("close-workspace", "--workspace", wsRef)
			_ = c.deleteSurfaces(sprintID)
		}
	}()

	for i, m := range members {
		surfaceRef, err := c.ensureSurface(wsRef, i)
		if err != nil {
			return fmt.Errorf("ensure surface: %w", err)
		}

		if err := c.setupSurface(wsRef, surfaceRef, m); err != nil {
			return err
		}

		if err := c.saveSurface(sprintID, m.ID, wsRef, surfaceRef); err != nil {
			return fmt.Errorf("save surface: %w", err)
		}
	}

	success = true
	return nil
}

func (c *CmuxTerminal) Send(sprintID, memberID, text string) error {
	surfaceRef, err := c.getSurfaceRef(sprintID, memberID)
	if err != nil {
		return fmt.Errorf("get surface ref for %s: %w", memberID, err)
	}
	return c.sendAndEnter(surfaceRef, text)
}

func (c *CmuxTerminal) Terminate(sprintID string) error {
	wsRef, err := c.getWorkspaceRef(sprintID)
	if err != nil {
		return fmt.Errorf("get workspace ref: %w", err)
	}

	if _, err := c.run("close-workspace", "--workspace", wsRef); err != nil {
		return err
	}

	return c.deleteSurfaces(sprintID)
}

// setupSurface renames the tab and launches the command via respawn-pane.
// Using respawn-pane --command avoids the lazy-init issue where cmux
// terminals in non-visible workspaces reject send commands.
func (c *CmuxTerminal) setupSurface(wsRef, surfaceRef string, m sprint.MemberSpec) error {
	if err := c.renameTab(wsRef, surfaceRef, m.Name); err != nil {
		return fmt.Errorf("rename tab: %w", err)
	}
	if m.Command != "" {
		if _, err := c.run("respawn-pane", "--workspace", wsRef, "--surface", surfaceRef, "--command", m.Command); err != nil {
			return fmt.Errorf("launch command: %w", err)
		}
	}
	return nil
}

// persistence — sprint_surfaces table

func (c *CmuxTerminal) saveSurface(sprintID, memberID, workspaceRef, surfaceRef string) error {
	_, err := c.db.Exec(
		"INSERT INTO sprint_surfaces (sprint_id, member_id, workspace_ref, surface_ref) VALUES (?, ?, ?, ?)",
		sprintID, memberID, workspaceRef, surfaceRef,
	)
	return err
}

func (c *CmuxTerminal) getSurfaceRef(sprintID, memberID string) (string, error) {
	var ref string
	err := c.db.QueryRow(
		"SELECT surface_ref FROM sprint_surfaces WHERE sprint_id = ? AND member_id = ?",
		sprintID, memberID,
	).Scan(&ref)
	return ref, err
}

func (c *CmuxTerminal) getWorkspaceRef(sprintID string) (string, error) {
	var ref string
	err := c.db.QueryRow(
		"SELECT workspace_ref FROM sprint_surfaces WHERE sprint_id = ? LIMIT 1",
		sprintID,
	).Scan(&ref)
	return ref, err
}

func (c *CmuxTerminal) deleteSurfaces(sprintID string) error {
	_, err := c.db.Exec("DELETE FROM sprint_surfaces WHERE sprint_id = ?", sprintID)
	return err
}

// cmux command helpers

func (c *CmuxTerminal) createWorkspace(name string) (string, error) {
	out, err := c.run("new-workspace")
	if err != nil {
		return "", err
	}
	wsRef, err := parseRef(out, "workspace:")
	if err != nil {
		return "", err
	}
	if err := c.renameWorkspace(wsRef, name); err != nil {
		_, _ = c.run("close-workspace", "--workspace", wsRef)
		return "", err
	}
	return wsRef, nil
}

// ensureSurface returns a surface ref. The first surface (index 0) is created
// with the workspace; subsequent surfaces are added explicitly.
func (c *CmuxTerminal) ensureSurface(wsRef string, index int) (string, error) {
	var out string
	var err error
	if index == 0 {
		out, err = c.run("list-pane-surfaces", "--workspace", wsRef)
	} else {
		out, err = c.run("new-surface", "--workspace", wsRef)
	}
	if err != nil {
		return "", err
	}
	return parseRef(out, "surface:")
}

func (c *CmuxTerminal) renameWorkspace(wsRef, name string) error {
	_, err := c.run("rename-workspace", "--workspace", wsRef, name)
	return err
}

func (c *CmuxTerminal) renameTab(wsRef, surfaceRef, name string) error {
	_, err := c.run("tab-action", "--action", "rename", "--surface", surfaceRef, "--workspace", wsRef, "--title", name)
	return err
}

func (c *CmuxTerminal) sendAndEnter(surfaceRef, text string) error {
	if _, err := c.run("send", "--surface", surfaceRef, text); err != nil {
		return err
	}
	_, err := c.run("send-key", "--surface", surfaceRef, "Enter")
	return err
}

// run executes the cmux binary with the given arguments.
func (c *CmuxTerminal) run(args ...string) (string, error) {
	cmd := exec.Command(c.binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", c.binary, args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func parseRef(output, prefix string) (string, error) {
	for _, part := range strings.Fields(output) {
		if strings.HasPrefix(part, prefix) {
			return part, nil
		}
	}
	return "", fmt.Errorf("ref not found in output: %s", output)
}
