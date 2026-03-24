package sprint

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func newTestTeam(rootID string, members []domain.MemberSnapshot) domain.TeamSnapshot {
	return domain.TeamSnapshot{
		TeamName:     "MyTeam",
		RootMemberID: rootID,
		Members:      members,
	}
}

func TestBuildMemberPrompt(t *testing.T) {
	t.Run("RootWithWorkers_HasWorkerGuidanceOnly", func(t *testing.T) {
		// given: Boss is root member with one worker (no leader)
		team := newTestTeam("boss-1", []domain.MemberSnapshot{
			{MemberID: "boss-1", MemberName: "Boss", Relations: domain.MemberRelations{Workers: []string{"worker-1"}}},
			{MemberID: "worker-1", MemberName: "Writer"},
		})

		// when
		got, err := BuildMemberPrompt(team, "boss-1")
		if err != nil {
			t.Fatal(err)
		}

		// then: prompt is
		//   ## Team Protocol
		//
		//   You are "Boss", part of team "MyTeam".
		//
		//   | Role   | Name   | ID       |
		//   | Worker | Writer | worker-1 |
		//
		//   Delegate sub-tasks to workers. Wait for all responses before wrapping up.
		//
		//   To message a teammate:
		//   clier message send <id> "<message>"
		for _, want := range []string{
			`"Boss"`,
			`"MyTeam"`,
			"| Worker | Writer | worker-1 |",
			"Delegate sub-tasks",
			"clier message send",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}
		// no root-specific guidance
		if strings.Contains(got, "root member") {
			t.Errorf("should not mention root role:\n%s", got)
		}
	})

	t.Run("NonRootWithLeader_MentionsLeaderName", func(t *testing.T) {
		// given: Writer is non-root with leader Editor and peer Reviewer
		team := newTestTeam("leader-1", []domain.MemberSnapshot{
			{MemberID: "leader-1", MemberName: "Editor"},
			{MemberID: "writer-1", MemberName: "Writer", Relations: domain.MemberRelations{
				Leaders: []string{"leader-1"},
				Peers:   []string{"peer-1"},
			}},
			{MemberID: "peer-1", MemberName: "Reviewer"},
		})

		// when
		got, err := BuildMemberPrompt(team, "writer-1")
		if err != nil {
			t.Fatal(err)
		}

		// then: prompt is
		//   ## Team Protocol
		//
		//   You are "Writer", part of team "MyTeam".
		//
		//   | Role   | Name     | ID       |
		//   | Leader | Editor   | leader-1 |
		//   | Peer   | Reviewer | peer-1   |
		//
		//   Your leader is "Editor". Report results to them. Ask them if stuck.
		//
		//   Coordinate with peers when tasks overlap.
		//
		//   Messages from teammates appear directly in your conversation.
		//   clier message send <id> "<message>"
		for _, want := range []string{
			"| Leader | Editor | leader-1 |",
			"| Peer | Reviewer | peer-1 |",
			`Your leader is "Editor"`,
			"Coordinate with peers",
			"clier message send",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}
	})

	t.Run("NoRelations_HasUserGuidanceOnly", func(t *testing.T) {
		// given: Solo is root with no relations
		team := newTestTeam("solo-1", []domain.MemberSnapshot{
			{MemberID: "solo-1", MemberName: "Solo"},
		})

		// when
		got, err := BuildMemberPrompt(team, "solo-1")
		if err != nil {
			t.Fatal(err)
		}

		// then: no relation table, no teammate messaging, but has user guidance
		if strings.Contains(got, "| Role |") {
			t.Errorf("should not have relation table:\n%s", got)
		}
		if strings.Contains(got, "message send --to <id>") {
			t.Errorf("should not have teammate message section:\n%s", got)
		}
		if !strings.Contains(got, "message send --to "+domain.UserMemberID) {
			t.Errorf("should have user guidance:\n%s", got)
		}
	})

	t.Run("AllRelationTypes_ShowsFullTable", func(t *testing.T) {
		// given: Agent has leader, worker, and peer
		team := newTestTeam("leader-1", []domain.MemberSnapshot{
			{MemberID: "leader-1", MemberName: "Editor"},
			{MemberID: "agent-1", MemberName: "Agent", Relations: domain.MemberRelations{
				Leaders: []string{"leader-1"},
				Workers: []string{"worker-1"},
				Peers:   []string{"peer-1"},
			}},
			{MemberID: "worker-1", MemberName: "Writer"},
			{MemberID: "peer-1", MemberName: "Reviewer"},
		})

		// when
		got, err := BuildMemberPrompt(team, "agent-1")
		if err != nil {
			t.Fatal(err)
		}

		// then: prompt has all three relation rows
		//   | Role   | Name     | ID       |
		//   | Leader | Editor   | leader-1 |
		//   | Worker | Writer   | worker-1 |
		//   | Peer   | Reviewer | peer-1   |
		for _, want := range []string{
			"| Leader | Editor | leader-1 |",
			"| Worker | Writer | worker-1 |",
			"| Peer | Reviewer | peer-1 |",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}
	})

	t.Run("WithSystemPrompts_PrependedBeforeProtocol", func(t *testing.T) {
		// given: Agent has two system prompts
		team := newTestTeam("m-1", []domain.MemberSnapshot{
			{
				MemberID:   "m-1",
				MemberName: "Agent",
				SystemPrompts: []domain.PromptSnapshot{
					{Name: "style", Prompt: "Be concise."},
					{Name: "testing", Prompt: "Write tests."},
				},
			},
		})

		// when
		got, err := BuildMemberPrompt(team, "m-1")
		if err != nil {
			t.Fatal(err)
		}

		// then: prompt is
		//   Be concise.
		//
		//   Write tests.
		//
		//   ## Team Protocol
		//
		//   You are "Agent", part of team "MyTeam".
		for _, want := range []string{
			"Be concise.",
			"Write tests.",
			"## Team Protocol",
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

	t.Run("UnknownMemberID_ReturnsError", func(t *testing.T) {
		// given: team has only m-1
		team := newTestTeam("m-1", []domain.MemberSnapshot{
			{MemberID: "m-1", MemberName: "Agent"},
		})

		// when: build prompt for nonexistent member
		_, err := BuildMemberPrompt(team, "nonexistent")

		// then
		if err == nil {
			t.Error("should return error for unknown member ID")
		}
	})
}
