package session

import "github.com/jakeraft/clier/internal/domain"

// buildClaudeFiles wraps the profile's JSON strings into FileEntry values.
// Placeholder replacement is handled later by expandPlaceholders.
func buildClaudeFiles(settingsJSON, claudeJSON, memberspacePlaceholder string) []domain.FileEntry {
	return []domain.FileEntry{
		{Path: memberspacePlaceholder + "/.claude/settings.json", Content: settingsJSON},
		{Path: memberspacePlaceholder + "/.claude/.claude.json", Content: claudeJSON},
	}
}
