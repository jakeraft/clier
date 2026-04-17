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
		return incomplete("working-copy manifest is missing")
	}
	projections, err := collectRunnableAgents(manifest)
	if err != nil {
		return err
	}
	if len(projections) == 0 {
		return incomplete("team has no runnable agents")
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
		return incomplete("incomplete projection for " + projection.Name)
	}
	materialized, err := appworkspace.IsMaterializedRoot(newFileMaterializer(), newGitRepo(), projection.GitRepoURL, base)
	if err != nil {
		return err
	}
	if !materialized {
		return incomplete("local clone is incomplete at " + base)
	}
	if _, err := domain.ProfileFor(projection.AgentType); err != nil {
		return &domain.Fault{
			Kind:    domain.KindUnsupportedKind,
			Subject: map[string]string{"resource_kind": projection.AgentType},
			Cause:   err,
		}
	}
	profile, _ := domain.ProfileFor(projection.AgentType)

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
		if errors.Is(err, os.ErrNotExist) {
			return incomplete("local clone is missing " + path)
		}
		return fmt.Errorf("stat working-copy path %s: %w", path, err)
	}
	return nil
}

func incomplete(detail string) *domain.Fault {
	return &domain.Fault{
		Kind:    domain.KindWorkingCopyIncomplete,
		Subject: map[string]string{"detail": detail},
	}
}
