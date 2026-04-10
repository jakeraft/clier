package run

type State = RunPlan

func NewState(plan *RunPlan) *State {
	return plan
}

func StatePath(copyRoot, runID string) string {
	return PlanPath(copyRoot, runID)
}

func SaveState(copyRoot string, state *State) error {
	return SavePlan(copyRoot, state.RunID, state)
}

func LoadState(copyRoot, runID string) (*State, error) {
	return LoadPlan(copyRoot, runID)
}

func LoadStateFromPath(path string) (*State, error) {
	return LoadPlanFromPath(path)
}
