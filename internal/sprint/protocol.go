package sprint

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// BuildProtocol generates the team protocol prompt for a member.
func BuildProtocol(memberName, teamName string, binary domain.CliBinary, isRoot bool, relations domain.MemberRelations, memberNames map[string]string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## Team Protocol\n\nYou are %q, part of team %q.\n", memberName, teamName)

	// Relation table
	rows := buildRelationRows(relations, memberNames)
	if len(rows) > 0 {
		b.WriteString("\n| Role | Name | ID |\n|------|------|----|")
		for _, row := range rows {
			fmt.Fprintf(&b, "\n| %s | %s | %s |", row.role, row.name, row.id)
		}
		b.WriteString("\n")
	}

	// Role guidance
	b.WriteString("\n")
	if isRoot {
		b.WriteString("Your leader is the Operator. When you complete a task, send the results to them. If you get stuck or need context, ask them.\n")
	} else if len(relations.Leaders) > 0 {
		leaderName := memberNames[relations.Leaders[0]]
		fmt.Fprintf(&b, "Your leader is %s. When you complete a task, send the results back to them. If you get stuck or need more context, ask them.\n", leaderName)
	}

	if len(relations.Workers) > 0 {
		b.WriteString("\nYou have workers who handle tasks better than doing them yourself. Delegate sub-tasks to them and wait for all responses before wrapping up.\n")
	}

	if len(relations.Peers) > 0 {
		b.WriteString("\nCoordinate with your peers when tasks overlap.\n")
	}

	// Communication section
	hasRelations := len(rows) > 0
	if hasRelations || isRoot {
		sendTarget := "a teammate"
		if isRoot {
			sendTarget = "a teammate or the Operator"
		}

		fmt.Fprintf(&b, "\nMessages from teammates appear directly in your conversation.\n\nTo message %s:\n\n```bash\n%s message send <id> \"<message>\"\n```\n", sendTarget, binary)

		if isRoot && !hasRelations {
			fmt.Fprintf(&b, "\nTo send results to the Operator:\n\n```bash\n%s message send operator \"<message>\"\n```\n", binary)
		}
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
func ComposePrompt(systemPrompts []domain.SnapshotPrompt, protocol string) string {
	var parts []string
	for _, sp := range systemPrompts {
		parts = append(parts, sp.Prompt)
	}
	parts = append(parts, protocol)
	return strings.Join(parts, "\n\n")
}
