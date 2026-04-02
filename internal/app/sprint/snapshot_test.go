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

	t.Run("BuildsAllMembers", func(t *testing.T) {
		snap, err := BuildSprintSnapshot("sprint-1", "/base", team)
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

	t.Run("ClaudeMember_ResolvesCommandAndPaths", func(t *testing.T) {
		snap, err := BuildSprintSnapshot("sprint-1", "/base", team)
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
		} {
			if !strings.Contains(m.Command, want) {
				t.Errorf("Command missing %q:\n%s", want, m.Command)
			}
		}

		// Workspace fields preserved
		if m.Binary != domain.BinaryClaude {
			t.Errorf("Binary = %q, want claude", m.Binary)
		}
		if m.DotConfig["skipDangerousModePermissionPrompt"] != true {
			t.Error("DotConfig not preserved")
		}
		if m.GitRepo == nil || m.GitRepo.URL != "https://github.com/test/repo" {
			t.Error("GitRepo not preserved")
		}

		// Relations preserved
		if len(m.Relations.Workers) != 1 || m.Relations.Workers[0] != "m2" {
			t.Errorf("Relations not preserved: %+v", m.Relations)
		}
	})

	t.Run("CodexMember_UsesDeveloperInstructions", func(t *testing.T) {
		snap, err := BuildSprintSnapshot("sprint-1", "/base", team)
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
	})

	t.Run("EmptyMembers_ReturnsEmptySnapshot", func(t *testing.T) {
		emptyTeam := domain.TeamSnapshot{
			TeamName:     "empty",
			RootMemberID: "root",
			Members:      []domain.MemberSnapshot{},
		}

		snap, err := BuildSprintSnapshot("sprint-1", "/base", emptyTeam)
		if err != nil {
			t.Fatalf("BuildSprintSnapshot: %v", err)
		}
		if len(snap.Members) != 0 {
			t.Errorf("expected 0 members, got %d", len(snap.Members))
		}
	})
}
