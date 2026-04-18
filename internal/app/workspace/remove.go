package workspace

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	apprun "github.com/jakeraft/clier/internal/app/run"
	"github.com/jakeraft/clier/internal/domain"
)

type RunPlanRemover interface {
	ListForWorkingCopy(base string) ([]*apprun.RunPlan, error)
	StageDelete(runID, txRoot string) (apprun.Deletion, error)
}

func (s *Service) Remove(base string, runs RunPlanRemover) ([]string, error) {
	if _, err := LoadManifest(s.fs, base); err != nil {
		return nil, err
	}

	modified, err := s.ModifiedTrackedResources(base)
	if err != nil {
		return nil, err
	}
	if len(modified) > 0 {
		return nil, &domain.Fault{
			Kind:    domain.KindWorkspaceDirty,
			Subject: map[string]string{"modified": strconv.Itoa(len(modified))},
		}
	}

	owned, err := runs.ListForWorkingCopy(base)
	if err != nil {
		return nil, err
	}
	for _, plan := range owned {
		if plan.Status == apprun.StatusRunning {
			return nil, &domain.Fault{
				Kind:    domain.KindRunBlocksRemove,
				Subject: map[string]string{"run_id": plan.RunID},
			}
		}
	}

	txRoot := removeTxRoot(base)
	workspaceDeletion, err := stageWorkingCopyDeletion(s.fs, base, txRoot)
	if err != nil {
		return nil, err
	}

	staged := make([]apprun.Deletion, 0, len(owned))
	restore := func() {
		for i := len(staged) - 1; i >= 0; i-- {
			_ = staged[i].Restore()
		}
		_ = workspaceDeletion.Restore()
	}

	removedRuns := make([]string, 0, len(owned))
	for _, plan := range owned {
		deletion, err := runs.StageDelete(plan.RunID, txRoot)
		if err != nil {
			restore()
			return nil, err
		}
		staged = append(staged, deletion)
		removedRuns = append(removedRuns, deletion.RunID())
	}

	bestEffortPurgeRemovalTx(s.fs, txRoot)

	return removedRuns, nil
}

type stagedWorkingCopy struct {
	fs           FileMaterializer
	originalPath string
	stagedPath   string
}

func stageWorkingCopyDeletion(fs FileMaterializer, base, txRoot string) (*stagedWorkingCopy, error) {
	staged := filepath.Join(txRoot, "workspace", filepath.Base(base))
	if err := fs.Rename(base, staged); err != nil {
		return nil, fmt.Errorf("stage working copy %s: %w", base, err)
	}
	return &stagedWorkingCopy{
		fs:           fs,
		originalPath: base,
		stagedPath:   staged,
	}, nil
}

func (d *stagedWorkingCopy) Restore() error {
	if err := d.fs.Rename(d.stagedPath, d.originalPath); err != nil {
		return fmt.Errorf("restore working copy %s: %w", d.originalPath, err)
	}
	return nil
}

func bestEffortPurgeRemovalTx(fs FileMaterializer, txRoot string) {
	_ = fs.RemoveAll(txRoot)
}

func removeTxRoot(base string) string {
	return filepath.Join(
		filepath.Dir(base),
		".deleting",
		fmt.Sprintf("remove-%s-%d", filepath.Base(base), time.Now().UTC().UnixNano()),
	)
}
