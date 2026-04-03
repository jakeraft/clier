package sprint

import (
	"encoding/json"
	"maps"
	"os"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
	toml "github.com/pelletier/go-toml/v2"
)

// buildClaudeFiles generates the Claude config files from DotConfig.
func buildClaudeFiles(dotConfig domain.DotConfig, workDir string) []domain.FileEntry {
	settingsData, _ := json.MarshalIndent(dotConfig, "", "  ")
	settingsData = expandTildePaths(settingsData)

	trust := map[string]any{
		"hasCompletedOnboarding": true,
		"projects": map[string]any{
			workDir: map[string]any{
				"hasTrustDialogAccepted":        true,
				"hasCompletedProjectOnboarding": true,
			},
		},
	}
	trustData, _ := json.MarshalIndent(trust, "", "  ")

	return []domain.FileEntry{
		{Path: ".claude/settings.json", Content: string(settingsData)},
		{Path: ".claude/.claude.json", Content: string(trustData)},
	}
}

// buildCodexFiles generates the Codex config files from DotConfig.
func buildCodexFiles(dotConfig domain.DotConfig, workDir string) []domain.FileEntry {
	config := make(map[string]any, len(dotConfig))
	maps.Copy(config, dotConfig)
	config["projects"] = map[string]any{
		workDir: map[string]any{
			"trust_level": "trusted",
		},
	}

	data, _ := toml.Marshal(config)

	return []domain.FileEntry{
		{Path: ".codex/config.toml", Content: string(data)},
	}
}

func expandTildePaths(data []byte) []byte {
	home, err := os.UserHomeDir()
	if err != nil {
		return data
	}
	return []byte(strings.ReplaceAll(string(data), "~/", home+"/"))
}
