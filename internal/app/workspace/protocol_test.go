package workspace

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildAgentFacingTeamProtocol_UsesMemberIDsForTellCommands(t *testing.T) {
	protocol := BuildAgentFacingTeamProtocol(
		"alpha",
		"leader",
		domain.MemberRelations{
			Workers: []int64{12},
		},
		map[int64]ProtocolMember{
			12: {ID: 12, Name: "worker"},
		},
	)

	if !strings.Contains(protocol, "Workers: worker (12)") {
		t.Fatalf("protocol should include worker id in team structure:\n%s", protocol)
	}
	if !strings.Contains(protocol, "Tell worker (team member 12):") {
		t.Fatalf("protocol should label tell target with numeric team member id:\n%s", protocol)
	}
	if !strings.Contains(protocol, "clier run tell --to 12") {
		t.Fatalf("protocol should use numeric team member id in tell command:\n%s", protocol)
	}
	if strings.Contains(protocol, "clier run tell --to worker") {
		t.Fatalf("protocol should not use member names as tell targets:\n%s", protocol)
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


func TestBuildAgentFacingTeamProtocol_SingleMemberTeam(t *testing.T) {
	protocol := BuildAgentFacingTeamProtocol(
		"reviewer",
		"reviewer",
		domain.MemberRelations{Leaders: []int64{}, Workers: []int64{}},
		map[int64]ProtocolMember{42: {ID: 42, Name: "reviewer"}},
	)

	if !strings.Contains(protocol, "You are **reviewer**, operating as a member of team **reviewer**.") {
		t.Fatalf("protocol should identify single member:\n%s", protocol)
	}
	if !strings.Contains(protocol, "- (none)") {
		t.Fatalf("protocol should show no relations:\n%s", protocol)
	}
}

func TestBuildAgentFacingTeamProtocol_UsesProfessionalCommunicationTone(t *testing.T) {
	protocol := BuildAgentFacingTeamProtocol(
		"alpha",
		"leader",
		domain.MemberRelations{},
		map[int64]ProtocolMember{},
	)

	if !strings.Contains(protocol, "Use `clier run tell` to message another team member.") {
		t.Fatalf("team protocol should direct tell usage:\n%s", protocol)
	}
	if !strings.Contains(protocol, "Do not use built-in messaging tools for team coordination.") {
		t.Fatalf("team protocol should prohibit non-clier coordination tools:\n%s", protocol)
	}
}

func TestComposeInstruction_Claude_InjectsImportLines(t *testing.T) {
	content := "You are a reviewer.\n"
	composed := ComposeInstruction("claude", "reviewer", content, "ignored team protocol")
	if !strings.HasPrefix(composed, "@.clier/work-log-protocol.md\n@.clier/reviewer-team-protocol.md") {
		t.Fatalf("claude compose should inject @import lines:\n%s", composed)
	}
	stripped := StripInstructionPrelude("claude", "reviewer", composed)
	if stripped != content {
		t.Fatalf("claude strip: got %q, want %q", stripped, content)
	}
}

func TestComposeInstruction_Codex_InlinesProtocol(t *testing.T) {
	content := "You are a reviewer.\n"
	teamProtocol := "# Team Protocol\n\nYou are **reviewer**.\n"
	composed := ComposeInstruction("codex", "reviewer", content, teamProtocol)

	if !strings.Contains(composed, "# Work Log Protocol") {
		t.Fatalf("codex compose should inline work log protocol:\n%s", composed)
	}
	if !strings.Contains(composed, "# Team Protocol") {
		t.Fatalf("codex compose should inline team protocol:\n%s", composed)
	}
	if !strings.Contains(composed, instructionPreludeEnd) {
		t.Fatalf("codex compose should contain prelude end marker:\n%s", composed)
	}
	if !strings.Contains(composed, content) {
		t.Fatalf("codex compose should contain user content:\n%s", composed)
	}

	stripped := StripInstructionPrelude("codex", "reviewer", composed)
	if stripped != content {
		t.Fatalf("codex strip: got %q, want %q", stripped, content)
	}
}

func TestComposeInstruction_Codex_EmptyContent(t *testing.T) {
	teamProtocol := "# Team Protocol\n"
	composed := ComposeInstruction("codex", "reviewer", "", teamProtocol)
	if !strings.Contains(composed, "# Work Log Protocol") {
		t.Fatalf("codex compose with empty content should still have work log:\n%s", composed)
	}
	stripped := StripInstructionPrelude("codex", "reviewer", composed)
	if stripped != "" {
		t.Fatalf("codex strip of empty content: got %q, want empty", stripped)
	}
}

func TestComposeInstruction_Codex_EmptyTeamProtocol(t *testing.T) {
	content := "You are solo.\n"
	composed := ComposeInstruction("codex", "solo", content, "")
	if !strings.Contains(composed, "# Work Log Protocol") {
		t.Fatalf("codex compose should have work log even without team protocol:\n%s", composed)
	}
	stripped := StripInstructionPrelude("codex", "solo", composed)
	if stripped != content {
		t.Fatalf("codex strip: got %q, want %q", stripped, content)
	}
}
