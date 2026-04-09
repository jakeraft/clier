package workspace

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildProtocol_UsesTeamMemberIDsForTellCommands(t *testing.T) {
	protocol := BuildProtocol(
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
