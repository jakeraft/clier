package sprint

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildProtocol(t *testing.T) {
	names := map[string]string{
		"leader-1": "Editor",
		"worker-1": "Writer",
		"peer-1":   "Reviewer",
	}

	t.Run("RootMember/MentionsOperator", func(t *testing.T) {
		relations := domain.MemberRelations{
			Workers: []string{"worker-1"},
		}
		got := BuildProtocol("Boss", "MyTeam", domain.BinaryClaude, true, relations, names)

		if !strings.Contains(got, `"Boss"`) {
			t.Errorf("should contain member name: %s", got)
		}
		if !strings.Contains(got, `"MyTeam"`) {
			t.Errorf("should contain team name: %s", got)
		}
		if !strings.Contains(got, "Operator") {
			t.Errorf("root should mention Operator: %s", got)
		}
		if !strings.Contains(got, "claude message send") {
			t.Errorf("should contain send command: %s", got)
		}
	})

	t.Run("NonRoot/MentionsLeader", func(t *testing.T) {
		relations := domain.MemberRelations{
			Leaders: []string{"leader-1"},
			Peers:   []string{"peer-1"},
		}
		got := BuildProtocol("Writer", "MyTeam", domain.BinaryCodex, false, relations, names)

		if !strings.Contains(got, "Editor") {
			t.Errorf("should mention leader name: %s", got)
		}
		if !strings.Contains(got, "codex message send") {
			t.Errorf("should use codex binary in command: %s", got)
		}
		if strings.Contains(got, "Operator") {
			t.Errorf("non-root should not mention Operator: %s", got)
		}
	})

	t.Run("NoRelations/RootGetsOperatorHint", func(t *testing.T) {
		relations := domain.MemberRelations{}
		got := BuildProtocol("Solo", "MyTeam", domain.BinaryClaude, true, relations, names)

		if !strings.Contains(got, "message send operator") {
			t.Errorf("solo root should get operator send hint: %s", got)
		}
	})

	t.Run("RelationTable/ShowsAllRoles", func(t *testing.T) {
		relations := domain.MemberRelations{
			Leaders: []string{"leader-1"},
			Workers: []string{"worker-1"},
			Peers:   []string{"peer-1"},
		}
		got := BuildProtocol("Agent", "MyTeam", domain.BinaryClaude, false, relations, names)

		if !strings.Contains(got, "| Leader | Editor | leader-1 |") {
			t.Errorf("should show leader row: %s", got)
		}
		if !strings.Contains(got, "| Worker | Writer | worker-1 |") {
			t.Errorf("should show worker row: %s", got)
		}
		if !strings.Contains(got, "| Peer | Reviewer | peer-1 |") {
			t.Errorf("should show peer row: %s", got)
		}
	})
}

func TestComposePrompt(t *testing.T) {
	t.Run("CombinesPromptsAndProtocol", func(t *testing.T) {
		prompts := []domain.SnapshotPrompt{
			{Name: "p1", Prompt: "Be concise."},
			{Name: "p2", Prompt: "Write tests."},
		}
		got := ComposePrompt(prompts, "## Team Protocol\n...")

		if !strings.Contains(got, "Be concise.") {
			t.Errorf("should contain first prompt: %s", got)
		}
		if !strings.Contains(got, "Write tests.") {
			t.Errorf("should contain second prompt: %s", got)
		}
		if !strings.Contains(got, "## Team Protocol") {
			t.Errorf("should contain protocol: %s", got)
		}
	})
}
