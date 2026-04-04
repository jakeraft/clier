package session

// setAuth returns auth environment variable placeholders for the Claude CLI.
func setAuth() []string {
	return []string{"CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude}
}
