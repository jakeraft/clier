package run

import (
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// expandPlaceholders replaces all {{CLIER_*}} placeholders in a MemberPlan
// and expands ~/ paths to the user's home directory.
// This is the expand phase: plan with placeholders -> plan with concrete paths.
func expandPlaceholders(m domain.MemberPlan, base, homeDir, runID, authToken string) domain.MemberPlan {
	memberspace := strings.ReplaceAll(m.Workspace.Memberspace, PlaceholderBase, base)
	memberspace = strings.ReplaceAll(memberspace, PlaceholderRunID, runID)

	replacer := strings.NewReplacer(
		PlaceholderMemberspace, memberspace,
		PlaceholderRunID, runID,
		PlaceholderAuthClaude, authToken,
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
