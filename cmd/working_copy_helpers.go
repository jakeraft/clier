package cmd

import (
	"errors"
	"fmt"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/domain"
)

func errNotInWorkingCopy() error {
	return errors.New("no local clone found in the current directory or its ancestors")
}

// collectRunnableAgents walks the team projection tree and returns every node
// that has a known agent profile. Uniform for leaf and composite teams —
// no special casing. The root itself is included when ProfileFor succeeds.
func collectRunnableAgents(fs appworkspace.FileMaterializer, copyRoot string, root *appworkspace.TeamProjection) ([]*appworkspace.TeamProjection, error) {
	var agents []*appworkspace.TeamProjection

	// Root node.
	if _, err := domain.ProfileFor(root.AgentType); err == nil {
		agents = append(agents, root)
	}

	// Children.
	for _, child := range root.Children {
		cp, err := appworkspace.LoadTeamProjection(fs, appworkspace.ChildTeamProjectionPath(copyRoot, child.Name))
		if err != nil {
			return nil, fmt.Errorf("load child projection %s: %w", child.Name, err)
		}
		if _, err := domain.ProfileFor(cp.AgentType); err == nil {
			agents = append(agents, cp)
		}
	}

	return agents, nil
}
