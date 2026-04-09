package clone

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildAgentFacingTeamProtocol_UsesTeamMemberIDsForTellCommands(t *testing.T) {
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

func TestComposeAndStripTeamClaudeMdPrelude(t *testing.T) {
	t.Parallel()

	content := "You are a reviewer.\n"
	composed := ComposeTeamClaudeMd("reviewer", content)
	if !strings.HasPrefix(composed, "@.clier/work-log-protocol.md\n@.clier/reviewer-team-protocol.md") {
		t.Fatalf("missing import prelude:\n%s", composed)
	}

	stripped := StripTeamClaudeMdPrelude("reviewer", composed)
	if stripped != content {
		t.Fatalf("stripped content = %q, want %q", stripped, content)
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

func TestComposeAndStripMemberClaudeMdPrelude(t *testing.T) {
	t.Parallel()

	content := "You are a tech lead.\n"
	composed := ComposeMemberClaudeMd(content)
	if !strings.HasPrefix(composed, "@.clier/work-log-protocol.md") {
		t.Fatalf("missing member import prelude:\n%s", composed)
	}

	stripped := StripMemberClaudeMdPrelude(composed)
	if stripped != content {
		t.Fatalf("stripped content = %q, want %q", stripped, content)
	}
}
