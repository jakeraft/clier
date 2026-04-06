package task

// buildAuthEnvs returns auth environment variable placeholders for the Claude CLI.
func buildAuthEnvs() []string {
	return []string{"CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude}
}
