package domain

// AgentProfile defines agent-specific paths, markers, and commands.
type AgentProfile struct {
	InstructionFile string // root instruction file: "CLAUDE.md", "AGENTS.md", "GEMINI.md"
	SettingsDir     string // agent config directory: ".claude", ".codex", ".gemini"
	SettingsFile    string // settings file name: "settings.json", "config.toml"
	SkillsDir       string // skills subdirectory under SettingsDir, empty if not supported
	ReadyMarker     string // pane title substring indicating agent is ready, empty to skip
	ExitCommand     string // command to gracefully exit the agent, empty to skip
	InstructionKind string // server resource kind: "instruction"
	SettingsKind    string // server resource kind: "claude-setting", "codex-setting"
}

// ProfileFor returns the AgentProfile for the given agent type.
func ProfileFor(agentType string) (AgentProfile, error) {
	switch agentType {
	case "claude", "":
		return AgentProfile{
			InstructionFile: "CLAUDE.md",
			SettingsDir:     ".claude",
			SettingsFile:    "settings.json",
			SkillsDir:       "skills",
			ReadyMarker:     "Claude",
			ExitCommand:     "/exit",
			InstructionKind: "instruction",
			SettingsKind:    "claude-setting",
		}, nil
	case "codex":
		return AgentProfile{
			InstructionFile: "AGENTS.md",
			SettingsDir:     ".codex",
			SettingsFile:    "config.toml",
			SkillsDir:       "skills",
			ReadyMarker:     "",
			ExitCommand:     "/exit",
			InstructionKind: "instruction",
			SettingsKind:    "codex-setting",
		}, nil
	default:
		return AgentProfile{}, &Fault{
			Kind:    KindUnsupportedKind,
			Subject: map[string]string{"resource_kind": agentType},
		}
	}
}
