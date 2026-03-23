package sprint

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// BuildProtocol generates the team protocol prompt for a member.
func BuildProtocol(team domain.TeamSnapshot, member domain.MemberSnapshot) string {
	memberNames := make(map[string]string, len(team.Members))
	for _, m := range team.Members {
		memberNames[m.MemberID] = m.MemberName
	}

	var b strings.Builder

	fmt.Fprintf(&b, "## Team Protocol\n\nYou are %q, part of team %q.\n", member.MemberName, team.TeamName)

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
	isRoot := member.MemberID == team.RootMemberID
	b.WriteString("\n")
	if isRoot && len(member.Relations.Workers) > 0 {
		b.WriteString("You are the root member. Coordinate your workers and synthesize their results.\n")
	} else if len(member.Relations.Leaders) > 0 {
		leaderName := memberNames[member.Relations.Leaders[0]]
		fmt.Fprintf(&b, "Your leader is %s. When you complete a task, send the results back to them. If you get stuck or need more context, ask them.\n", leaderName)
	}

	if len(member.Relations.Workers) > 0 {
		b.WriteString("\nYou have workers who handle tasks better than doing them yourself. Delegate sub-tasks to them and wait for all responses before wrapping up.\n")
	}

	if len(member.Relations.Peers) > 0 {
		b.WriteString("\nCoordinate with your peers when tasks overlap.\n")
	}

	// Communication section
	if len(rows) > 0 {
		fmt.Fprintf(&b, "\nMessages from teammates appear directly in your conversation.\n\nTo message a teammate:\n\n```bash\nclier message send <id> \"<message>\"\n```\n")
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

// ComposePrompt combines system prompts and team protocol into a single prompt.
func ComposePrompt(prompts []domain.SnapshotPrompt, protocol string) string {
	var parts []string
	for _, sp := range prompts {
		parts = append(parts, sp.Prompt)
	}
	parts = append(parts, protocol)
	return strings.Join(parts, "\n\n")
}
