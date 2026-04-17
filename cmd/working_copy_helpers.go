package cmd

import (
	"errors"
	"os"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/domain"
)

// errNoWorkingCopy returns a uniform "not cloned yet" Fault used by
// status / pull / push / run start so the user always gets the same
// remediation hint regardless of which command they tried.
func errNoWorkingCopy(owner, name, base string) error {
	return &domain.Fault{
		Kind: domain.KindWorkingCopyMissing,
		Subject: map[string]string{
			"path":  base,
			"owner": owner,
			"name":  name,
		},
	}
}

// classifyWorkingCopyError wraps the raw os.ErrNotExist from manifest
// loading into the friendly errNoWorkingCopy error. Other errors pass
// through untouched. Uses errors.Is so wrapped errors (e.g.,
// "read manifest: %w") still match.
func classifyWorkingCopyError(owner, name, base string, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return errNoWorkingCopy(owner, name, base)
	}
	return err
}

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
				return &domain.Fault{
					Kind:    domain.KindWorkingCopyIncomplete,
					Subject: map[string]string{"detail": "local dir missing for runnable agent " + owner + "/" + projection.Name},
				}
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
				return &domain.Fault{
					Kind:    domain.KindWorkingCopyIncomplete,
					Subject: map[string]string{"detail": "team state missing for child " + child.Owner + "/" + child.Name},
				}
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
		return nil, &domain.Fault{
			Kind:    domain.KindWorkingCopyIncomplete,
			Subject: map[string]string{"detail": "root team state missing for " + state.Owner + "/" + state.Name},
		}
	}
	rootProjection := root.Projection
	if err := walk(state.Owner, &rootProjection); err != nil {
		return nil, err
	}
	return agents, nil
}
