package workspace

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// ProtocolMember carries the runtime identity agents must use when communicating.
type ProtocolMember struct {
	ID   int64
	Name string
}

// BuildProtocol generates the team protocol CLAUDE.md content.
// Written to {memberDir}/CLAUDE.md (parent of project/).
func BuildProtocol(teamName, memberName string, relations domain.MemberRelations, membersByID map[int64]ProtocolMember) string {
	var b strings.Builder

	// Header
	b.WriteString("# Team Protocol\n\n")
	fmt.Fprintf(&b, "You are **%s**, a member of team **%s**.\n\n", memberName, teamName)

	// Team Structure
	b.WriteString("## Team Structure\n\n")
	writeRelNames(&b, "Leaders", relations.Leaders, membersByID)
	writeRelNames(&b, "Workers", relations.Workers, membersByID)
	if len(relations.Leaders) == 0 && len(relations.Workers) == 0 {
		b.WriteString("- (none)\n")
	}

	// Communication
	b.WriteString("\n## Communication\n\n")
	b.WriteString("**IMPORTANT:** Only use the `clier run tell` bash command below.\n")
	b.WriteString("Do NOT use SendMessage, Agent, or any other built-in tool for communication.\n\n")
	b.WriteString("The `--to` value must be the numeric team member ID shown below.\n\n")
	b.WriteString("Use heredoc to avoid shell escaping issues with backticks, flags, etc.:\n\n")
	writeTellCommands(&b, relations, membersByID)
	b.WriteString("- Replies arrive directly in your terminal input. Do not poll.\n")
	b.WriteString("- Keep each tell substantive. Avoid short fragments like \"ok\" or \"hi\".\n")

	// Progress Notes
	b.WriteString("\n## Progress Notes\n\n")
	b.WriteString("Post a note when you start, complete, or encounter issues:\n\n")
	b.WriteString("```bash\nclier run note <<'EOF'\n<content>\nEOF\n```\n")

	// Rules
	b.WriteString("\n## Rules\n\n")
	if len(relations.Workers) > 0 {
		b.WriteString("- Delegate to your workers. Do not do their work yourself.\n")
		b.WriteString("- Wait for ALL worker responses before wrapping up.\n")
	}
	if len(relations.Leaders) > 0 {
		b.WriteString("- Report your results to your leader when done.\n")
	}
	return b.String()
}

// writeTellCommands writes ready-to-use tell commands for each related member.
func writeTellCommands(b *strings.Builder, rel domain.MemberRelations, membersByID map[int64]ProtocolMember) {
	all := make([]int64, 0, len(rel.Leaders)+len(rel.Workers))
	all = append(all, rel.Leaders...)
	all = append(all, rel.Workers...)
	for _, id := range all {
		member := membersByID[id]
		fmt.Fprintf(b, "Tell %s (team member %d):\n```bash\nclier run tell --to %d <<'EOF'\n<message>\nEOF\n```\n", member.Name, member.ID, member.ID)
	}
}

// writeRelNames formats a relation line like "- Leaders: alice, bob".
func writeRelNames(b *strings.Builder, label string, ids []int64, membersByID map[int64]ProtocolMember) {
	if len(ids) == 0 {
		return
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		member := membersByID[id]
		names = append(names, fmt.Sprintf("%s (%d)", member.Name, member.ID))
	}
	fmt.Fprintf(b, "- %s: %s\n", label, strings.Join(names, ", "))
}
