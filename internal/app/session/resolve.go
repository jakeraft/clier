package session

import (
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// resolvePlaceholders replaces all {{CLIER_*}} placeholders in a MemberPlan
// and expands ~/ paths to the user's home directory.
func resolvePlaceholders(m domain.MemberPlan, base, homeDir, sessionID, claudeToken, codexAuth string) domain.MemberPlan {
	memberspace := strings.ReplaceAll(m.Workspace.Memberspace, PlaceholderBase, base)
	memberspace = strings.ReplaceAll(memberspace, PlaceholderSessionID, sessionID)

	replacer := strings.NewReplacer(
		PlaceholderMemberspace, memberspace,
		PlaceholderSessionID, sessionID,
		PlaceholderAuthClaude, claudeToken,
		PlaceholderAuthCodex, codexAuth,
	)

	m.Workspace.Memberspace = memberspace
	m.Terminal.Command = replacer.Replace(m.Terminal.Command)

	// Copy the slice to avoid mutating the original plan's shared backing array.
	files := make([]domain.FileEntry, len(m.Workspace.Files))
	for i, f := range m.Workspace.Files {
		path := replacer.Replace(f.Path)
		content := replacer.Replace(f.Content)
		if homeDir != "" {
			content = strings.ReplaceAll(content, "~/", homeDir+"/")
		}
		files[i] = domain.FileEntry{Path: path, Content: content}
	}
	m.Workspace.Files = files

	return m
}
