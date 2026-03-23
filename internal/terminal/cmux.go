package terminal

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type CmuxTerminal struct {
	mu         sync.Mutex
	workspaces map[string]string // sprintID → workspace ref
}

func NewCmuxTerminal() *CmuxTerminal {
	return &CmuxTerminal{
		workspaces: make(map[string]string),
	}
}

func (c *CmuxTerminal) Launch(sprintID, sprintName string, members []MemberLaunch) (*LaunchResult, error) {
	if len(members) == 0 {
		return nil, fmt.Errorf("no members to launch")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.workspaces[sprintID]; exists {
		return nil, fmt.Errorf("sprint already launched: %s", sprintID)
	}

	first := members[0]

	wsRef, err := createWorkspace(first.Command, first.Env)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	c.workspaces[sprintID] = wsRef

	if err := renameWorkspace(wsRef, sprintName); err != nil {
		return nil, fmt.Errorf("rename workspace: %w", err)
	}

	result := &LaunchResult{
		WorkspaceRef: wsRef,
		Surfaces:     make(map[string]string, len(members)),
	}

	// First surface is created with the workspace
	firstSurfaceRef, err := findFirstSurface(wsRef)
	if err != nil {
		return nil, fmt.Errorf("find first surface: %w", err)
	}
	result.Surfaces[first.MemberID] = firstSurfaceRef

	if err := renameTab(firstSurfaceRef, first.MemberName); err != nil {
		return nil, fmt.Errorf("rename tab: %w", err)
	}

	// Launch remaining members
	for _, m := range members[1:] {
		surfaceRef, err := addSurface(wsRef, m.Command, m.Env)
		if err != nil {
			return nil, fmt.Errorf("add surface for %s: %w", m.MemberID, err)
		}
		result.Surfaces[m.MemberID] = surfaceRef

		if err := renameTab(surfaceRef, m.MemberName); err != nil {
			return nil, fmt.Errorf("rename tab for %s: %w", m.MemberID, err)
		}
	}

	return result, nil
}

func (c *CmuxTerminal) Terminate(sprintID string) error {
	c.mu.Lock()
	wsRef, ok := c.workspaces[sprintID]
	if !ok {
		c.mu.Unlock()
		return fmt.Errorf("sprint not found: %s", sprintID)
	}
	delete(c.workspaces, sprintID)
	c.mu.Unlock()

	return cmuxRun("close-workspace", "--workspace", wsRef)
}

// CmuxSend sends text to a surface. Exported for use by message routing.
func CmuxSend(surfaceRef, text string) error {
	return cmuxRun("send", "--surface", surfaceRef, text+"\n")
}

// cmux command helpers

func createWorkspace(command string, env []string) (string, error) {
	args := []string{"new-workspace"}
	if command != "" {
		args = append(args, "--command", buildEnvCommand(command, env))
	}
	out, err := cmuxOutput(args...)
	if err != nil {
		return "", err
	}
	return parseRef(out, "workspace:")
}

func findFirstSurface(wsRef string) (string, error) {
	out, err := cmuxOutput("list-pane-surfaces", "--workspace", wsRef)
	if err != nil {
		return "", err
	}
	return parseRef(out, "surface:")
}

func addSurface(wsRef, command string, env []string) (string, error) {
	out, err := cmuxOutput("new-surface", "--workspace", wsRef)
	if err != nil {
		return "", err
	}
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

func renameWorkspace(wsRef, name string) error {
	return cmuxRun("rename-workspace", "--workspace", wsRef, name)
}

func renameTab(surfaceRef, name string) error {
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
	parts := make([]string, 0, len(env)+1)
	for _, e := range env {
		parts = append(parts, "export "+e)
	}
	parts = append(parts, command)
	return strings.Join(parts, " && ")
}
