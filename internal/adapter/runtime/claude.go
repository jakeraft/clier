package runtime

import "fmt"

// ClaudeRuntime implements task.AgentRuntime for Claude Code.
type ClaudeRuntime struct{}

func (c *ClaudeRuntime) Binary() string { return "claude" }

func (c *ClaudeRuntime) ConfigDirEnv(memberspace string) string {
	return "CLAUDE_CONFIG_DIR=" + memberspace + "/.claude"
}

func (c *ClaudeRuntime) AuthEnvs(token string) []string {
	return []string{"CLAUDE_CODE_OAUTH_TOKEN=" + token}
}

func (c *ClaudeRuntime) InstructionFile() string  { return "CLAUDE.md" }
func (c *ClaudeRuntime) ConfigDir() string         { return ".claude" }
func (c *ClaudeRuntime) SettingsFile() string      { return "settings.json" }
func (c *ClaudeRuntime) ProjectConfigFile() string { return ".claude.json" }
func (c *ClaudeRuntime) SkillsDir() string         { return ".claude/skills" }

func (c *ClaudeRuntime) SystemConfig(memberspace string) string {
	return fmt.Sprintf(`{"hasCompletedOnboarding":true,"projects":{"%s/project":{"hasTrustDialogAccepted":true,"hasCompletedProjectOnboarding":true}}}`, memberspace)
}
