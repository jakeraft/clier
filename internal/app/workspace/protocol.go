package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// ProtocolAgent carries the runtime identity agents must use when communicating.
type ProtocolAgent struct {
	ID    string
	Owner string
	Name  string
}

const workLogProtocolFileName = "work-log-protocol.md"

func WorkLogProtocolImportPath() string {
	return filepath.ToSlash(filepath.Join(".clier", workLogProtocolFileName))
}

func TeamProtocolFileName(agentID string) string {
	return sanitizeRepoDirName(agentID) + "-team-protocol.md"
}

func TeamProtocolImportPath(agentID string) string {
	return filepath.ToSlash(filepath.Join(".clier", TeamProtocolFileName(agentID)))
}

func TeamProtocolImportLine(agentID string) string {
	return "@" + TeamProtocolImportPath(agentID)
}

func TeamWorkLogProtocolImportLine() string {
	return "@" + WorkLogProtocolImportPath()
}

// ComposeInstruction wraps instruction content with agent-specific protocol references.
// Claude: injects @import lines (native import syntax).
// Codex: injects plain-text reference lines (agent reads files when needed).
// Both agents use the same file structure; only the reference format differs.
func ComposeInstruction(agentType, agentID, content string) string {
	switch agentType {
	case "codex":
		return composeCodexInstruction(agentID, content)
	default:
		return composeClaudeInstruction(agentID, content)
	}
}

// StripInstructionPrelude removes the agent-specific protocol prelude for push.
func StripInstructionPrelude(agentType, agentID, content string) string {
	switch agentType {
	case "codex":
		return stripCodexInstructionPrelude(agentID, content)
	default:
		return stripClaudeInstructionPrelude(agentID, content)
	}
}

func composeClaudeInstruction(agentID, content string) string {
	content = strings.TrimLeft(content, "\n")
	workLogLine := TeamWorkLogProtocolImportLine()
	teamLine := TeamProtocolImportLine(agentID)
	if content == "" {
		return workLogLine + "\n" + teamLine + "\n"
	}
	return workLogLine + "\n" + teamLine + "\n\n" + content
}

func stripClaudeInstructionPrelude(agentID, content string) string {
	prefixes := []string{
		TeamWorkLogProtocolImportLine() + "\n" + TeamProtocolImportLine(agentID) + "\n\n",
		TeamWorkLogProtocolImportLine() + "\n" + TeamProtocolImportLine(agentID) + "\n",
		TeamProtocolImportLine(agentID) + "\n\n",
		TeamProtocolImportLine(agentID) + "\n",
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
func CodexTeamProtocolReferenceLine(agentID string) string {
	return "Read " + TeamProtocolImportPath(agentID) + " for team coordination."
}

func composeCodexInstruction(agentID, content string) string {
	content = strings.TrimLeft(content, "\n")
	workLogLine := CodexWorkLogReferenceLine()
	teamLine := CodexTeamProtocolReferenceLine(agentID)
	if content == "" {
		return workLogLine + "\n" + teamLine + "\n"
	}
	return workLogLine + "\n" + teamLine + "\n\n" + content
}

func stripCodexInstructionPrelude(agentID, content string) string {
	prefixes := []string{
		CodexWorkLogReferenceLine() + "\n" + CodexTeamProtocolReferenceLine(agentID) + "\n\n",
		CodexWorkLogReferenceLine() + "\n" + CodexTeamProtocolReferenceLine(agentID) + "\n",
		CodexTeamProtocolReferenceLine(agentID) + "\n\n",
		CodexTeamProtocolReferenceLine(agentID) + "\n",
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
// protocol content for an agent. Written to each agent's local workspace.
// Claude imports it via @-reference; Codex inlines it into AGENTS.md.
func BuildAgentFacingTeamProtocol(teamName string, self ProtocolAgent, relations domain.TeamRelations, agentsByKey map[string]ProtocolAgent) string {
	var b strings.Builder

	// Header
	b.WriteString("# Team Protocol\n\n")
	fmt.Fprintf(&b, "You are **%s** (`%s`), an agent in team **%s**.\n\n", self.Name, self.ID, teamName)

	// Team Structure
	b.WriteString("## Team Structure\n\n")
	writeRelNames(&b, "Leaders", relations.Leaders, agentsByKey)
	writeRelNames(&b, "Workers", relations.Workers, agentsByKey)
	if len(relations.Leaders) == 0 && len(relations.Workers) == 0 {
		b.WriteString("- (none)\n")
	}

	// Communication
	b.WriteString("\n## Communication\n\n")
	b.WriteString("Use `clier run tell` to message another agent.\n")
	b.WriteString("Use the full `owner/name` agent IDs below in `--to`.\n")
	b.WriteString("Do not use built-in messaging tools for team coordination.\n\n")
	writeTellCommands(&b, relations, agentsByKey)
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

// writeTellCommands writes ready-to-use tell commands for each related agent.
func writeTellCommands(b *strings.Builder, rel domain.TeamRelations, agentsByKey map[string]ProtocolAgent) {
	all := make([]string, 0, len(rel.Leaders)+len(rel.Workers))
	all = append(all, rel.Leaders...)
	all = append(all, rel.Workers...)
	for _, name := range all {
		agent := agentsByKey[name]
		fmt.Fprintf(b, "Tell %s (`%s`):\n```bash\nclier run tell --to %s <<'EOF'\n<message>\nEOF\n```\n", agent.Name, agent.ID, agent.ID)
	}
}

// writeRelNames formats a relation line like "- Leaders: alice, bob".
func writeRelNames(b *strings.Builder, label string, names []string, agentsByKey map[string]ProtocolAgent) {
	if len(names) == 0 {
		return
	}
	displayNames := make([]string, 0, len(names))
	for _, name := range names {
		agent := agentsByKey[name]
		displayNames = append(displayNames, agent.ID)
	}
	fmt.Fprintf(b, "- %s: %s\n", label, strings.Join(displayNames, ", "))
}
