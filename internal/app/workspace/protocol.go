package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// ProtocolMember carries the runtime identity agents must use when communicating.
type ProtocolMember struct {
	ID   int64
	Name string
}

const workLogProtocolFileName = "work-log-protocol.md"

func WorkLogProtocolImportPath() string {
	return filepath.ToSlash(filepath.Join(".clier", workLogProtocolFileName))
}

func TeamWorkLogProtocolImportPath() string {
	return filepath.ToSlash(filepath.Join("..", ".clier", workLogProtocolFileName))
}

func TeamProtocolFileName(memberName string) string {
	return sanitizeRepoDirName(memberName) + "-team-protocol.md"
}

func TeamProtocolImportPath(memberName string) string {
	return filepath.ToSlash(filepath.Join("..", ".clier", TeamProtocolFileName(memberName)))
}

func TeamProtocolImportLine(memberName string) string {
	return "@" + TeamProtocolImportPath(memberName)
}

func WorkLogProtocolImportLine() string {
	return "@" + WorkLogProtocolImportPath()
}

func TeamWorkLogProtocolImportLine() string {
	return "@" + TeamWorkLogProtocolImportPath()
}

func ComposeMemberClaudeMd(content string) string {
	content = strings.TrimLeft(content, "\n")
	importLine := WorkLogProtocolImportLine()
	if content == "" {
		return importLine + "\n"
	}
	return importLine + "\n\n" + content
}

func StripMemberClaudeMdPrelude(content string) string {
	importLine := WorkLogProtocolImportLine()
	switch {
	case strings.HasPrefix(content, importLine+"\n\n"):
		return strings.TrimLeft(strings.TrimPrefix(content, importLine+"\n\n"), "\n")
	case strings.HasPrefix(content, importLine+"\n"):
		return strings.TrimLeft(strings.TrimPrefix(content, importLine+"\n"), "\n")
	default:
		return content
	}
}

func ComposeTeamClaudeMd(memberName, content string) string {
	content = strings.TrimLeft(content, "\n")
	workLogLine := TeamWorkLogProtocolImportLine()
	teamLine := TeamProtocolImportLine(memberName)
	if content == "" {
		return workLogLine + "\n" + teamLine + "\n"
	}
	return workLogLine + "\n" + teamLine + "\n\n" + content
}

func StripTeamClaudeMdPrelude(memberName, content string) string {
	prefixes := []string{
		TeamWorkLogProtocolImportLine() + "\n" + TeamProtocolImportLine(memberName) + "\n\n",
		TeamWorkLogProtocolImportLine() + "\n" + TeamProtocolImportLine(memberName) + "\n",
		TeamProtocolImportLine(memberName) + "\n\n",
		TeamProtocolImportLine(memberName) + "\n",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(content, prefix) {
			return strings.TrimLeft(strings.TrimPrefix(content, prefix), "\n")
		}
	}
	return content
}

func BuildWorkLogProtocol() string {
	var b strings.Builder

	b.WriteString("# Work Log Protocol\n\n")
	b.WriteString("Work logs are operational records, not chat replies.\n")
	b.WriteString("If you are asked to write, record, post, or leave a note, run `clier run note`.\n")
	b.WriteString("Use notes to record starts, completions, blockers, and hand-off context.\n")
	b.WriteString("Keep each note brief and factual. Include status, blockers, and next steps when relevant.\n\n")
	b.WriteString("Use the command below:\n\n")
	b.WriteString("```bash\nclier run note <<'EOF'\n<content>\nEOF\n```\n")
	b.WriteString("- Example: if someone asks you to leave a note, record a short status update with `clier run note`.\n")

	return b.String()
}

// BuildProtocol generates the team protocol CLAUDE.md content.
// Written to {teamRoot}/.clier/{member}-team-protocol.md and imported by
// each member's root CLAUDE.md.
func BuildProtocol(teamName, memberName string, relations domain.MemberRelations, membersByID map[int64]ProtocolMember) string {
	var b strings.Builder

	// Header
	b.WriteString("# Team Protocol\n\n")
	fmt.Fprintf(&b, "You are **%s**, operating as a member of team **%s**.\n\n", memberName, teamName)

	// Team Structure
	b.WriteString("## Team Structure\n\n")
	writeRelNames(&b, "Leaders", relations.Leaders, membersByID)
	writeRelNames(&b, "Workers", relations.Workers, membersByID)
	if len(relations.Leaders) == 0 && len(relations.Workers) == 0 {
		b.WriteString("- (none)\n")
	}

	// Communication
	b.WriteString("\n## Communication\n\n")
	b.WriteString("Team communication is operational, not conversational.\n")
	b.WriteString("When you need to contact another team member, run `clier run tell`.\n")
	b.WriteString("Do not use SendMessage, Agent, or any other built-in tool for team coordination.\n\n")
	b.WriteString("Use the numeric team member IDs shown below in the `--to` field.\n\n")
	b.WriteString("Use heredoc form to avoid shell escaping issues:\n\n")
	writeTellCommands(&b, relations, membersByID)
	b.WriteString("- Replies arrive directly in your terminal input. Do not poll for them.\n")
	b.WriteString("- Keep each message substantive and action-oriented. Avoid short fragments such as \"ok\" or \"hi\".\n")

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
