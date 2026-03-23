package terminal

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type CmuxTerminal struct {
	mu         sync.Mutex
	workspaces map[string]string            // sprintID → workspace ref
	surfaces   map[string]map[string]string // sprintID → memberID → surface ref
}

func NewCmuxTerminal() *CmuxTerminal {
	return &CmuxTerminal{
		workspaces: make(map[string]string),
		surfaces:   make(map[string]map[string]string),
	}
}

func (c *CmuxTerminal) Launch(sprintID, sprintName string, members []MemberLaunch) error {
	if len(members) == 0 {
		return fmt.Errorf("no members to launch")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.workspaces[sprintID]; exists {
		return fmt.Errorf("sprint already launched: %s", sprintID)
	}

	first := members[0]

	// Create workspace with first member's command
	wsRef, err := c.createWorkspace(first.Command, first.Env)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	c.workspaces[sprintID] = wsRef
	c.surfaces[sprintID] = make(map[string]string)

	if err := c.renameWorkspace(wsRef, sprintName); err != nil {
		return fmt.Errorf("rename workspace: %w", err)
	}

	// First surface is created with the workspace — find it
	firstSurfaceRef, err := c.findFirstSurface(wsRef)
	if err != nil {
		return fmt.Errorf("find first surface: %w", err)
	}
	c.surfaces[sprintID][first.MemberID] = firstSurfaceRef

	if err := c.renameTab(firstSurfaceRef, first.MemberName); err != nil {
		return fmt.Errorf("rename tab: %w", err)
	}

	// Launch remaining members
	for _, m := range members[1:] {
		surfaceRef, err := c.addSurface(wsRef, m.Command, m.Env)
		if err != nil {
			return fmt.Errorf("add surface for %s: %w", m.MemberID, err)
		}
		c.surfaces[sprintID][m.MemberID] = surfaceRef

		if err := c.renameTab(surfaceRef, m.MemberName); err != nil {
			return fmt.Errorf("rename tab for %s: %w", m.MemberID, err)
		}
	}

	return nil
}

func (c *CmuxTerminal) DeliverText(sprintID, memberID, text string) error {
	c.mu.Lock()
	surfaceRef, ok := c.surfaces[sprintID][memberID]
	c.mu.Unlock()

	if !ok {
		return fmt.Errorf("surface not found: sprint=%s member=%s", sprintID, memberID)
	}

	return cmuxRun("send", "--surface", surfaceRef, text+"\n")
}

func (c *CmuxTerminal) Terminate(sprintID string) error {
	c.mu.Lock()
	wsRef, ok := c.workspaces[sprintID]
	if !ok {
		c.mu.Unlock()
		return fmt.Errorf("sprint not found: %s", sprintID)
	}
	delete(c.workspaces, sprintID)
	delete(c.surfaces, sprintID)
	c.mu.Unlock()

	return cmuxRun("close-workspace", "--workspace", wsRef)
}

// cmux command helpers

func (c *CmuxTerminal) createWorkspace(command string, env []string) (string, error) {
	args := []string{"new-workspace"}
	if command != "" {
		args = append(args, "--command", buildEnvCommand(command, env))
	}

	out, err := cmuxOutput(args...)
	if err != nil {
		return "", err
	}
	// Output: "OK workspace:N"
	return parseRef(out, "workspace:")
}

func (c *CmuxTerminal) findFirstSurface(wsRef string) (string, error) {
	out, err := cmuxOutput("list-pane-surfaces", "--workspace", wsRef)
	if err != nil {
		return "", err
	}
	// Output: "* surface:N  title  [selected]"
	return parseRef(out, "surface:")
}

func (c *CmuxTerminal) addSurface(wsRef, command string, env []string) (string, error) {
	out, err := cmuxOutput("new-surface", "--workspace", wsRef)
	if err != nil {
		return "", err
	}
	// Output: "OK surface:N pane:N workspace:N"
	surfaceRef, err := parseRef(out, "surface:")
	if err != nil {
		return "", err
	}

	if command != "" {
		if err := cmuxRun("send", "--surface", surfaceRef, buildEnvCommand(command, env)+"\n"); err != nil {
			return "", err
		}
	}

	return surfaceRef, nil
}

func (c *CmuxTerminal) renameWorkspace(wsRef, name string) error {
	return cmuxRun("rename-workspace", "--workspace", wsRef, name)
}

func (c *CmuxTerminal) renameTab(surfaceRef, name string) error {
	return cmuxRun("rename-tab", "--surface", surfaceRef, name)
}

// utilities

func cmuxRun(args ...string) error {
	cmd := exec.Command("cmux", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cmux %s: %s", args[0], strings.TrimSpace(string(out)))
	}
	return nil
}

func cmuxOutput(args ...string) (string, error) {
	cmd := exec.Command("cmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cmux %s: %s", args[0], strings.TrimSpace(string(out)))
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

func buildEnvCommand(command string, env []string) string {
	if len(env) == 0 {
		return command
	}
	parts := make([]string, 0, len(env)+2)
	parts = append(parts, "env")
	parts = append(parts, env...)
	parts = append(parts, command)
	return strings.Join(parts, " ")
}
