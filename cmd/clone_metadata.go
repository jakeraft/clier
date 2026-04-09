package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	appclone "github.com/jakeraft/clier/internal/app/clone"
)

const (
	resourceKindMember         = "member"
	resourceKindTeam           = "team"
	resourceKindClaudeMd       = "claude-md"
	resourceKindClaudeSettings = "claude-settings"
	resourceKindSkill          = "skill"
)

func resolveCloneFromCWD(expectedKind string) (string, *appclone.CloneMetadata, error) {
	base, err := resolveWorkspaceBase()
	if err != nil {
		return "", nil, err
	}
	for dir := base; ; dir = filepath.Dir(dir) {
		metaPath := filepath.Join(dir, ".clier", appclone.CloneMetadataFile)
		if _, err := os.Stat(metaPath); err == nil {
			meta, err := appclone.LoadCloneMetadata(dir)
			if err != nil {
				return "", nil, err
			}
			if meta.Kind != expectedKind {
				return "", nil, fmt.Errorf("current workspace is %s, not %s", meta.Kind, expectedKind)
			}
			return dir, meta, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", nil, fmt.Errorf("workspace metadata not found in current directory")
}

func cloneMetadataExists(base string) bool {
	_, err := os.Stat(filepath.Join(base, ".clier", appclone.CloneMetadataFile))
	return err == nil
}

func buildMemberCloneMetadata(client *api.Client, owner, name string) (*appclone.CloneMetadata, error) {
	member, err := client.GetMember(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get member %s/%s: %w", owner, name, err)
	}

	meta := &appclone.CloneMetadata{
		Kind:          resourceKindMember,
		Owner:         member.OwnerLogin,
		Name:          member.Name,
		Materializer:  "local-git",
		GitRepoURL:    member.GitRepoURL,
		LatestVersion: member.LatestVersion,
		ClonedAt:      time.Now().UTC(),
		Workspace: &appclone.WorkspaceMetadata{
			Member: &appclone.MemberWorkspaceMetadata{
				ID:         member.ID,
				Name:       member.Name,
				Command:    member.Command,
				GitRepoURL: member.GitRepoURL,
			},
		},
	}
	resources, err := memberMaterializedResources(client, "", member)
	if err != nil {
		return nil, err
	}
	meta.Resources = resources
	sortCloneResources(meta.Resources)
	return meta, nil
}

func buildTeamCloneMetadata(client *api.Client, owner, name string) (*appclone.CloneMetadata, error) {
	team, err := client.GetTeam(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get team %s/%s: %w", owner, name, err)
	}

	meta := &appclone.CloneMetadata{
		Kind:          resourceKindTeam,
		Owner:         team.OwnerLogin,
		Name:          team.Name,
		Materializer:  "local-git",
		LatestVersion: team.LatestVersion,
		ClonedAt:      time.Now().UTC(),
		Workspace: &appclone.WorkspaceMetadata{
			Team: &appclone.TeamWorkspaceMetadata{
				ID:   team.ID,
				Name: team.Name,
			},
		},
	}

	for _, tm := range team.TeamMembers {
		member, err := client.GetMember(tm.Member.Owner, tm.Member.Name)
		if err != nil {
			return nil, fmt.Errorf("get member %s/%s: %w", tm.Member.Owner, tm.Member.Name, err)
		}
		meta.Resources = append(meta.Resources, appclone.CloneResourceMetadata{
			Kind:          resourceKindMember,
			Owner:         member.OwnerLogin,
			Name:          member.Name,
			GitRepoURL:    member.GitRepoURL,
			LocalPath:     filepath.ToSlash(tm.Name),
			LatestVersion: member.LatestVersion,
		})
		meta.Workspace.Team.Members = append(meta.Workspace.Team.Members, appclone.TeamMemberWorkspaceMetadata{
			TeamMemberID: tm.ID,
			Name:         tm.Name,
			Command:      member.Command,
			GitRepoURL:   member.GitRepoURL,
		})
		resources, err := memberMaterializedResources(client, tm.Name, member)
		if err != nil {
			return nil, err
		}
		meta.Resources = append(meta.Resources, resources...)
	}

	sortCloneResources(meta.Resources)
	return meta, nil
}

func memberMaterializedResources(client *api.Client, memberDir string, member *api.MemberResponse) ([]appclone.CloneResourceMetadata, error) {
	var resources []appclone.CloneResourceMetadata

	if member.ClaudeMd != nil {
		claudeMd, err := client.GetClaudeMd(member.ClaudeMd.Owner, member.ClaudeMd.Name)
		if err != nil {
			return nil, fmt.Errorf("get claude md %s/%s: %w", member.ClaudeMd.Owner, member.ClaudeMd.Name, err)
		}
		resources = append(resources, appclone.CloneResourceMetadata{
			Kind:          resourceKindClaudeMd,
			Owner:         claudeMd.OwnerLogin,
			Name:          claudeMd.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberDir, "CLAUDE.md")),
			LatestVersion: claudeMd.LatestVersion,
		})
	}

	if member.ClaudeSettings != nil {
		settings, err := client.GetClaudeSettings(member.ClaudeSettings.Owner, member.ClaudeSettings.Name)
		if err != nil {
			return nil, fmt.Errorf("get claude settings %s/%s: %w", member.ClaudeSettings.Owner, member.ClaudeSettings.Name, err)
		}
		resources = append(resources, appclone.CloneResourceMetadata{
			Kind:          resourceKindClaudeSettings,
			Owner:         settings.OwnerLogin,
			Name:          settings.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberDir, ".claude", "settings.json")),
			LatestVersion: settings.LatestVersion,
		})
	}

	for _, skillRef := range member.Skills {
		skill, err := client.GetSkill(skillRef.Owner, skillRef.Name)
		if err != nil {
			return nil, fmt.Errorf("get skill %s/%s: %w", skillRef.Owner, skillRef.Name, err)
		}
		resources = append(resources, appclone.CloneResourceMetadata{
			Kind:          resourceKindSkill,
			Owner:         skill.OwnerLogin,
			Name:          skill.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberDir, ".claude", "skills", skill.Name, "SKILL.md")),
			LatestVersion: skill.LatestVersion,
		})
	}

	return resources, nil
}

func sortCloneResources(resources []appclone.CloneResourceMetadata) {
	sort.Slice(resources, func(i, j int) bool {
		left := cloneMetadataResourceKey(resources[i].Kind, resources[i].Owner, resources[i].Name, resources[i].LocalPath)
		right := cloneMetadataResourceKey(resources[j].Kind, resources[j].Owner, resources[j].Name, resources[j].LocalPath)
		return left < right
	})
}

func cloneMetadataResourceKey(kind, owner, name, localPath string) string {
	return kind + "|" + owner + "|" + name + "|" + filepath.ToSlash(filepath.Clean(localPath))
}
