package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SavePlan writes the RunPlan to {copyRoot}/.clier/{runID}.json.
func SavePlan(copyRoot, runID string, plan *RunPlan) error {
	dir := filepath.Join(copyRoot, ".clier")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create plan dir: %w", err)
	}

	path := PlanPath(copyRoot, runID)
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// PlanPath returns the absolute path of a run plan file under a local clone.
func PlanPath(copyRoot, runID string) string {
	return filepath.Join(copyRoot, ".clier", runID+".json")
}

// LoadPlan reads a saved RunPlan from {copyRoot}/.clier/{runID}.json.
func LoadPlan(copyRoot, runID string) (*RunPlan, error) {
	return LoadPlanFromPath(PlanPath(copyRoot, runID))
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
