package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/domain"
)

func validateWorkingCopy(base string, manifest *appworkspace.Manifest) error {
	if manifest == nil {
		return errors.New("working-copy manifest is missing")
	}
	projections, err := collectRunnableAgents(manifest)
	if err != nil {
		return err
	}
	if len(projections) == 0 {
		return errors.New("team has no runnable agents")
	}
	for _, p := range projections {
		if err := validateAgentCopy(filepath.Join(base, filepath.FromSlash(p.LocalBase)), p); err != nil {
			return err
		}
	}
	return nil
}

func validateAgentCopy(base string, agent runnableAgent) error {
	projection := agent.Projection
	if projection.AgentType == "" || projection.Name == "" {
		return fmt.Errorf("incomplete projection for %s; pull the local clone again", projection.Name)
	}
	materialized, err := appworkspace.IsMaterializedRoot(newFileMaterializer(), newGitRepo(), projection.GitRepoURL, base)
	if err != nil {
		return err
	}
	if !materialized {
		return fmt.Errorf("local clone is incomplete at %s", base)
	}
	profile, err := domain.ProfileFor(projection.AgentType)
	if err != nil {
		return fmt.Errorf("unknown agent type %q for %s", projection.AgentType, projection.Name)
	}

	required := []string{
		filepath.Join(base, profile.InstructionFile),
		filepath.Join(base, ".clier", "work-log-protocol.md"),
		filepath.Join(base, ".clier", appworkspace.TeamProtocolFileName(agent.ID)),
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
		if os.IsNotExist(err) {
			return fmt.Errorf("local clone is missing %s", path)
		}
		return fmt.Errorf("stat working-copy path %s: %w", path, err)
	}
	return nil
}
