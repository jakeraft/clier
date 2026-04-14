package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func manifestPathLabel() string {
	return filepath.ToSlash(filepath.Join(".clier", appworkspace.ManifestFile))
}

func validateWorkingCopy(base string, manifest *appworkspace.Manifest) error {
	if manifest == nil {
		return errors.New("working-copy manifest is missing")
	}
	switch manifest.Kind {
	case string(api.KindMember):
		if manifest.Runtime == nil || manifest.Runtime.Member == nil {
			return fmt.Errorf("manifest in %s is incomplete for member runs", manifestPathLabel())
		}
		return validateMemberCopy(base, manifest.Runtime.Member, "")
	case string(api.KindTeam):
		if manifest.Runtime == nil || manifest.Runtime.Team == nil {
			return fmt.Errorf("manifest in %s is incomplete for team runs", manifestPathLabel())
		}
		if manifest.Runtime.Team.ID == 0 || len(manifest.Runtime.Team.Members) == 0 {
			return fmt.Errorf("manifest in %s is incomplete; pull the local clone again", manifestPathLabel())
		}
		for _, member := range manifest.Runtime.Team.Members {
			memberBase := filepath.Join(base, member.Name)
			if err := validateMemberCopy(memberBase, &appworkspace.MemberRuntimeMetadata{
				ID:         member.MemberID,
				Name:       member.Name,
				Command:    member.Command,
				GitRepoURL: member.GitRepoURL,
			}, member.Name); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported working-copy kind %q", manifest.Kind)
	}
}

func validateMemberCopy(base string, member *appworkspace.MemberRuntimeMetadata, teamMemberName string) error {
	if member == nil {
		return errors.New("working-copy member metadata is missing")
	}
	if member.ID == 0 || member.Name == "" || member.Command == "" {
		return fmt.Errorf("manifest in %s is incomplete; pull the local clone again", manifestPathLabel())
	}
	materialized, err := appworkspace.IsMaterializedRoot(newFileMaterializer(), newGitRepo(), member.GitRepoURL, base)
	if err != nil {
		return err
	}
	if !materialized {
		return fmt.Errorf("local clone is incomplete at %s", base)
	}
	required := []string{
		filepath.Join(base, "CLAUDE.md"),
		filepath.Join(base, ".clier", "work-log-protocol.md"),
		filepath.Join(base, ".claude", "settings.local.json"),
	}
	if teamMemberName != "" {
		required = append(required, filepath.Join(base, ".clier", appworkspace.TeamProtocolFileName(teamMemberName)))
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
