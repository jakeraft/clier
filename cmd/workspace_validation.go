package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func workspaceMetadataPathLabel() string {
	return filepath.ToSlash(filepath.Join(".clier", appworkspace.WorkspaceMetadataFile))
}

func validateDownloadedWorkspace(base string, meta *appworkspace.Manifest) error {
	if meta == nil {
		return errors.New("workspace metadata is missing")
	}
	switch meta.Kind {
	case resourceKindMember:
		if meta.Workspace == nil || meta.Workspace.Member == nil {
			return fmt.Errorf("workspace metadata in %s is incomplete for member runs", workspaceMetadataPathLabel())
		}
		return validateMemberWorkspace(base, meta.Workspace.Member, "")
	case resourceKindTeam:
		if meta.Workspace == nil || meta.Workspace.Team == nil {
			return fmt.Errorf("workspace metadata in %s is incomplete for team runs", workspaceMetadataPathLabel())
		}
		if meta.Workspace.Team.ID == 0 || len(meta.Workspace.Team.Members) == 0 {
			return fmt.Errorf("workspace metadata in %s is incomplete; download the resource again", workspaceMetadataPathLabel())
		}
		for _, member := range meta.Workspace.Team.Members {
			memberBase := filepath.Join(base, member.Name)
			if err := validateMemberWorkspace(memberBase, &appworkspace.MemberWorkspaceMetadata{
				ID:         member.TeamMemberID,
				Name:       member.Name,
				Command:    member.Command,
				GitRepoURL: member.GitRepoURL,
			}, member.Name); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported workspace kind %q", meta.Kind)
	}
}

func validateMemberWorkspace(base string, member *appworkspace.MemberWorkspaceMetadata, teamMemberName string) error {
	if member == nil {
		return errors.New("workspace member metadata is missing")
	}
	if member.ID == 0 || member.Name == "" || member.Command == "" {
		return fmt.Errorf("workspace metadata in %s is incomplete; download the resource again", workspaceMetadataPathLabel())
	}
	prepared, err := appworkspace.IsPreparedRoot(member.GitRepoURL, base)
	if err != nil {
		return err
	}
	if !prepared {
		return fmt.Errorf("downloaded workspace is incomplete at %s", base)
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
		if err := requireWorkspacePath(path); err != nil {
			return err
		}
	}
	return nil
}

func requireWorkspacePath(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("downloaded workspace is missing %s", path)
		}
		return fmt.Errorf("stat workspace path %s: %w", path, err)
	}
	return nil
}
