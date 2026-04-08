package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// LocalRefStore persists terminal refs to the local filesystem under
// ~/.clier/refs/. Each run gets a JSON file keyed by member ID.
type LocalRefStore struct {
	dir string
	mu  sync.Mutex
}

// NewLocalRefStore creates a file-backed RefStore.
// If dir is empty, ~/.clier/refs is used.
func NewLocalRefStore(dir string) *LocalRefStore {
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".clier", "refs")
	}
	return &LocalRefStore{dir: dir}
}

// runFile returns the path for a run's refs file.
func (s *LocalRefStore) runFile(runID string) string {
	return filepath.Join(s.dir, runID+".json")
}

// readAll reads all member refs for a run.
func (s *LocalRefStore) readAll(runID string) (map[string]map[string]string, error) {
	data, err := os.ReadFile(s.runFile(runID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("refs not found for run %s", runID)
		}
		return nil, err
	}
	var m map[string]map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// writeAll writes all member refs for a run.
func (s *LocalRefStore) writeAll(runID string, m map[string]map[string]string) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(s.runFile(runID), data, 0o644)
}

func (s *LocalRefStore) SaveRefs(_ context.Context, runID, memberID string, refs map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, _ := s.readAll(runID)
	if m == nil {
		m = make(map[string]map[string]string)
	}
	m[memberID] = refs
	return s.writeAll(runID, m)
}

func (s *LocalRefStore) GetRefs(_ context.Context, runID, memberID string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := s.readAll(runID)
	if err != nil {
		return nil, err
	}
	refs, ok := m[memberID]
	if !ok {
		return nil, errors.New("member refs not found")
	}
	return refs, nil
}

func (s *LocalRefStore) GetRunRefs(_ context.Context, runID string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, err := s.readAll(runID)
	if err != nil {
		return nil, err
	}
	// Return the first member's refs (contains session name).
	for _, refs := range m {
		return refs, nil
	}
	return nil, errors.New("no refs found for run")
}

func (s *LocalRefStore) DeleteRefs(_ context.Context, runID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.runFile(runID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
