package sprint

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

const (
	promptHeader = "## Team Protocol\n\nYou are %q, part of team %q.\n"

	promptLeaderGuidance = "Your leader is %q. Report results to them.\n"
	promptWorkerGuidance = "Delegate sub-tasks to workers. Wait for all responses before wrapping up.\n"
	promptPeerGuidance   = "Coordinate with peers when tasks overlap.\n"

	promptMessaging = "To message a teammate:\n\n```bash\nclier message send <id> \"<message>\"\n```\n"
)

// BuildMemberPrompt generates the full prompt for a member by combining
// system prompts and team protocol into a single string.
func BuildMemberPrompt(team domain.TeamSnapshot, memberID string) (string, error) {
	member, ok := findMember(team.Members, memberID)
	if !ok {
		return "", fmt.Errorf("member %q not found in team %q", memberID, team.TeamName)
	}

	memberNames := make(map[string]string, len(team.Members))
	for _, m := range team.Members {
		memberNames[m.MemberID] = m.MemberName
	}

	var parts []string
	for _, sp := range member.SystemPrompts {
		parts = append(parts, sp.Prompt)
	}
	parts = append(parts, buildProtocol(team.TeamName, member, memberNames))

	return strings.Join(parts, "\n\n"), nil
}

func buildProtocol(teamName string, member domain.MemberSnapshot, memberNames map[string]string) string {
	var b strings.Builder

	fmt.Fprintf(&b, promptHeader, member.MemberName, teamName)

	// Relation table
	rows := buildRelationRows(member.Relations, memberNames)
	if len(rows) > 0 {
		b.WriteString("\n| Role | Name | ID |\n|------|------|----|")
		for _, row := range rows {
			fmt.Fprintf(&b, "\n| %s | %s | %s |", row.role, row.name, row.id)
		}
		b.WriteString("\n")
	}

	// Role guidance
	if len(member.Relations.Leaders) > 0 {
		b.WriteString("\n")
		fmt.Fprintf(&b, promptLeaderGuidance, memberNames[member.Relations.Leaders[0]])
	}

	if len(member.Relations.Workers) > 0 {
		b.WriteString("\n")
		b.WriteString(promptWorkerGuidance)
	}

	if len(member.Relations.Peers) > 0 {
		b.WriteString("\n")
		b.WriteString(promptPeerGuidance)
	}

	// Communication section
	if len(rows) > 0 {
		b.WriteString("\n")
		b.WriteString(promptMessaging)
	}

	return b.String()
}

type relationRow struct {
	role string
	name string
	id   string
}

func buildRelationRows(relations domain.MemberRelations, memberNames map[string]string) []relationRow {
	var rows []relationRow
	for _, id := range relations.Leaders {
		rows = append(rows, relationRow{"Leader", memberNames[id], id})
	}
	for _, id := range relations.Workers {
		rows = append(rows, relationRow{"Worker", memberNames[id], id})
	}
	for _, id := range relations.Peers {
		rows = append(rows, relationRow{"Peer", memberNames[id], id})
	}
	return rows
}

func findMember(members []domain.MemberSnapshot, memberID string) (domain.MemberSnapshot, bool) {
	for _, m := range members {
		if m.MemberID == memberID {
			return m, true
		}
	}
	return domain.MemberSnapshot{}, false
}
