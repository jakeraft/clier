package sprint

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildMemberPrompt(t *testing.T) {
	t.Run("BundledProtocolAppendedAfterUserPrompts", func(t *testing.T) {
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
						{Name: "Team Protocol", Prompt: domain.DefaultProtocol},
					},
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

		// user prompts come before bundled protocol
		protoIdx := strings.Index(got, "## Team Protocol")
		if idx := strings.Index(got, "Be concise."); idx > protoIdx {
			t.Errorf("user prompt should precede protocol:\n%s", got)
		}
	})

	t.Run("ProtocolOnlyMember", func(t *testing.T) {
		team := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{
					MemberID:   "m-1",
					MemberName: "Solo",
					SystemPrompts: []domain.PromptSnapshot{
						{Name: "Team Protocol", Prompt: domain.DefaultProtocol},
					},
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

	t.Run("NoPrompts_ReturnsEmpty", func(t *testing.T) {
		team := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{
					MemberID:   "m-1",
					MemberName: "Agent",
				},
			},
		}

		got, err := BuildMemberPrompt(team, "m-1")
		if err != nil {
			t.Fatal(err)
		}

		if got != "" {
			t.Errorf("expected empty prompt, got:\n%s", got)
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
