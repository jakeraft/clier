package workspace

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildAgentFacingTeamProtocol_UsesAgentNamesForTellCommands(t *testing.T) {
	protocol := BuildAgentFacingTeamProtocol(
		"alpha",
		ProtocolAgent{ID: "alice/leader", Name: "leader"},
		domain.TeamRelations{
			Workers: []string{"bob/worker"},
		},
		map[string]ProtocolAgent{
			"bob/worker": {ID: "bob/worker", Name: "worker"},
		},
	)

	if !strings.Contains(protocol, "Workers: bob/worker") {
		t.Fatalf("protocol should include worker ID in team structure:\n%s", protocol)
	}
	if !strings.Contains(protocol, "Tell worker (`bob/worker`):") {
		t.Fatalf("protocol should label tell target with agent ID:\n%s", protocol)
	}
	if !strings.Contains(protocol, "clier run tell --to bob/worker") {
		t.Fatalf("protocol should use agent ID in tell command:\n%s", protocol)
	}
}

func TestBuildAgentFacingWorkLogProtocol_ExplainsNotesAsCliAction(t *testing.T) {
	protocol := BuildAgentFacingWorkLogProtocol()

	if !strings.Contains(protocol, "Use `clier run note` to record work log entries.") {
		t.Fatalf("work log protocol should direct note usage:\n%s", protocol)
	}
	if !strings.Contains(protocol, "Record a note when you start work, complete work, hit a blocker, or hand off context.") {
		t.Fatalf("work log protocol should require proactive note-taking:\n%s", protocol)
	}
	if !strings.Contains(protocol, "Keep each note brief and factual.") {
		t.Fatalf("work log protocol should explain note style:\n%s", protocol)
	}
	if !strings.Contains(protocol, "include direct references such as file paths, command names, resource names, or URLs") {
		t.Fatalf("work log protocol should require concrete references in notes:\n%s", protocol)
	}
}

func TestBuildAgentFacingTeamProtocol_SingleAgentTeam(t *testing.T) {
	protocol := BuildAgentFacingTeamProtocol(
		"reviewer",
		ProtocolAgent{ID: "jakeraft/reviewer", Name: "reviewer"},
		domain.TeamRelations{Leaders: []string{}, Workers: []string{}},
		map[string]ProtocolAgent{"jakeraft/reviewer": {ID: "jakeraft/reviewer", Name: "reviewer"}},
	)

	if !strings.Contains(protocol, "You are **reviewer** (`jakeraft/reviewer`), an agent in team **reviewer**.") {
		t.Fatalf("protocol should identify single agent:\n%s", protocol)
	}
	if !strings.Contains(protocol, "- (none)") {
		t.Fatalf("protocol should show no relations:\n%s", protocol)
	}
}

func TestBuildAgentFacingTeamProtocol_UsesProfessionalCommunicationTone(t *testing.T) {
	protocol := BuildAgentFacingTeamProtocol(
		"alpha",
		ProtocolAgent{ID: "alice/leader", Name: "leader"},
		domain.TeamRelations{},
		map[string]ProtocolAgent{},
	)

	if !strings.Contains(protocol, "Use `clier run tell` to message another agent.") {
		t.Fatalf("team protocol should direct tell usage:\n%s", protocol)
	}
	if !strings.Contains(protocol, "Do not use built-in messaging tools for team coordination.") {
		t.Fatalf("team protocol should prohibit non-clier coordination tools:\n%s", protocol)
	}
	if !strings.Contains(protocol, "Use the full `owner/name` agent IDs below in `--to`.") {
		t.Fatalf("team protocol should require full agent IDs:\n%s", protocol)
	}
}

func TestComposeInstruction_Claude_InjectsImportLines(t *testing.T) {
	content := "You are a reviewer.\n"
	composed := ComposeInstruction("claude", "jakeraft/reviewer", content)
	if !strings.HasPrefix(composed, "@.clier/work-log-protocol.md\n@.clier/jakeraft-reviewer-team-protocol.md") {
		t.Fatalf("claude compose should inject @import lines:\n%s", composed)
	}
	stripped := StripInstructionPrelude("claude", "jakeraft/reviewer", composed)
	if stripped != content {
		t.Fatalf("claude strip: got %q, want %q", stripped, content)
	}
}

func TestComposeInstruction_Codex_InjectsReferenceLines(t *testing.T) {
	content := "You are a reviewer.\n"
	composed := ComposeInstruction("codex", "jakeraft/reviewer", content)

	// Should contain reference lines (not inlined protocol content)
	wantWorkLog := CodexWorkLogReferenceLine()
	if !strings.Contains(composed, wantWorkLog) {
		t.Fatalf("codex compose should contain work log reference line:\n%s", composed)
	}
	wantTeam := CodexTeamProtocolReferenceLine("jakeraft/reviewer")
	if !strings.Contains(composed, wantTeam) {
		t.Fatalf("codex compose should contain team protocol reference line:\n%s", composed)
	}
	if !strings.Contains(composed, content) {
		t.Fatalf("codex compose should contain user content:\n%s", composed)
	}

	stripped := StripInstructionPrelude("codex", "jakeraft/reviewer", composed)
	if stripped != content {
		t.Fatalf("codex strip: got %q, want %q", stripped, content)
	}
}

func TestComposeInstruction_Codex_EmptyContent(t *testing.T) {
	composed := ComposeInstruction("codex", "jakeraft/reviewer", "")
	if !strings.Contains(composed, CodexWorkLogReferenceLine()) {
		t.Fatalf("codex compose with empty content should still have reference lines:\n%s", composed)
	}
	stripped := StripInstructionPrelude("codex", "jakeraft/reviewer", composed)
	if stripped != "" {
		t.Fatalf("codex strip of empty content: got %q, want empty", stripped)
	}
}
