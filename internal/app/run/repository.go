package run

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	storerunplan "github.com/jakeraft/clier/internal/store/runplan"
)

const RunsDirName = storerunplan.RunsDirName

// Repository is the app-level boundary for persisted run plans.
type Repository struct {
	dir string
}

type Deletion interface {
	RunID() string
	Restore() error
}

type StagedDeletion struct {
	runID        string
	originalPath string
	stagedPath   string
}

func NewRepository(dir string) *Repository {
	return &Repository{dir: dir}
}

func (r *Repository) Save(plan *RunPlan) error {
	return storerunplan.Save(r.dir, plan.RunID, plan)
}

func (r *Repository) Load(runID string) (*RunPlan, error) {
	return storerunplan.Load(r.dir, runID)
}

func (r *Repository) List() ([]*RunPlan, error) {
	return storerunplan.List(r.dir)
}

func (r *Repository) Delete(runID string) error {
	if err := os.Remove(storerunplan.Path(r.dir, runID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove run plan %s: %w", runID, err)
	}
	return nil
}

func (r *Repository) StageDelete(runID, txRoot string) (Deletion, error) {
	original := storerunplan.Path(r.dir, runID)
	stagedDir := filepath.Join(txRoot, "runs")
	if err := os.MkdirAll(stagedDir, 0o755); err != nil {
		return nil, fmt.Errorf("create staged deletion dir: %w", err)
	}
	staged := filepath.Join(stagedDir, runID+".json")
	if err := os.Rename(original, staged); err != nil {
		return nil, fmt.Errorf("stage run plan %s: %w", runID, err)
	}
	return &StagedDeletion{
		runID:        runID,
		originalPath: original,
		stagedPath:   staged,
	}, nil
}

func (d *StagedDeletion) RunID() string {
	return d.runID
}

func (d *StagedDeletion) Restore() error {
	if err := os.Rename(d.stagedPath, d.originalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("restore run plan %s: %w", d.runID, err)
	}
	return nil
}

func (r *Repository) FindRunningForWorkingCopy(base string) (*RunPlan, bool, error) {
	plans, err := r.List()
	if err != nil {
		return nil, false, err
	}
	for _, plan := range plans {
		if plan.WorkingCopyPath == base && plan.Status == StatusRunning {
			return plan, true, nil
		}
	}
	return nil, false, nil
}

func (r *Repository) ListForWorkingCopy(base string) ([]*RunPlan, error) {
	plans, err := r.List()
	if err != nil {
		return nil, err
	}
	owned := make([]*RunPlan, 0)
	for _, plan := range plans {
		if plan.WorkingCopyPath == base {
			owned = append(owned, plan)
		}
	}
	return owned, nil
}
