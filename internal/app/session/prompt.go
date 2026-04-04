package session

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// joinPrompts combines multiple system prompts into a single string,
// separated by double newlines.
func joinPrompts(prompts []domain.PromptSnapshot) string {
	parts := make([]string, 0, len(prompts))
	for _, sp := range prompts {
		parts = append(parts, sp.Prompt)
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// buildClierPrompt generates the team protocol that clier injects at plan build time.
// It gives the agent full context about its role, team structure, and communication
// protocol — no runtime discovery needed.
// memberName identifies this agent; relations are resolved from the team;
// nameByID maps TeamMember IDs to display names.
func buildClierPrompt(teamName, memberName string, relations domain.MemberRelations, nameByID map[string]string) string {
	var b strings.Builder

	// Header + intro
	b.WriteString("# Team Protocol\n\n")
	fmt.Fprintf(&b, "You are **%s**, a member of team **%s**.\n", memberName, teamName)
	b.WriteString("This protocol defines your role, team structure, and how to coordinate with teammates.\n")

	// Team Structure
	b.WriteString("\n## Team Structure\n")
	writeRelNames(&b, "Leaders", relations.Leaders, nameByID)
	writeRelNames(&b, "Workers", relations.Workers, nameByID)
	writeRelNames(&b, "Peers", relations.Peers, nameByID)
	if len(relations.Leaders) == 0 && len(relations.Workers) == 0 && len(relations.Peers) == 0 {
		b.WriteString("(none)\n")
	}

	// Communication
	b.WriteString("\n## Communication\n")
	b.WriteString("Use the commands below to send messages to your teammates.\n")
	b.WriteString("Replies arrive directly in your terminal input — do not poll or call any receive command.\n")
	b.WriteString("Keep each message substantive. Avoid short fragments like \"ok\" or \"hi\".\n\n")
	writeSendCommands(&b, relations, nameByID)

	// Logging
	b.WriteString("\n## Logging\n")
	b.WriteString("Record your progress and results:\n")
	b.WriteString("```bash\nclier session log \"<content>\"\n```\n")
	b.WriteString("Log when you: start a task, complete a task, encounter issues, produce final results.\n")

	// Operating Rules
	b.WriteString("\n## Operating Rules\n")
	if len(relations.Workers) > 0 {
		b.WriteString("- Your workers have specialized roles — always delegate through them instead of doing their work yourself.\n")
		b.WriteString("- You MUST wait for all worker responses before wrapping up.\n")
	}
	if len(relations.Leaders) > 0 {
		b.WriteString("- You MUST report your results to your leader when done.\n")
	}
	if len(relations.Peers) > 0 {
		b.WriteString("- Coordinate with your peers when tasks overlap.\n")
	}
	return b.String()
}

// writeSendCommands writes ready-to-use send commands for each related member.
func writeSendCommands(b *strings.Builder, rel domain.MemberRelations, nameByID map[string]string) {
	all := make([]string, 0, len(rel.Leaders)+len(rel.Workers)+len(rel.Peers))
	all = append(all, rel.Leaders...)
	all = append(all, rel.Workers...)
	all = append(all, rel.Peers...)
	for _, id := range all {
		fmt.Fprintf(b, "Send to %s:\n```bash\nclier session send --to %s \"<message>\"\n```\n", nameByID[id], id)
	}
}

// writeRelNames formats a relation line like "- Leaders: alice, bob".
func writeRelNames(b *strings.Builder, label string, ids []string, nameByID map[string]string) {
	if len(ids) == 0 {
		return
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, nameByID[id])
	}
	fmt.Fprintf(b, "- %s: %s\n", label, strings.Join(names, ", "))
}
