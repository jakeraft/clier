package session

import (
	"strings"

	"github.com/jakeraft/clier/internal/app/runplan"
	"github.com/jakeraft/clier/internal/domain"
)

// resolvePlaceholders replaces all {{CLIER_*}} placeholders in a MemberSessionPlan
// and expands ~/ paths to the user's home directory.
func resolvePlaceholders(m domain.MemberSessionPlan, base, homeDir, sessionID, claudeToken, codexAuth string) domain.MemberSessionPlan {
	memberspace := strings.ReplaceAll(m.Workspace.Memberspace, runplan.PlaceholderBase, base)

	replacer := strings.NewReplacer(
		runplan.PlaceholderMemberspace, memberspace,
		runplan.PlaceholderSessionID, sessionID,
		runplan.PlaceholderAuthClaude, claudeToken,
		runplan.PlaceholderAuthCodex, codexAuth,
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
