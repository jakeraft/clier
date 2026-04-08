package run

// AgentRuntime provides agent-specific behavior for command building
// and workspace layout. Each supported agent type has its own implementation.
type AgentRuntime interface {
	// Command building
	Binary() string
	ConfigDirEnv(memberspace string) string
	AuthEnvs(token string) []string

	// Workspace layout
	InstructionFile() string
	ConfigDir() string
	SettingsFile() string
	ProjectConfigFile() string
	SkillsDir() string
	SystemConfig(memberspace string) string
}
