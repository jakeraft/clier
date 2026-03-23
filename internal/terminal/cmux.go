package terminal

import (
	"fmt"
	"os/exec"
	"strings"
)

type SurfaceSpec struct {
	Name    string
	Command string
}

type LaunchResult struct {
	WorkspaceRef string
	Surfaces     []string // surface refs in same order as input specs
}

type CmuxTerminal struct {
	binary string
}

func NewCmuxTerminal() *CmuxTerminal {
	return &CmuxTerminal{binary: "cmux"}
}

func (c *CmuxTerminal) Launch(workspaceName string, specs []SurfaceSpec) (*LaunchResult, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("no surfaces to launch")
	}

	wsRef, err := c.createWorkspace()
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	// Cleanup workspace on any subsequent failure
	success := false
	defer func() {
		if !success {
			_, _ = c.run("close-workspace", "--workspace", wsRef)
		}
	}()

	if err := c.renameWorkspace(wsRef, workspaceName); err != nil {
		return nil, fmt.Errorf("rename workspace: %w", err)
	}

	result := &LaunchResult{
		WorkspaceRef: wsRef,
		Surfaces:     make([]string, 0, len(specs)),
	}

	// First surface is created with the workspace
	firstSurfaceRef, err := c.findFirstSurface(wsRef)
	if err != nil {
		return nil, fmt.Errorf("find first surface: %w", err)
	}

	if err := c.renameTab(firstSurfaceRef, specs[0].Name); err != nil {
		return nil, fmt.Errorf("rename tab: %w", err)
	}

	if specs[0].Command != "" {
		if err := c.Send(firstSurfaceRef, specs[0].Command); err != nil {
			return nil, fmt.Errorf("send command: %w", err)
		}
	}

	result.Surfaces = append(result.Surfaces, firstSurfaceRef)

	// Launch remaining surfaces
	for _, s := range specs[1:] {
		surfaceRef, err := c.addSurface(wsRef)
		if err != nil {
			return nil, fmt.Errorf("add surface: %w", err)
		}

		if err := c.renameTab(surfaceRef, s.Name); err != nil {
			return nil, fmt.Errorf("rename tab: %w", err)
		}

		if s.Command != "" {
			if err := c.Send(surfaceRef, s.Command); err != nil {
				return nil, fmt.Errorf("send command: %w", err)
			}
		}

		result.Surfaces = append(result.Surfaces, surfaceRef)
	}

	success = true
	return result, nil
}

func (c *CmuxTerminal) Terminate(workspaceRef string) error {
	_, err := c.run("close-workspace", "--workspace", workspaceRef)
	return err
}

// Send sends text to a surface.
func (c *CmuxTerminal) Send(surfaceRef, text string) error {
	_, err := c.run("send", "--surface", surfaceRef, text+"\n")
	return err
}

// cmux command helpers

func (c *CmuxTerminal) createWorkspace() (string, error) {
	out, err := c.run("new-workspace")
	if err != nil {
		return "", err
	}
	return parseRef(out, "workspace:")
}

func (c *CmuxTerminal) findFirstSurface(wsRef string) (string, error) {
	out, err := c.run("list-pane-surfaces", "--workspace", wsRef)
	if err != nil {
		return "", err
	}
	return parseRef(out, "surface:")
}

func (c *CmuxTerminal) addSurface(wsRef string) (string, error) {
	out, err := c.run("new-surface", "--workspace", wsRef)
	if err != nil {
		return "", err
	}
	return parseRef(out, "surface:")
}

func (c *CmuxTerminal) renameWorkspace(wsRef, name string) error {
	_, err := c.run("rename-workspace", "--workspace", wsRef, name)
	return err
}

func (c *CmuxTerminal) renameTab(surfaceRef, name string) error {
	_, err := c.run("rename-tab", "--surface", surfaceRef, name)
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

// ShellQuote wraps a string in single quotes, escaping embedded single quotes.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
