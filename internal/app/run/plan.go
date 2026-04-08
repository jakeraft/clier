package run

import (
	"context"
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
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

	var claudeSettings *resource.ClaudeSettings
	if member.ClaudeSettingsID != "" {
		cs, err := s.store.GetClaudeSettings(ctx, member.ClaudeSettingsID)
		if err != nil {
			return nil, fmt.Errorf("get claude settings for %s: %w", tm.Name, err)
		}
		claudeSettings = &cs
	}

	relations := team.MemberRelations(tm.ID)

	return &domain.ResolvedMember{
		TeamMemberID:   tm.ID,
		Name:           tm.Name,
		Command:        member.Command,
		ClaudeMd:       claudeMd,
		Skills:         skills,
		ClaudeSettings: claudeSettings,
		GitRepoURL:     member.GitRepoURL,
		Relations:      relations,
	}, nil
}

// buildPlans constructs MemberPlans from a resolved team.
// This is the build phase: resolved objects -> execution plan with concrete paths.
func buildPlans(resolved *domain.ResolvedTeam, base, runID string, runtimes map[string]AgentRuntime) []domain.MemberPlan {
	nameByID := make(map[string]string, len(resolved.Members))
	for _, rm := range resolved.Members {
		nameByID[rm.TeamMemberID] = rm.Name
	}

	plans := make([]domain.MemberPlan, 0, len(resolved.Members))
	for _, rm := range resolved.Members {
		plan := buildMemberPlan(&rm, nameByID, resolved.Name, base, runID, runtimes)
		plans = append(plans, plan)
	}
	return plans
}

// buildMemberPlan constructs a single MemberPlan from a resolved member.
// This is the transparent facade: each building block and its destination is visible.
func buildMemberPlan(rm *domain.ResolvedMember, nameByID map[string]string, teamName, base, runID string, runtimes map[string]AgentRuntime) domain.MemberPlan {
	memberspace := fmt.Sprintf("%s/%s/%s", base, runID, rm.TeamMemberID)

	// Detect agent type from the first word of Command (e.g. "claude", "codex").
	binary := strings.Fields(rm.Command)[0]
	rt := runtimes[binary]
	if rt == nil {
		rt = runtimes["claude"]
	}

	// === Instruction file (e.g. CLAUDE.md) ===
	systemClaudeMd := buildClierPrompt(teamName, rm.Name, rm.Relations, nameByID) // Clier system
	var userClaudeMd string                                                        // user building block
	if rm.ClaudeMd != nil {
		userClaudeMd = rm.ClaudeMd.Content
	}

	// === settings (e.g. settings.json) ===
	var userClaudeSettings string // user building block (no system injection currently)
	if rm.ClaudeSettings != nil {
		userClaudeSettings = rm.ClaudeSettings.Content
	}

	// === project config (e.g. .claude.json) ===
	systemProjectConfig := rt.SystemConfig(memberspace) // runtime-provided system config

	// === Skills ===
	userSkills := rm.Skills // user building block (no system injection)

	// === Assemble workspace files ===
	files := buildWorkspaceFiles(rt, memberspace, systemClaudeMd, userClaudeMd, userClaudeSettings, systemProjectConfig, userSkills)

	// === Command: user-provided command string ===
	// === Command: Clier system-generated ===
	// (system envs are assembled inside buildCommand -> buildEnv)

	// === Assemble command ===
	cmd := buildCommand(rt, rm.Command, memberspace+"/project",
		memberspace, teamName, rm.Name, runID, rm.TeamMemberID)

	launchPath := memberspace + "/launch.sh"
	files = append(files, domain.FileEntry{Path: launchPath, Content: cmd})

	return domain.MemberPlan{
		TeamMemberID: rm.TeamMemberID,
		MemberName:   rm.Name,
		Terminal:     domain.TerminalPlan{Command: ". " + launchPath},
		Workspace: domain.WorkspacePlan{
			Memberspace: memberspace,
			Files:       files,
			GitRepoURL:  rm.GitRepoURL,
		},
	}
}
