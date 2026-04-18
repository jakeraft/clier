package workspace

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	apprun "github.com/jakeraft/clier/internal/app/run"
	"github.com/jakeraft/clier/internal/domain"
)

type removeFS struct {
	FileMaterializer
	removeErr error
}

func (f *removeFS) RemoveAll(path string) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	return f.FileMaterializer.RemoveAll(path)
}

type stagedDeletionStub struct {
	runID    string
	restored bool
}

func (d *stagedDeletionStub) RunID() string  { return d.runID }
func (d *stagedDeletionStub) Restore() error { d.restored = true; return nil }

type runRemoverStub struct {
	plans  []*apprun.RunPlan
	staged []*stagedDeletionStub
}

func (r *runRemoverStub) ListForWorkingCopy(base string) ([]*apprun.RunPlan, error) {
	var out []*apprun.RunPlan
	for _, plan := range r.plans {
		if plan.WorkingCopyPath == base {
			out = append(out, plan)
		}
	}
	return out, nil
}

func (r *runRemoverStub) StageDelete(runID, _ string) (apprun.Deletion, error) {
	d := &stagedDeletionStub{runID: runID}
	r.staged = append(r.staged, d)
	return d, nil
}

func TestRemove_SucceedsWhenCleanupPurgeFails(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	localFS := &removeFS{FileMaterializer: osFS{}}
	manifestPath := filepath.Join(base, ".clier", "state.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{"format":1,"kind":"team","owner":"org","name":"root","cloned_at":"2026-04-18T00:00:00Z","root_resource":{"kind":"team","owner":"org","name":"root","local_path":".clier/org.root.team","editable":true},"teams":[],"tracked_resources":[],"generated_files":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	localFS.removeErr = errors.New("boom")

	repo := &runRemoverStub{
		plans: []*apprun.RunPlan{{RunID: "run-1", WorkingCopyPath: base, Status: apprun.StatusStopped}},
	}
	svc := NewService(nil, localFS, nil)

	removedRuns, err := svc.Remove(base, repo)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(removedRuns) != 1 || removedRuns[0] != "run-1" {
		t.Fatalf("removedRuns = %v, want [run-1]", removedRuns)
	}
	if len(repo.staged) != 1 || repo.staged[0].restored {
		t.Fatalf("staged deletions should remain staged on cleanup failure, got %+v", repo.staged)
	}
}

func TestRemove_RejectsRunningRun(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	manifestPath := filepath.Join(base, ".clier", "state.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{"format":1,"kind":"team","owner":"org","name":"root","cloned_at":"2026-04-18T00:00:00Z","root_resource":{"kind":"team","owner":"org","name":"root","local_path":".clier/org.root.team","editable":true},"teams":[],"tracked_resources":[],"generated_files":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := &runRemoverStub{
		plans: []*apprun.RunPlan{{RunID: "run-1", WorkingCopyPath: base, Status: apprun.StatusRunning}},
	}
	svc := NewService(nil, osFS{}, nil)

	_, err := svc.Remove(base, repo)
	var fault *domain.Fault
	if !errors.As(err, &fault) || fault.Kind != domain.KindRunBlocksRemove {
		t.Fatalf("expected run blocks remove fault, got %v", err)
	}
}

type osFS struct{}

func (osFS) EnsureFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}
func (osFS) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (osFS) MkdirAll(path string) error                 { return os.MkdirAll(path, 0o755) }
func (osFS) Stat(path string) (fs.FileInfo, error)      { return os.Stat(path) }
func (osFS) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
func (osFS) MkdirTemp(pattern string) (string, error)   { return os.MkdirTemp("", pattern) }
func (osFS) Rename(oldPath, newPath string) error {
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}
func (osFS) RemoveAll(path string) error { return os.RemoveAll(path) }
