package sprint

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildMemberPrompt(t *testing.T) {
	t.Run("MultiplePromptsConcatenated", func(t *testing.T) {
		team := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{
					MemberID:   "m-1",
					MemberName: "Agent",
					SystemPrompts: []domain.PromptSnapshot{
						{Name: "protocol", Prompt: "# Team Protocol"},
						{Name: "style", Prompt: "Be concise."},
					},
				},
			},
		}

		got, err := BuildMemberPrompt(team, "m-1")
		if err != nil {
			t.Fatal(err)
		}

		for _, want := range []string{
			"# Team Protocol",
			"Be concise.",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}

		// first prompt comes before second
		if strings.Index(got, "# Team Protocol") > strings.Index(got, "Be concise.") {
			t.Errorf("prompts should be in order:\n%s", got)
		}
	})

	t.Run("SinglePrompt", func(t *testing.T) {
		team := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{
					MemberID:   "m-1",
					MemberName: "Solo",
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
			t.Errorf("expected single prompt, got:\n%s", got)
		}
	})

	t.Run("NoPrompts_ReturnsEmpty", func(t *testing.T) {
		team := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{MemberID: "m-1", MemberName: "Agent"},
			},
		}

		got, err := BuildMemberPrompt(team, "m-1")
		if err != nil {
			t.Fatal(err)
		}

		if got != "" {
			t.Errorf("expected empty, got:\n%s", got)
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
