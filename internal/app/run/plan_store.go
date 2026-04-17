package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RunsDirName is the subdirectory under WorkspaceDir that holds all run
// plans. The leading dot prevents collision with team owner names.
const RunsDirName = ".runs"

// SavePlan writes the RunPlan to <runsDir>/<runID>.json.
func SavePlan(runsDir, runID string, plan *RunPlan) error {
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return fmt.Errorf("create runs dir: %w", err)
	}

	path := PlanPath(runsDir, runID)
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// PlanPath returns the absolute path of a run plan file under the runs dir.
func PlanPath(runsDir, runID string) string {
	return filepath.Join(runsDir, runID+".json")
}

// LoadPlan reads a saved RunPlan from <runsDir>/<runID>.json.
func LoadPlan(runsDir, runID string) (*RunPlan, error) {
	return LoadPlanFromPath(PlanPath(runsDir, runID))
}

// LoadPlanFromPath reads a saved RunPlan from an absolute file path.
func LoadPlanFromPath(path string) (*RunPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plan: %w", err)
	}
	var plan RunPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("unmarshal plan: %w", err)
	}
	return &plan, nil
}

// ListPlans returns every loadable run plan in runsDir. Corrupt or
// unparseable files are silently skipped so a single bad file cannot
// brick scanning commands. A missing runsDir returns an empty slice
// (no error). Callers that need filtering (by working copy, status,
// etc.) should iterate the returned slice — this is the single
// scan-policy entry point for the runs directory.
func ListPlans(runsDir string) ([]*RunPlan, error) {
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*RunPlan{}, nil
		}
		return nil, fmt.Errorf("read runs dir: %w", err)
	}
	plans := make([]*RunPlan, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		plan, err := LoadPlanFromPath(filepath.Join(runsDir, name))
		if err != nil {
			continue
		}
		plans = append(plans, plan)
	}
	return plans, nil
}
