package run

type State = RunPlan

func NewState(plan *RunPlan) *State {
	return plan
}

func StatePath(workspaceBase, runID string) string {
	return PlanPath(workspaceBase, runID)
}

func SaveState(workspaceBase string, state *State) error {
	return SavePlan(workspaceBase, state.RunID, state)
}

func LoadState(workspaceBase, runID string) (*State, error) {
	return LoadPlan(workspaceBase, runID)
}

func LoadStateFromPath(path string) (*State, error) {
	return LoadPlanFromPath(path)
}
