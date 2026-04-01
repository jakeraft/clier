package sprint

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildMemberPrompt(t *testing.T) {
	t.Run("ProtocolAppendedAfterSystemPrompts", func(t *testing.T) {
		team := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{
					MemberID:   "m-1",
					MemberName: "Agent",
					SystemPrompts: []domain.PromptSnapshot{
						{Name: "style", Prompt: "Be concise."},
						{Name: "testing", Prompt: "Write tests."},
					},
					Protocol: domain.DefaultProtocol,
				},
			},
		}

		got, err := BuildMemberPrompt(team, "m-1")
		if err != nil {
			t.Fatal(err)
		}

		for _, want := range []string{
			"Be concise.",
			"Write tests.",
			"## Team Protocol",
			"clier sprint context",
			"clier message send",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}

		// system prompts come before protocol
		protoIdx := strings.Index(got, "## Team Protocol")
		if idx := strings.Index(got, "Be concise."); idx > protoIdx {
			t.Errorf("system prompt should precede protocol:\n%s", got)
		}
	})

	t.Run("NoSystemPrompts_ProtocolOnly", func(t *testing.T) {
		team := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{
					MemberID:   "m-1",
					MemberName: "Solo",
					Protocol:   domain.DefaultProtocol,
				},
			},
		}

		got, err := BuildMemberPrompt(team, "m-1")
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(got, "## Team Protocol") {
			t.Errorf("missing protocol:\n%s", got)
		}
	})

	t.Run("EmptyProtocol_SystemPromptsOnly", func(t *testing.T) {
		team := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{
					MemberID:   "m-1",
					MemberName: "Agent",
					SystemPrompts: []domain.PromptSnapshot{
						{Name: "style", Prompt: "Be concise."},
					},
				},
			},
		}

		got, err := BuildMemberPrompt(team, "m-1")
		if err != nil {
			t.Fatal(err)
		}

		if got != "Be concise." {
			t.Errorf("expected only system prompt, got:\n%s", got)
		}
	})

	t.Run("UnknownMemberID_ReturnsError", func(t *testing.T) {
		team := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{MemberID: "m-1", MemberName: "Agent"},
			},
		}

		_, err := BuildMemberPrompt(team, "nonexistent")
		if err == nil {
			t.Error("should return error for unknown member ID")
		}
	})
}
