package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// ProtocolMember carries the runtime identity agents must use when communicating.
type ProtocolMember struct {
	Owner string
	Name  string
}

// memberKey returns the unique identifier for a member (owner/name).
func memberKey(owner, name string) string {
	return owner + "/" + name
}

const workLogProtocolFileName = "work-log-protocol.md"

func WorkLogProtocolImportPath() string {
	return filepath.ToSlash(filepath.Join(".clier", workLogProtocolFileName))
}

func TeamProtocolFileName(memberName string) string {
	return sanitizeRepoDirName(memberName) + "-team-protocol.md"
}

func TeamProtocolImportPath(memberName string) string {
	return filepath.ToSlash(filepath.Join(".clier", TeamProtocolFileName(memberName)))
}

func TeamProtocolImportLine(memberName string) string {
	return "@" + TeamProtocolImportPath(memberName)
}

func WorkLogProtocolImportLine() string {
	return "@" + WorkLogProtocolImportPath()
}

func TeamWorkLogProtocolImportLine() string {
	return "@" + WorkLogProtocolImportPath()
}

// ComposeInstruction wraps instruction content with agent-specific protocol references.
// Claude: injects @import lines (native import syntax).
// Codex: injects plain-text reference lines (agent reads files when needed).
// Both agents use the same file structure; only the reference format differs.
func ComposeInstruction(agentType, memberName, content string) string {
	switch agentType {
	case "codex":
		return composeCodexInstruction(memberName, content)
	default:
		return composeClaudeInstruction(memberName, content)
	}
}

// StripInstructionPrelude removes the agent-specific protocol prelude for push.
func StripInstructionPrelude(agentType, memberName, content string) string {
	switch agentType {
	case "codex":
		return stripCodexInstructionPrelude(memberName, content)
	default:
		return stripClaudeInstructionPrelude(memberName, content)
	}
}

func composeClaudeInstruction(memberName, content string) string {
	content = strings.TrimLeft(content, "\n")
	workLogLine := TeamWorkLogProtocolImportLine()
	teamLine := TeamProtocolImportLine(memberName)
	if content == "" {
		return workLogLine + "\n" + teamLine + "\n"
	}
	return workLogLine + "\n" + teamLine + "\n\n" + content
}

func stripClaudeInstructionPrelude(memberName, content string) string {
	prefixes := []string{
		TeamWorkLogProtocolImportLine() + "\n" + TeamProtocolImportLine(memberName) + "\n\n",
		TeamWorkLogProtocolImportLine() + "\n" + TeamProtocolImportLine(memberName) + "\n",
		TeamProtocolImportLine(memberName) + "\n\n",
		TeamProtocolImportLine(memberName) + "\n",
	}
	for _, prefix := range prefixes {
		if stripped, ok := strings.CutPrefix(content, prefix); ok {
			return strings.TrimLeft(stripped, "\n")
		}
	}
	return content
}

// CodexWorkLogReferenceLine returns the reference line for work log protocol in Codex instruction files.
func CodexWorkLogReferenceLine() string {
	return "Read " + WorkLogProtocolImportPath() + " for work logging conventions."
}

// CodexTeamProtocolReferenceLine returns the reference line for team protocol in Codex instruction files.
func CodexTeamProtocolReferenceLine(memberName string) string {
	return "Read " + TeamProtocolImportPath(memberName) + " for team coordination."
}

func composeCodexInstruction(memberName, content string) string {
	content = strings.TrimLeft(content, "\n")
	workLogLine := CodexWorkLogReferenceLine()
	teamLine := CodexTeamProtocolReferenceLine(memberName)
	if content == "" {
		return workLogLine + "\n" + teamLine + "\n"
	}
	return workLogLine + "\n" + teamLine + "\n\n" + content
}

