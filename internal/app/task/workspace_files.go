package task

import (
	"encoding/json"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

// buildWorkspaceFiles creates all file entries for a member's workspace.
// Clearly separates system-generated and user-defined content.
func buildWorkspaceFiles(rt AgentRuntime, memberspace, systemClaudeMd, userClaudeMd, userClaudeSettings, systemProjectConfig, userProjectConfig string, userSkills []resource.Skill) []domain.FileEntry {
	var files []domain.FileEntry

	// Instruction file -> {memberspace}/project/{rt.InstructionFile()}
	claudeMdContent := systemClaudeMd
	if userClaudeMd != "" {
		claudeMdContent += "\n\n---\n\n" + userClaudeMd
	}
	if claudeMdContent != "" {
		files = append(files, domain.FileEntry{
			Path:    memberspace + "/project/" + rt.InstructionFile(),
			Content: claudeMdContent,
		})
	}

	// Settings -> {memberspace}/{rt.ConfigDir()}/{rt.SettingsFile()}
	if userClaudeSettings != "" {
		files = append(files, domain.FileEntry{
			Path:    fmt.Sprintf("%s/%s/%s", memberspace, rt.ConfigDir(), rt.SettingsFile()),
			Content: userClaudeSettings,
		})
	}

	// Project config -> {memberspace}/{rt.ConfigDir()}/{rt.ProjectConfigFile()}
	projectConfigContent := mergeJSON(systemProjectConfig, userProjectConfig)
	if projectConfigContent != "" {
		files = append(files, domain.FileEntry{
			Path:    fmt.Sprintf("%s/%s/%s", memberspace, rt.ConfigDir(), rt.ProjectConfigFile()),
			Content: projectConfigContent,
		})
	}

	// Skills -> {memberspace}/{rt.SkillsDir()}/{name}/SKILL.md
	for _, skill := range userSkills {
		files = append(files, domain.FileEntry{
			Path:    fmt.Sprintf("%s/%s/%s/SKILL.md", memberspace, rt.SkillsDir(), skill.Name),
			Content: skill.Content,
		})
	}

	return files
}

// NOTE: Claude-specific JSON merge. Other agent runtimes may need different merge strategy (e.g. TOML for Codex).
// mergeJSON merges two JSON object strings. System keys are set first, then user keys override/extend.
// The "projects" key is deep-merged: system project entries and user project entries are combined.
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

	var systemMap map[string]json.RawMessage
	_ = json.Unmarshal([]byte(systemJSON), &systemMap)
	if systemMap == nil {
		systemMap = make(map[string]json.RawMessage)
	}

	var userMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(userJSON), &userMap); err == nil {
		for k, v := range userMap {
			if k == "projects" {
				// Deep merge projects: combine system and user project entries
				systemMap[k] = mergeJSONObjects(systemMap[k], v)
			} else {
				systemMap[k] = v
			}
		}
	}

	out, err := json.Marshal(systemMap)
	if err != nil {
		return systemJSON
	}
	return string(out)
}

// mergeJSONObjects merges two JSON objects, with b's keys overriding a's.
func mergeJSONObjects(a, b json.RawMessage) json.RawMessage {
	var aMap, bMap map[string]json.RawMessage
	_ = json.Unmarshal(a, &aMap)
	if aMap == nil {
		aMap = make(map[string]json.RawMessage)
	}
	if err := json.Unmarshal(b, &bMap); err != nil {
		return b
	}
	for k, v := range bMap {
		aMap[k] = v
	}
	out, err := json.Marshal(aMap)
	if err != nil {
		return b
	}
	return out
}
