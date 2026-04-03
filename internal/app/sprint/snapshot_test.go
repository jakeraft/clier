package sprint

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildSprintSnapshot(t *testing.T) {
	team := domain.TeamSnapshot{
		TeamName:     "test-team",
		RootMemberID: "m1",
		Members: []domain.MemberSnapshot{
			{
				MemberID:   "m1",
				MemberName: "leader",
				Binary:     domain.BinaryClaude,
				Model:      "claude-sonnet-4-6",
				SystemArgs: []string{"--dangerously-skip-permissions"},
				CustomArgs: []string{},
				DotConfig:  domain.DotConfig{"skipDangerousModePermissionPrompt": true},
				SystemPrompts: []domain.PromptSnapshot{
					{Name: "protocol", Prompt: "You are a team leader."},
				},
				Envs: []domain.EnvSnapshot{
					{Name: "token", Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
				},
				GitRepo:   &domain.GitRepoSnapshot{Name: "repo", URL: "https://github.com/test/repo"},
				Relations: domain.MemberRelations{Workers: []string{"m2"}},
			},
			{
				MemberID:   "m2",
				MemberName: "worker",
				Binary:     domain.BinaryCodex,
				Model:      "gpt-5.4",
				SystemArgs: []string{"--dangerously-bypass-approvals-and-sandbox"},
				CustomArgs: []string{},
				DotConfig:  domain.DotConfig{"sandbox_mode": "danger-full-access"},
				SystemPrompts: []domain.PromptSnapshot{
					{Name: "role", Prompt: "You are a worker."},
				},
				Envs:      nil,
				GitRepo:   nil,
				Relations: domain.MemberRelations{Leaders: []string{"m1"}},
			},
		},
	}

	tokens := map[domain.CliBinary]string{
		domain.BinaryClaude: "test-claude-token",
	}

	t.Run("BuildsAllMembers", func(t *testing.T) {
		snap, err := BuildSprintSnapshot("sprint-1", "/base", team, tokens)
		if err != nil {
			t.Fatalf("BuildSprintSnapshot: %v", err)
		}

		if snap.TeamName != "test-team" {
			t.Errorf("TeamName = %q, want %q", snap.TeamName, "test-team")
		}
		if snap.RootMemberID != "m1" {
			t.Errorf("RootMemberID = %q, want %q", snap.RootMemberID, "m1")
		}
		if len(snap.Members) != 2 {
			t.Fatalf("Members count = %d, want 2", len(snap.Members))
		}
	})

	t.Run("ClaudeMember_ResolvesCommandPathsAndFiles", func(t *testing.T) {
		snap, err := BuildSprintSnapshot("sprint-1", "/base", team, tokens)
		if err != nil {
			t.Fatalf("BuildSprintSnapshot: %v", err)
		}

		m := snap.Members[0] // leader (claude)

		// Paths
		if m.Home != "/base/sprint-1/m1" {
			t.Errorf("Home = %q, want /base/sprint-1/m1", m.Home)
		}
		if m.WorkDir != "/base/sprint-1/m1/project" {
			t.Errorf("WorkDir = %q, want /base/sprint-1/m1/project", m.WorkDir)
		}

		// Command contains resolved prompt, env, binary
		for _, want := range []string{
			"claude",
			"--append-system-prompt",
			"You are a team leader.",
			"GITHUB_TOKEN",
			"CLIER_SPRINT_ID='sprint-1'",
			"CLIER_MEMBER_ID='m1'",
			"CLAUDE_CODE_OAUTH_TOKEN",
		} {
			if !strings.Contains(m.Command, want) {
				t.Errorf("Command missing %q:\n%s", want, m.Command)
			}
		}

		// Files resolved from config.go
		if len(m.Files) == 0 {
			t.Fatal("expected Files to have entries for Claude member")
		}
		hasSettings := false
		for _, f := range m.Files {
			if f.Path == ".claude/settings.json" {
				hasSettings = true
			}
		}
		if !hasSettings {
			t.Error("expected .claude/settings.json in Files")
		}

		// GitRepo preserved
		if m.GitRepo == nil || m.GitRepo.URL != "https://github.com/test/repo" {
			t.Error("GitRepo not preserved")
		}
	})

	t.Run("CodexMember_UsesDeveloperInstructionsAndFiles", func(t *testing.T) {
		snap, err := BuildSprintSnapshot("sprint-1", "/base", team, tokens)
		if err != nil {
			t.Fatalf("BuildSprintSnapshot: %v", err)
		}

		m := snap.Members[1] // worker (codex)

		if !strings.Contains(m.Command, "developer_instructions=") {
			t.Errorf("Codex command should use developer_instructions:\n%s", m.Command)
		}
		if m.GitRepo != nil {
			t.Errorf("GitRepo should be nil for worker")
		}

		// Files resolved from config.go
		if len(m.Files) == 0 {
			t.Fatal("expected Files to have entries for Codex member")
		}
		hasConfig := false
		for _, f := range m.Files {
			if f.Path == ".codex/config.toml" {
				hasConfig = true
			}
		}
		if !hasConfig {
			t.Error("expected .codex/config.toml in Files")
		}
	})

	t.Run("EmptyMembers_ReturnsEmptySnapshot", func(t *testing.T) {
		emptyTeam := domain.TeamSnapshot{
			TeamName:     "empty",
			RootMemberID: "root",
			Members:      []domain.MemberSnapshot{},
		}

		snap, err := BuildSprintSnapshot("sprint-1", "/base", emptyTeam, nil)
		if err != nil {
			t.Fatalf("BuildSprintSnapshot: %v", err)
		}
		if len(snap.Members) != 0 {
			t.Errorf("expected 0 members, got %d", len(snap.Members))
		}
	})
}