func stripCodexInstructionPrelude(memberName, content string) string {
	prefixes := []string{
		CodexWorkLogReferenceLine() + "\n" + CodexTeamProtocolReferenceLine(memberName) + "\n\n",
		CodexWorkLogReferenceLine() + "\n" + CodexTeamProtocolReferenceLine(memberName) + "\n",
		CodexTeamProtocolReferenceLine(memberName) + "\n\n",
		CodexTeamProtocolReferenceLine(memberName) + "\n",
	}
	for _, prefix := range prefixes {
		if stripped, ok := strings.CutPrefix(content, prefix); ok {
			return strings.TrimLeft(stripped, "\n")
		}
	}
	return content
}

func BuildAgentFacingWorkLogProtocol() string {
	var b strings.Builder

	b.WriteString("# Work Log Protocol\n\n")
	b.WriteString("Use `clier run note` to record work log entries.\n")
	b.WriteString("Record a note when you start work, complete work, hit a blocker, or hand off context.\n")
	b.WriteString("If someone asks you to write, record, post, or leave a note, use `clier run note`.\n")
	b.WriteString("Keep each note brief and factual.\n")
	b.WriteString("When a note mentions concrete work, include direct references such as file paths, command names, resource names, or URLs whenever they would help someone continue from the note later.\n\n")
	b.WriteString("```bash\nclier run note <<'EOF'\n<content>\nEOF\n```\n")

	return b.String()
}

// BuildAgentFacingTeamProtocol generates the team-specific agent-facing
// protocol content for a member. Written to {teamRoot}/{member}/.clier/{member}-team-protocol.md.
// Claude imports it via @-reference; Codex inlines it into AGENTS.md.
func BuildAgentFacingTeamProtocol(teamName, memberName string, relations domain.MemberRelations, membersByName map[string]ProtocolMember) string {
	var b strings.Builder

	// Header
	b.WriteString("# Team Protocol\n\n")
	fmt.Fprintf(&b, "You are **%s**, operating as a member of team **%s**.\n\n", memberName, teamName)

	// Team Structure
	b.WriteString("## Team Structure\n\n")
	writeRelNames(&b, "Leaders", relations.Leaders, membersByName)
	writeRelNames(&b, "Workers", relations.Workers, membersByName)
	if len(relations.Leaders) == 0 && len(relations.Workers) == 0 {
		b.WriteString("- (none)\n")
	}

	// Communication
	b.WriteString("\n## Communication\n\n")
	b.WriteString("Use `clier run tell` to message another team member.\n")
	b.WriteString("Use the team member names below in `--to`.\n")
	b.WriteString("Do not use built-in messaging tools for team coordination.\n\n")
	writeTellCommands(&b, relations, membersByName)
	b.WriteString("- Replies arrive directly in your terminal input.\n")
	b.WriteString("- Keep each message substantive and action-oriented.\n")

	// Rules
	b.WriteString("\n## Operating Rules\n\n")
	if len(relations.Workers) > 0 {
		b.WriteString("- Delegate work to your workers. Do not do their assigned work yourself.\n")
		b.WriteString("- Wait for all worker responses before wrapping up your own task.\n")
	}
	if len(relations.Leaders) > 0 {
		b.WriteString("- Report your results to your leader when your work is complete.\n")
	}
	return b.String()
}

// writeTellCommands writes ready-to-use tell commands for each related member.
func writeTellCommands(b *strings.Builder, rel domain.MemberRelations, membersByName map[string]ProtocolMember) {
	all := make([]string, 0, len(rel.Leaders)+len(rel.Workers))
	all = append(all, rel.Leaders...)
	all = append(all, rel.Workers...)
	for _, name := range all {
		member := membersByName[name]
		fmt.Fprintf(b, "Tell %s:\n```bash\nclier run tell --to %s <<'EOF'\n<message>\nEOF\n```\n", member.Name, member.Name)
	}
}

// writeRelNames formats a relation line like "- Leaders: alice, bob".
func writeRelNames(b *strings.Builder, label string, names []string, membersByName map[string]ProtocolMember) {
	if len(names) == 0 {
		return
	}
	displayNames := make([]string, 0, len(names))
	for _, name := range names {
		member := membersByName[name]
		displayNames = append(displayNames, member.Name)
	}
	fmt.Fprintf(b, "- %s: %s\n", label, strings.Join(displayNames, ", "))
}
