package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Git is the narrow port the runner uses. Only `Clone` is needed for the
// thin CLI — push/pull/fetch live in the server side now.
type Git interface {
	Clone(repoURL, targetDir string) error
}

// Real shells out to the git CLI. Shallow clone (depth 1) keeps the run
// scratch dir cheap.
type Real struct{}

func New() *Real { return &Real{} }

func (g *Real) Clone(repoURL, targetDir string) error {
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return fmt.Errorf("create parent dir for clone: %w", err)
	}
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, targetDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %w: %s", repoURL, err, strings.TrimSpace(string(out)))
	}
	return nil
}
