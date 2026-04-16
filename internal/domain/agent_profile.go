package domain

import "fmt"

// AgentProfile defines agent-specific paths, markers, and commands.
type AgentProfile struct {
	InstructionFile   string // root instruction file: "CLAUDE.md", "AGENTS.md", "GEMINI.md"
	SettingsDir       string // agent config directory: ".claude", ".codex", ".gemini"
	SettingsFile      string // settings file name: "settings.json", "config.toml"
	LocalSettingsFile string // local overlay file name, empty if not applicable
	SkillsDir         string // skills subdirectory under SettingsDir, empty if not supported
	ReadyMarker       string // pane title substring indicating agent is ready, empty to skip
	ExitCommand       string // command to gracefully exit the agent, empty to skip
	HomeExcludeKey    string // JSON key for home dir exclusion in local settings, empty to skip
	InstructionKind   string // server resource kind: "instruction"
	SettingsKind      string // server resource kind: "claude-setting", "codex-setting"
}

// ProfileFor returns the AgentProfile for the given agent type.
func ProfileFor(agentType string) (AgentProfile, error) {
	switch agentType {
	case "claude", "":
		return AgentProfile{
			InstructionFile:   "CLAUDE.md",
			SettingsDir:       ".claude",
			SettingsFile:      "settings.json",
			LocalSettingsFile: "settings.local.json",
			SkillsDir:         "skills",
			ReadyMarker:       "Claude",
			ExitCommand:       "/exit",
			HomeExcludeKey:    "claudeMdExcludes",
			InstructionKind:   "instruction",
			SettingsKind:      "claude-setting",
		}, nil
	case "codex":
		return AgentProfile{
			InstructionFile:   "AGENTS.md",
			SettingsDir:       ".codex",
			SettingsFile:      "config.toml",
			LocalSettingsFile: "",
			SkillsDir:         "skills",
			ReadyMarker:       "",
			ExitCommand:       "/exit",
			HomeExcludeKey:    "",
			InstructionKind:   "instruction",
			SettingsKind:      "codex-setting",
		}, nil
	default:
		return AgentProfile{}, fmt.Errorf("unknown agent type: %q", agentType)
	}
}
