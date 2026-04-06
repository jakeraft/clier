package task

import (
	"encoding/json"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

// buildWorkspaceFiles creates all file entries for a member's workspace.
// Clearly separates system-generated and user-defined content.
func buildWorkspaceFiles(memberspace, systemClaudeMd, userClaudeMd, userSettings, systemClaudeJson, userClaudeJson string, userSkills []resource.Skill) []domain.FileEntry {
	var files []domain.FileEntry

	// CLAUDE.md -> {memberspace}/project/CLAUDE.md
	claudeMdContent := systemClaudeMd
	if userClaudeMd != "" {
		claudeMdContent += "\n\n---\n\n" + userClaudeMd
	}
	if claudeMdContent != "" {
		files = append(files, domain.FileEntry{
			Path:    memberspace + "/project/CLAUDE.md",
			Content: claudeMdContent,
		})
	}

	// settings.json -> {memberspace}/.claude/settings.json
	if userSettings != "" {
		files = append(files, domain.FileEntry{
			Path:    memberspace + "/.claude/settings.json",
			Content: userSettings,
		})
	}

	// .claude.json -> {memberspace}/.claude/.claude.json
	claudeJsonContent := mergeJSON(systemClaudeJson, userClaudeJson)
	if claudeJsonContent != "" {
		files = append(files, domain.FileEntry{
			Path:    memberspace + "/.claude/.claude.json",
			Content: claudeJsonContent,
		})
	}

	// Skills -> {memberspace}/.claude/skills/{name}/SKILL.md
	for _, skill := range userSkills {
		files = append(files, domain.FileEntry{
			Path:    fmt.Sprintf("%s/.claude/skills/%s/SKILL.md", memberspace, skill.Name),
			Content: skill.Content,
		})
	}

	return files
}

// buildSystemClaudeJson generates the system-required .claude.json fields.
// The projects key ensures Claude recognizes the workspace directory.
func buildSystemClaudeJson(memberspace string) string {
	return fmt.Sprintf(`{"hasCompletedOnboarding":true,"projects":{"%s/project":{"hasTrustDialogAccepted":true,"hasCompletedProjectOnboarding":true}}}`, memberspace)
}

// mergeJSON merges two JSON object strings. System keys are set first, then user keys override/extend.
// Returns empty string if both inputs are empty.
func mergeJSON(systemJSON, userJSON string) string {
	if systemJSON == "" && userJSON == "" {
		return ""
	}
	if systemJSON == "" {
		return userJSON
	}
	if userJSON == "" {
		return systemJSON
	}

	// Parse both
	var merged map[string]json.RawMessage
	_ = json.Unmarshal([]byte(systemJSON), &merged)
	if merged == nil {
		merged = make(map[string]json.RawMessage)
	}

	var userMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(userJSON), &userMap); err == nil {
		for k, v := range userMap {
			merged[k] = v
		}
	}

	out, err := json.Marshal(merged)
	if err != nil {
		return systemJSON
	}
	return string(out)
}
