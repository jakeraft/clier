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

// SurfaceStore persists session surface refs across CLI invocations.
type SurfaceStore interface {
	SaveSessionSurface(ctx context.Context, sessionID, memberID, workspaceRef, surfaceRef string) error
	GetSessionSurface(ctx context.Context, sessionID, memberID string) (workspaceRef, surfaceRef string, err error)
	GetSessionWorkspaceRef(ctx context.Context, sessionID, excludeMemberID string) (string, error)
	DeleteSessionSurfaces(ctx context.Context, sessionID string) error
}

// callerMemberID is the reserved member ID for the human caller who started the session.
const callerMemberID = "00000000-0000-0000-0000-000000000000"

type CmuxTerminal struct {
	binary   string
	surfaces SurfaceStore
}

func NewCmuxTerminal(surfaces SurfaceStore) *CmuxTerminal {
	return &CmuxTerminal{binary: "cmux", surfaces: surfaces}
}

func (c *CmuxTerminal) Launch(sessionID, sessionName string, members []domain.MemberPlan) error {
	if len(members) == 0 {
		return errors.New("no members to launch")
	}

	if err := c.saveCallerSurface(sessionID); err != nil {
		return fmt.Errorf("save caller surface: %w", err)
	}

	wsRef, err := c.createWorkspace(sessionName)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_, _ = c.run("close-workspace", "--workspace", wsRef)
			_ = c.deleteSurfaces(sessionID)
		}
	}()

	for i, m := range members {
		surfaceRef, err := c.ensureSurface(wsRef, i)
		if err != nil {
			return fmt.Errorf("ensure surface: %w", err)
		}

		if err := c.setupMemberSurface(wsRef, surfaceRef, m); err != nil {
			return err
		}

		if err := c.saveSurface(sessionID, m.TeamMemberID, wsRef, surfaceRef); err != nil {
			return fmt.Errorf("save surface: %w", err)
		}
	}

	success = true
	return nil
}

func (c *CmuxTerminal) Send(sessionID, memberID, text string) error {
	wsRef, surfaceRef, err := c.getRefs(sessionID, memberID)
	if err != nil {
		return fmt.Errorf("get refs for %s: %w", memberID, err)
	}
	return c.sendAndEnter(wsRef, surfaceRef, text)
}

func (c *CmuxTerminal) Terminate(sessionID string) error {
	wsRef, err := c.getWorkspaceRef(sessionID)
	if err == nil {
		// Gracefully exit each agent before closing the workspace.
		c.exitAllSurfaces(wsRef)
		_, _ = c.run("close-workspace", "--workspace", wsRef)
	}
	return c.deleteSurfaces(sessionID)
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

func (c *CmuxTerminal) setupMemberSurface(wsRef, surfaceRef string, m domain.MemberPlan) error {
	if err := c.renameTab(wsRef, surfaceRef, m.MemberName); err != nil {
		return fmt.Errorf("rename tab: %w", err)
	}
	if m.Terminal.Command != "" {
		if err := c.sendAndEnter(wsRef, surfaceRef, m.Terminal.Command); err != nil {
			return fmt.Errorf("send command: %w", err)
		}
	}
	return nil
}

// persistence — delegated to SurfaceStore

func (c *CmuxTerminal) saveSurface(sessionID, memberID, workspaceRef, surfaceRef string) error {
	return c.surfaces.SaveSessionSurface(context.Background(), sessionID, memberID, workspaceRef, surfaceRef)
}

func (c *CmuxTerminal) getRefs(sessionID, memberID string) (wsRef, surfaceRef string, err error) {
	return c.surfaces.GetSessionSurface(context.Background(), sessionID, memberID)
}

func (c *CmuxTerminal) getWorkspaceRef(sessionID string) (string, error) {
	return c.surfaces.GetSessionWorkspaceRef(context.Background(), sessionID, callerMemberID)
}

func (c *CmuxTerminal) deleteSurfaces(sessionID string) error {
	return c.surfaces.DeleteSessionSurfaces(context.Background(), sessionID)
}

func (c *CmuxTerminal) saveCallerSurface(sessionID string) error {
	wsRef := os.Getenv("CMUX_WORKSPACE_ID")
	surfaceRef := os.Getenv("CMUX_SURFACE_ID")
	if wsRef == "" || surfaceRef == "" {
		return nil
	}
	return c.saveSurface(sessionID, callerMemberID, wsRef, surfaceRef)
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
