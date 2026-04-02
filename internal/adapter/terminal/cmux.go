package terminal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// SurfaceStore persists sprint surface refs across CLI invocations.
type SurfaceStore interface {
	SaveSprintSurface(ctx context.Context, sprintID, memberID, workspaceRef, surfaceRef string) error
	GetSprintSurface(ctx context.Context, sprintID, memberID string) (workspaceRef, surfaceRef string, err error)
	GetSprintWorkspaceRef(ctx context.Context, sprintID, excludeMemberID string) (string, error)
	DeleteSprintSurfaces(ctx context.Context, sprintID string) error
}

type CmuxTerminal struct {
	binary   string
	surfaces SurfaceStore
}

func NewCmuxTerminal(surfaces SurfaceStore) *CmuxTerminal {
	return &CmuxTerminal{binary: "cmux", surfaces: surfaces}
}

func (c *CmuxTerminal) Launch(sprintID, sprintName string, snapshot domain.SprintSnapshot) error {
	if len(snapshot.Members) == 0 {
		return errors.New("no members to launch")
	}

	if err := c.saveCallerSurface(sprintID); err != nil {
		return fmt.Errorf("save caller surface: %w", err)
	}

	wsRef, err := c.createWorkspace(sprintName)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_, _ = c.run("close-workspace", "--workspace", wsRef)
			_ = c.deleteSurfaces(sprintID)
		}
	}()

	for i, m := range snapshot.Members {
		surfaceRef, err := c.ensureSurface(wsRef, i)
		if err != nil {
			return fmt.Errorf("ensure surface: %w", err)
		}

		if err := c.setupMemberSurface(wsRef, surfaceRef, m); err != nil {
			return err
		}

		if err := c.saveSurface(sprintID, m.MemberID, wsRef, surfaceRef); err != nil {
			return fmt.Errorf("save surface: %w", err)
		}
	}

	success = true
	return nil
}

func (c *CmuxTerminal) Send(sprintID, memberID, text string) error {
	wsRef, surfaceRef, err := c.getRefs(sprintID, memberID)
	if err != nil {
		return fmt.Errorf("get refs for %s: %w", memberID, err)
	}
	return c.sendAndEnter(wsRef, surfaceRef, text)
}

func (c *CmuxTerminal) Terminate(sprintID string) error {
	wsRef, err := c.getWorkspaceRef(sprintID)
	if err == nil {
		// Gracefully exit each agent before closing the workspace.
		c.exitAllSurfaces(wsRef)
		_, _ = c.run("close-workspace", "--workspace", wsRef)
	}
	return c.deleteSurfaces(sprintID)
}

// exitAllSurfaces sends /exit to every surface in the workspace so agents
// shut down gracefully and don't recreate config dirs after cleanup.
func (c *CmuxTerminal) exitAllSurfaces(wsRef string) {
	out, err := c.run("list-pane-surfaces", "--workspace", wsRef)
	if err != nil {
		return
	}
	for ref := range strings.FieldsSeq(out) {
		if strings.HasPrefix(ref, "surface:") {
			_ = c.sendAndEnter(wsRef, ref, "/exit")
		}
	}
}

func (c *CmuxTerminal) setupMemberSurface(wsRef, surfaceRef string, m domain.SprintMemberSnapshot) error {
	if err := c.renameTab(wsRef, surfaceRef, m.MemberName); err != nil {
		return fmt.Errorf("rename tab: %w", err)
	}
	if m.Command != "" {
		if err := c.sendAndEnter(wsRef, surfaceRef, m.Command); err != nil {
			return fmt.Errorf("send command: %w", err)
		}
	}
	return nil
}

// persistence — delegated to SurfaceStore

func (c *CmuxTerminal) saveSurface(sprintID, memberID, workspaceRef, surfaceRef string) error {
	return c.surfaces.SaveSprintSurface(context.Background(), sprintID, memberID, workspaceRef, surfaceRef)
}

func (c *CmuxTerminal) getRefs(sprintID, memberID string) (wsRef, surfaceRef string, err error) {
	return c.surfaces.GetSprintSurface(context.Background(), sprintID, memberID)
}

func (c *CmuxTerminal) getWorkspaceRef(sprintID string) (string, error) {
	return c.surfaces.GetSprintWorkspaceRef(context.Background(), sprintID, domain.UserMemberID)
}

func (c *CmuxTerminal) deleteSurfaces(sprintID string) error {
	return c.surfaces.DeleteSprintSurfaces(context.Background(), sprintID)
}

func (c *CmuxTerminal) saveCallerSurface(sprintID string) error {
	wsRef := os.Getenv("CMUX_WORKSPACE_ID")
	surfaceRef := os.Getenv("CMUX_SURFACE_ID")
	if wsRef == "" || surfaceRef == "" {
		return nil
	}
	return c.saveSurface(sprintID, domain.UserMemberID, wsRef, surfaceRef)
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
	c.setManagedStatus(wsRef)
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

func (c *CmuxTerminal) setManagedStatus(wsRef string) {
	_, _ = c.run("set-status", "clier", "Managed by clier",
		"--icon", "gearshape.2.fill", "--color", "#8B5CF6", "--workspace", wsRef)
}

func (c *CmuxTerminal) renameTab(wsRef, surfaceRef, name string) error {
	_, err := c.run("tab-action", "--action", "rename", "--surface", surfaceRef, "--workspace", wsRef, "--title", name)
	return err
}

func (c *CmuxTerminal) sendAndEnter(wsRef, surfaceRef, text string) error {
	if _, err := c.run("send", "--workspace", wsRef, "--surface", surfaceRef, text); err != nil {
		return err
	}
	_, err := c.run("send-key", "--workspace", wsRef, "--surface", surfaceRef, "Enter")
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
	for part := range strings.FieldsSeq(output) {
		if strings.HasPrefix(part, prefix) {
			return part, nil
		}
	}
	return "", fmt.Errorf("ref not found in output: %s", output)
}
