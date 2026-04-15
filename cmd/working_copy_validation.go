package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/domain"
)

func manifestPathLabel() string {
	return filepath.ToSlash(filepath.Join(".clier", appworkspace.ManifestFile))
}

func validateWorkingCopy(base string, manifest *appworkspace.Manifest) error {
	if manifest == nil {
		return errors.New("working-copy manifest is missing")
	}
	if manifest.Runtime == nil || manifest.Runtime.Team == nil {
		return fmt.Errorf("manifest in %s is incomplete for runs", manifestPathLabel())
	}
	if len(manifest.Runtime.Team.Members) == 0 {
		return fmt.Errorf("manifest in %s is incomplete; pull the local clone again", manifestPathLabel())
	}
	for _, member := range manifest.Runtime.Team.Members {
		memberBase := filepath.Join(base, member.Name)
		if err := validateMemberCopy(memberBase, &member, member.Name); err != nil {
			return err
		}
	}
	return nil
}

func validateMemberCopy(base string, member *appworkspace.TeamMemberRuntimeMetadata, teamMemberName string) error {
	if member == nil {
		return errors.New("working-copy member metadata is missing")
	}
	if member.Name == "" || member.Command == "" {
		return fmt.Errorf("manifest in %s is incomplete; pull the local clone again", manifestPathLabel())
	}
	materialized, err := appworkspace.IsMaterializedRoot(newFileMaterializer(), newGitRepo(), member.GitRepoURL, base)
	if err != nil {
		return err
	}
	if !materialized {
		return fmt.Errorf("local clone is incomplete at %s", base)
	}
	profile, err := domain.ProfileFor(member.AgentType)
	if err != nil {
		return fmt.Errorf("unknown agent type %q for member %s", member.AgentType, member.Name)
	}

	required := []string{
		filepath.Join(base, profile.InstructionFile),
		filepath.Join(base, ".clier", "work-log-protocol.md"),
		filepath.Join(base, ".clier", appworkspace.TeamProtocolFileName(teamMemberName)),
	}
	if profile.LocalSettingsFile != "" {
		required = append(required, filepath.Join(base, profile.SettingsDir, profile.LocalSettingsFile))
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
