package session

import (
	"encoding/json"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// buildClaudeFiles generates Claude config files (settings.json + trust config)
// with paths using memberspacePlaceholder instead of absolute paths.
func buildClaudeFiles(dotConfig domain.DotConfig, workDir, memberspacePlaceholder string) ([]domain.FileEntry, error) {
	settingsData, err := json.MarshalIndent(dotConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal settings: %w", err)
	}

	trust := map[string]any{
		"hasCompletedOnboarding": true,
		"projects": map[string]any{
			workDir: map[string]any{
				"hasTrustDialogAccepted":        true,
				"hasCompletedProjectOnboarding": true,
			},
		},
	}
	trustData, err := json.MarshalIndent(trust, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal trust: %w", err)
	}

	return []domain.FileEntry{
		{Path: memberspacePlaceholder + "/.claude/settings.json", Content: string(settingsData)},
		{Path: memberspacePlaceholder + "/.claude/.claude.json", Content: string(trustData)},
	}, nil
}
