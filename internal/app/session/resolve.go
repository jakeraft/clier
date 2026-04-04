package session

import (
	"strings"

	"github.com/jakeraft/clier/internal/app/team"
	"github.com/jakeraft/clier/internal/domain"
)

// resolvePlaceholders replaces all {{CLIER_*}} placeholders in a MemberPlan
// and expands ~/ paths to the user's home directory.
func resolvePlaceholders(m domain.MemberPlan, base, homeDir, sessionID, claudeToken, codexAuth string) domain.MemberPlan {
	memberspace := strings.ReplaceAll(m.Workspace.Memberspace, team.PlaceholderBase, base)
	memberspace = strings.ReplaceAll(memberspace, team.PlaceholderSessionID, sessionID)

	replacer := strings.NewReplacer(
		team.PlaceholderMemberspace, memberspace,
		team.PlaceholderSessionID, sessionID,
		team.PlaceholderAuthClaude, claudeToken,
		team.PlaceholderAuthCodex, codexAuth,
	)

	m.Workspace.Memberspace = memberspace
	m.Terminal.Command = replacer.Replace(m.Terminal.Command)
	for i := range m.Workspace.Files {
		m.Workspace.Files[i].Path = replacer.Replace(m.Workspace.Files[i].Path)
		content := replacer.Replace(m.Workspace.Files[i].Content)
		if homeDir != "" {
			content = strings.ReplaceAll(content, "~/", homeDir+"/")
		}
		m.Workspace.Files[i].Content = content
	}

	return m
}
