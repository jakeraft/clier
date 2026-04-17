package cmd

import (
	"fmt"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/domain"
)

type runnableAgent struct {
	ID         string
	Owner      string
	Name       string
	LocalBase  string
	Projection *appworkspace.TeamProjection
}

// collectRunnableAgents walks the team projection tree recursively and returns
// every node that has a known agent profile.
func collectRunnableAgents(state *appworkspace.Manifest) ([]runnableAgent, error) {
	var agents []runnableAgent

	var walk func(owner string, projection *appworkspace.TeamProjection) error
	walk = func(owner string, projection *appworkspace.TeamProjection) error {
		if _, err := domain.ProfileFor(projection.AgentType); err == nil {
			team, ok := state.FindTeam(owner, projection.Name)
			if !ok || team.LocalDir == "" {
				return fmt.Errorf("local dir missing for runnable agent %s/%s", owner, projection.Name)
			}
			agents = append(agents, runnableAgent{
				ID:         appworkspace.ResourceID(owner, projection.Name),
				Owner:      owner,
				Name:       projection.Name,
				LocalBase:  team.LocalDir,
				Projection: projection,
			})
		}

		for _, child := range projection.Children {
			cp, ok := state.FindTeam(child.Owner, child.Name)
			if !ok {
				return fmt.Errorf("team state missing for child %s/%s", child.Owner, child.Name)
			}
			projection := cp.Projection
			if err := walk(child.Owner, &projection); err != nil {
				return err
			}
		}
		return nil
	}

	root, ok := state.FindTeam(state.Owner, state.Name)
	if !ok {
		return nil, fmt.Errorf("root team state missing for %s/%s", state.Owner, state.Name)
	}
	rootProjection := root.Projection
	if err := walk(state.Owner, &rootProjection); err != nil {
		return nil, err
	}
	return agents, nil
}
