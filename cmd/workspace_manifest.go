package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

const (
	resourceKindMember         = "member"
	resourceKindTeam           = "team"
	resourceKindClaudeMd       = "claude-md"
	resourceKindClaudeSettings = "claude-settings"
	resourceKindSkill          = "skill"
)

func resolveWorkspaceFromCWD(expectedKind string) (string, *appworkspace.Manifest, error) {
	base, err := resolveWorkspaceBase()
	if err != nil {
		return "", nil, err
	}
	for dir := base; ; dir = filepath.Dir(dir) {
		if _, err := appworkspace.FindManifestPath(dir); err == nil {
			meta, err := appworkspace.LoadManifest(dir)
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
	return "", nil, errors.New("workspace metadata not found in current directory")
}

func buildMemberManifest(client *api.Client, owner, name string) (*appworkspace.Manifest, error) {
	member, err := client.GetMember(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get member %s/%s: %w", owner, name, err)
	}

	meta := &appworkspace.Manifest{
		Kind:          resourceKindMember,
		Owner:         member.OwnerLogin,
		Name:          member.Name,
		Materializer:  "local-git",
		GitRepoURL:    member.GitRepoURL,
		LatestVersion: member.LatestVersion,
		DownloadedAt:  time.Now().UTC(),
		Workspace: &appworkspace.WorkspaceMetadata{
			Member: &appworkspace.MemberWorkspaceMetadata{
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
	sortWorkspaceResources(meta.Resources)
	return meta, nil
}

func buildTeamManifest(client *api.Client, owner, name string) (*appworkspace.Manifest, error) {
	team, err := client.GetTeam(owner, name)
	if err != nil {
		return nil, fmt.Errorf("get team %s/%s: %w", owner, name, err)
	}

	meta := &appworkspace.Manifest{
		Kind:          resourceKindTeam,
		Owner:         team.OwnerLogin,
		Name:          team.Name,
		Materializer:  "local-git",
		LatestVersion: team.LatestVersion,
		DownloadedAt:  time.Now().UTC(),
		Workspace: &appworkspace.WorkspaceMetadata{
			Team: &appworkspace.TeamWorkspaceMetadata{
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
		meta.Resources = append(meta.Resources, appworkspace.ResourceManifest{
			Kind:          resourceKindMember,
			Owner:         member.OwnerLogin,
			Name:          member.Name,
			GitRepoURL:    member.GitRepoURL,
			LocalPath:     filepath.ToSlash(tm.Name),
			LatestVersion: member.LatestVersion,
		})
		meta.Workspace.Team.Members = append(meta.Workspace.Team.Members, appworkspace.TeamMemberWorkspaceMetadata{
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

	sortWorkspaceResources(meta.Resources)
	return meta, nil
}

func memberMaterializedResources(client *api.Client, memberDir string, member *api.MemberResponse) ([]appworkspace.ResourceManifest, error) {
	var resources []appworkspace.ResourceManifest

	if member.ClaudeMd != nil {
		claudeMd, err := client.GetClaudeMd(member.ClaudeMd.Owner, member.ClaudeMd.Name)
		if err != nil {
			return nil, fmt.Errorf("get claude md %s/%s: %w", member.ClaudeMd.Owner, member.ClaudeMd.Name, err)
		}
		resources = append(resources, appworkspace.ResourceManifest{
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
		resources = append(resources, appworkspace.ResourceManifest{
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
		resources = append(resources, appworkspace.ResourceManifest{
			Kind:          resourceKindSkill,
			Owner:         skill.OwnerLogin,
			Name:          skill.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(memberDir, ".claude", "skills", skill.Name, "SKILL.md")),
			LatestVersion: skill.LatestVersion,
		})
	}

	return resources, nil
}

func sortWorkspaceResources(resources []appworkspace.ResourceManifest) {
	sort.Slice(resources, func(i, j int) bool {
		left := workspaceResourceKey(resources[i].Kind, resources[i].Owner, resources[i].Name, resources[i].LocalPath)
		right := workspaceResourceKey(resources[j].Kind, resources[j].Owner, resources[j].Name, resources[j].LocalPath)
		return left < right
	})
}

func workspaceResourceKey(kind, owner, name, localPath string) string {
	return kind + "|" + owner + "|" + name + "|" + filepath.ToSlash(filepath.Clean(localPath))
}
