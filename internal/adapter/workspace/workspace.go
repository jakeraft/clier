package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jakeraft/clier/internal/domain"
)

// Workspace manages session member filesystem environments.
type Workspace struct {
	baseDir string
}

func New(baseDir string) *Workspace {
	return &Workspace{baseDir: baseDir}
}

// Prepare creates isolated environments for all members using resolved MemberPlans.
// File paths in each MemberPlan are absolute (placeholders already resolved).
func (w *Workspace) Prepare(ctx context.Context, members []domain.MemberPlan) error {
	if len(members) == 0 {
		return nil
	}

	for _, m := range members {
		if err := w.prepareMember(ctx, m); err != nil {
			return fmt.Errorf("prepare member %s: %w", m.MemberName, err)
		}
	}

	return nil
}

// Cleanup removes all workspace files for a session (baseDir/planID).
func (w *Workspace) Cleanup(planID string) error {
	return os.RemoveAll(filepath.Join(w.baseDir, planID))
}

func (w *Workspace) prepareMember(ctx context.Context, m domain.MemberPlan) error {
	ws := m.Workspace
	workDir := filepath.Join(ws.Memberspace, "project")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("create member dir: %w", err)
	}

	for _, f := range ws.Files {
		// f.Path is absolute (after placeholder resolution)
		if err := os.MkdirAll(filepath.Dir(f.Path), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(f.Path, []byte(f.Content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", f.Path, err)
		}
	}

	if err := w.setupGit(ctx, ws, workDir); err != nil {
		return fmt.Errorf("setup git: %w", err)
	}

	return nil
}

func (w *Workspace) setupGit(ctx context.Context, ws domain.WorkspacePlan, workDir string) error {
	if ws.GitRepo == nil {
		return exec.CommandContext(ctx, "git", "init", workDir).Run()
	}
	if err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", ws.GitRepo.URL, workDir).Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", ws.GitRepo.URL, err)
	}
	return nil
}
