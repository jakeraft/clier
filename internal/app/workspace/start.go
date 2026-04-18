package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
)

type RunnableAgent struct {
	ID         string
	Owner      string
	Name       string
	LocalBase  string
	Projection *TeamProjection
}

func CollectRunnableAgents(state *Manifest) ([]RunnableAgent, error) {
	var agents []RunnableAgent

	var walk func(owner string, projection *TeamProjection) error
	walk = func(owner string, projection *TeamProjection) error {
		runnable, err := validateProjectionAgentType(projection.AgentType)
		if err != nil {
			return err
		}
		if runnable {
			team, ok := state.FindTeam(owner, projection.Name)
			if !ok || team.LocalDir == "" {
				return incompleteWorkingCopy("local dir missing for runnable agent " + owner + "/" + projection.Name)
			}
			agents = append(agents, RunnableAgent{
				ID:         ResourceID(owner, projection.Name),
				Owner:      owner,
				Name:       projection.Name,
				LocalBase:  team.LocalDir,
				Projection: projection,
			})
		}

		for _, child := range projection.Children {
			cp, ok := state.FindTeam(child.Owner, child.Name)
			if !ok {
				return incompleteWorkingCopy("team state missing for child " + child.Owner + "/" + child.Name)
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
		return nil, incompleteWorkingCopy("root team state missing for " + state.Owner + "/" + state.Name)
	}
	rootProjection := root.Projection
	if err := walk(state.Owner, &rootProjection); err != nil {
		return nil, err
	}
	return agents, nil
}

func ValidateWorkingCopy(base string, manifest *Manifest, fs FileMaterializer, git GitRepo) error {
	if manifest == nil {
		return incompleteWorkingCopy("working-copy manifest is missing")
	}
	agents, err := CollectRunnableAgents(manifest)
	if err != nil {
		return err
	}
	if len(agents) == 0 {
		return incompleteWorkingCopy("team has no runnable agents")
	}
	for _, agent := range agents {
		if err := validateAgentCopy(base, agent, fs, git); err != nil {
			return err
		}
	}
	return nil
}

func validateProjectionAgentType(agentType string) (bool, error) {
	if _, err := domain.ProfileFor(agentType); err == nil {
		return true, nil
	}
	if remoteapi.IsAbstractTeamAgentType(agentType) {
		return false, nil
	}
	return false, &domain.Fault{
		Kind:    domain.KindUnsupportedKind,
		Subject: map[string]string{"resource_kind": agentType},
	}
}

func validateAgentCopy(base string, agent RunnableAgent, fs FileMaterializer, git GitRepo) error {
	projection := agent.Projection
	if projection.AgentType == "" || projection.Name == "" {
		return incompleteWorkingCopy("incomplete projection for " + projection.Name)
	}
	materialized, err := IsMaterializedRoot(fs, git, projection.GitRepoURL, filepath.Join(base, filepath.FromSlash(agent.LocalBase)))
	if err != nil {
		return err
	}
	if !materialized {
		return incompleteWorkingCopy("local clone is incomplete at " + filepath.Join(base, filepath.FromSlash(agent.LocalBase)))
	}
	profile, err := domain.ProfileFor(projection.AgentType)
	if err != nil {
		return &domain.Fault{
			Kind:    domain.KindUnsupportedKind,
			Subject: map[string]string{"resource_kind": projection.AgentType},
			Cause:   err,
		}
	}

	required := []string{
		filepath.Join(base, filepath.FromSlash(agent.LocalBase), profile.InstructionFile),
		filepath.Join(base, filepath.FromSlash(agent.LocalBase), ".clier", "work-log-protocol.md"),
		filepath.Join(base, filepath.FromSlash(agent.LocalBase), ".clier", TeamProtocolFileName()),
	}
	for _, path := range required {
		if err := requireCopyPath(path); err != nil {
			return err
		}
	}
	return nil
}

func requireCopyPath(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return incompleteWorkingCopy("local clone is missing " + path)
		}
		return fmt.Errorf("stat working-copy path %s: %w", path, err)
	}
	return nil
}

func incompleteWorkingCopy(detail string) *domain.Fault {
	return &domain.Fault{
		Kind:    domain.KindWorkingCopyIncomplete,
		Subject: map[string]string{"detail": detail},
	}
}
