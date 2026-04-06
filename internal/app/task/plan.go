package task

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

const (
	PlaceholderBase        = "{{CLIER_BASE}}"
	PlaceholderMemberspace = "{{CLIER_MEMBERSPACE}}"
	PlaceholderTaskID      = "{{CLIER_TASK_ID}}"
	PlaceholderAuthClaude  = "{{CLIER_AUTH_CLAUDE}}"
)

// resolveTeam loads all referenced resources for every team member.
// This is the resolve phase: ID strings -> actual domain objects.
func (s *Service) resolveTeam(ctx context.Context, team domain.Team) (*domain.ResolvedTeam, error) {
	members := make([]domain.ResolvedMember, 0, len(team.TeamMembers))
	for _, tm := range team.TeamMembers {
		rm, err := s.resolveMember(ctx, &team, tm)
		if err != nil {
			return nil, err
		}
		members = append(members, *rm)
	}
	return &domain.ResolvedTeam{Team: team, Members: members}, nil
}

// resolveMember loads the member spec and all its referenced resources.
func (s *Service) resolveMember(ctx context.Context, team *domain.Team, tm domain.TeamMember) (*domain.ResolvedMember, error) {
	member, err := s.store.GetMember(ctx, tm.MemberID)
	if err != nil {
		return nil, fmt.Errorf("get member %s: %w", tm.MemberID, err)
	}

	var claudeMd *resource.ClaudeMd
	if member.ClaudeMdID != "" {
		cm, err := s.store.GetClaudeMd(ctx, member.ClaudeMdID)
		if err != nil {
			return nil, fmt.Errorf("get claude md for %s: %w", tm.Name, err)
		}
		claudeMd = &cm
	}

	skills := make([]resource.Skill, 0, len(member.SkillIDs))
	for _, id := range member.SkillIDs {
		sk, err := s.store.GetSkill(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get skill %s: %w", id, err)
		}
		skills = append(skills, sk)
	}

	var settings *resource.Settings
	if member.SettingsID != "" {
		st, err := s.store.GetSettings(ctx, member.SettingsID)
		if err != nil {
			return nil, fmt.Errorf("get settings for %s: %w", tm.Name, err)
		}
		settings = &st
	}

	var claudeJson *resource.ClaudeJson
	if member.ClaudeJsonID != "" {
		cj, err := s.store.GetClaudeJson(ctx, member.ClaudeJsonID)
		if err != nil {
			return nil, fmt.Errorf("get claude json for %s: %w", tm.Name, err)
		}
		claudeJson = &cj
	}

	envs := make([]resource.Env, 0, len(member.EnvIDs))
	for _, id := range member.EnvIDs {
		env, err := s.store.GetEnv(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get env %s: %w", id, err)
		}
		envs = append(envs, env)
	}

	var repo *resource.GitRepo
	if member.GitRepoID != "" {
		r, err := s.store.GetGitRepo(ctx, member.GitRepoID)
		if err != nil {
			return nil, fmt.Errorf("get git repo for %s: %w", tm.Name, err)
		}
		repo = &r
	}

	relations := team.MemberRelations(tm.ID)

	return &domain.ResolvedMember{
		TeamMemberID: tm.ID,
		Name:         tm.Name,
		Model:        member.Model,
		Args:         member.Args,
		ClaudeMd:     claudeMd,
		Skills:       skills,
		Settings:     settings,
		ClaudeJson:   claudeJson,
		Envs:         envs,
		Repo:         repo,
		Relations:    relations,
	}, nil
}

// buildPlans constructs MemberPlans from a resolved team.
// This is the build phase: resolved objects -> execution plan with placeholders.
func buildPlans(resolved *domain.ResolvedTeam, taskID string) []domain.MemberPlan {
	nameByID := make(map[string]string, len(resolved.Members))
	for _, rm := range resolved.Members {
		nameByID[rm.TeamMemberID] = rm.Name
	}

	plans := make([]domain.MemberPlan, 0, len(resolved.Members))
	for _, rm := range resolved.Members {
		plan := buildMemberPlan(&rm, nameByID, resolved.Name, taskID)
		plans = append(plans, plan)
	}
	return plans
}

// buildMemberPlan constructs a single MemberPlan from a resolved member.
// This is the transparent facade: each building block and its destination is visible.
func buildMemberPlan(rm *domain.ResolvedMember, nameByID map[string]string, teamName, taskID string) domain.MemberPlan {
	memberspace := fmt.Sprintf("%s/%s/%s", PlaceholderBase, PlaceholderTaskID, rm.TeamMemberID)

	// === CLAUDE.md ===
	systemClaudeMd := buildClierPrompt(teamName, rm.Name, rm.Relations, nameByID) // Clier system
	var userClaudeMd string                                                        // user building block
	if rm.ClaudeMd != nil {
		userClaudeMd = rm.ClaudeMd.Content
	}

	// === settings.json ===
	var userSettings string // user building block (no system injection currently)
	if rm.Settings != nil {
		userSettings = rm.Settings.Content
	}

	// === .claude.json ===
	systemClaudeJson := buildSystemClaudeJson(PlaceholderMemberspace) // Clier system: projects path
	var userClaudeJson string                                         // user building block
	if rm.ClaudeJson != nil {
		userClaudeJson = rm.ClaudeJson.Content
	}

	// === Skills ===
	userSkills := rm.Skills // user building block (no system injection)

	// === Assemble workspace files ===
	files := buildWorkspaceFiles(PlaceholderMemberspace, systemClaudeMd, userClaudeMd, userSettings, systemClaudeJson, userClaudeJson, userSkills)

	// === Command: user building blocks ===
	model := rm.Model
	args := rm.Args
	userEnvs := rm.Envs

	// === Command: Clier system-generated ===
	// (system envs are assembled inside buildCommand -> buildEnv)

	// === Assemble command ===
	cmd := buildCommand(model, args, PlaceholderMemberspace+"/project", teamName, rm.Name, taskID, rm.TeamMemberID, userEnvs)

	launchPath := PlaceholderMemberspace + "/launch.sh"
	files = append(files, domain.FileEntry{Path: launchPath, Content: cmd})

	var gitRepo *domain.GitRepoRef
	if rm.Repo != nil {
		gitRepo = &domain.GitRepoRef{Name: rm.Repo.Name, URL: rm.Repo.URL}
	}

	return domain.MemberPlan{
		TeamMemberID: rm.TeamMemberID,
		MemberName:   rm.Name,
		Terminal:     domain.TerminalPlan{Command: ". " + launchPath},
		Workspace: domain.WorkspacePlan{
			Memberspace: memberspace,
			Files:       files,
			GitRepo:     gitRepo,
		},
	}
}
