package settings

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"
)

// Settings is the facade for all user-configurable settings.
type Settings struct {
	Paths *Paths
}

const dotDir = ".clier"

func New() (*Settings, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	return &Settings{
		Paths: &Paths{base: filepath.Join(u.HomeDir, dotDir)},
	}, nil
}

// Paths resolves filesystem paths under ~/.clier.
type Paths struct {
	base string
}

func (p *Paths) Base() string {
	return p.base
}

func (p *Paths) Workspaces() string {
	return filepath.Join(p.base, "workspaces")
}

func (p *Paths) Dashboard() string {
	return filepath.Join(p.base, "dashboard.html")
}

// ExpandTilde replaces ~/ prefixes with the parent of the base directory (OS home).
func (p *Paths) ExpandTilde(s string) string {
	return strings.ReplaceAll(s, "~/", filepath.Dir(p.base)+"/")
}

