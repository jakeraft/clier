package run

import (
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

// buildWorkspaceFiles creates all file entries for a member's workspace.
// Clearly separates system-generated and user-defined content.
func buildWorkspaceFiles(rt AgentRuntime, memberspace, systemClaudeMd, userClaudeMd, userClaudeSettings, systemProjectConfig string, userSkills []resource.Skill) []domain.FileEntry {
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
	if systemProjectConfig != "" {
		files = append(files, domain.FileEntry{
			Path:    fmt.Sprintf("%s/%s/%s", memberspace, rt.ConfigDir(), rt.ProjectConfigFile()),
			Content: systemProjectConfig,
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

