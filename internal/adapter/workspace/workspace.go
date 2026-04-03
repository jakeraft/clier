package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jakeraft/clier/internal/domain"
)

// Workspace manages sprint member filesystem environments.
type Workspace struct {
	baseDir string
}

func New(baseDir string) *Workspace {
	return &Workspace{baseDir: baseDir}
}

// Prepare creates isolated environments for all members using the sprint snapshot.
func (w *Workspace) Prepare(ctx context.Context, sprintID string, snapshot domain.SprintSnapshot) error {
	if len(snapshot.Members) == 0 {
		return nil
	}

	sprintDir := filepath.Join(w.baseDir, sprintID)
	if err := os.MkdirAll(sprintDir, 0755); err != nil {
		return fmt.Errorf("create sprint dir: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(sprintDir)
		}
	}()

	for _, m := range snapshot.Members {
		if err := w.prepareMember(ctx, m); err != nil {
			return fmt.Errorf("prepare member %s: %w", m.MemberName, err)
		}
	}

	success = true
	return nil
}

// Cleanup removes all workspace files for a sprint.
func (w *Workspace) Cleanup(sprintID string) error {
	return os.RemoveAll(filepath.Join(w.baseDir, sprintID))
}

func (w *Workspace) prepareMember(ctx context.Context, m domain.SprintMemberSnapshot) error {
	if err := os.MkdirAll(m.WorkDir, 0755); err != nil {
		return fmt.Errorf("create member dir: %w", err)
	}

	for _, f := range m.Files {
		path := filepath.Join(m.Home, f.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(path, []byte(f.Content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", f.Path, err)
		}
	}

	if err := w.setupGit(ctx, m); err != nil {
		return fmt.Errorf("setup git: %w", err)
	}

	return nil
}

func (w *Workspace) setupGit(ctx context.Context, m domain.SprintMemberSnapshot) error {
	if m.GitRepo == nil {
		return exec.CommandContext(ctx, "git", "init", m.WorkDir).Run()
	}
	if err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", m.GitRepo.URL, m.WorkDir).Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", m.GitRepo.URL, err)
	}
	return nil
}
