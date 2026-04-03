package sprint

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
	toml "github.com/pelletier/go-toml/v2"
)

// buildClaudeFiles generates the Claude config files from DotConfig.
func buildClaudeFiles(dotConfig domain.DotConfig, workDir, homeDir string) ([]domain.FileEntry, error) {
	settingsData, err := json.MarshalIndent(dotConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal settings: %w", err)
	}
	settingsData = expandTildePaths(settingsData, homeDir)

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
		{Path: ".claude/settings.json", Content: string(settingsData)},
		{Path: ".claude/.claude.json", Content: string(trustData)},
	}, nil
}

// buildCodexFiles generates the Codex config files from DotConfig.
func buildCodexFiles(dotConfig domain.DotConfig, workDir string) ([]domain.FileEntry, error) {
	config := make(map[string]any, len(dotConfig))
	maps.Copy(config, dotConfig)
	config["projects"] = map[string]any{
		workDir: map[string]any{
			"trust_level": "trusted",
		},
	}

	data, err := toml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal codex config: %w", err)
	}

	return []domain.FileEntry{
		{Path: ".codex/config.toml", Content: string(data)},
	}, nil
}

func expandTildePaths(data []byte, homeDir string) []byte {
	if homeDir == "" {
		return data
	}
	return []byte(strings.ReplaceAll(string(data), "~/", homeDir+"/"))
}
