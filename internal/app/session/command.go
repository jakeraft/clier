package session

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// shellQuote wraps a string in single quotes, escaping embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// quoteArgs quotes each element of args.
func quoteArgs(args []string) []string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shellQuote(a)
	}
	return quoted
}

// configDirEnv returns the env-var assignment that controls where the CLI
// stores its dotfiles, using PlaceholderMemberspace as the base path.
func configDirEnv() string {
	return "CLAUDE_CONFIG_DIR=" + PlaceholderMemberspace + "/.claude"
}

// buildEnv assembles the full set of environment variables for a member command.
func buildEnv(sessionID, memberID string,
	authEnvs []string, userEnvs []domain.EnvSnapshot) []string {

	env := []string{
		configDirEnv(),
		"CLIER_SESSION_ID=" + sessionID,
		"CLIER_MEMBER_ID=" + memberID,
	}
	env = append(env, authEnvs...)
	for _, e := range userEnvs {
		env = append(env, e.Key+"="+e.Value)
	}
	return env
}

// buildEnvCommand prepends "export K='V' && ..." to a command string.
func buildEnvCommand(command string, env []string) string {
	if len(env) == 0 {
		return command
	}
	parts := make([]string, 0, len(env)+1)
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		parts = append(parts, fmt.Sprintf("export %s=%s", k, shellQuote(v)))
	}
	parts = append(parts, command)
	return strings.Join(parts, " &&\n")
}

// buildAgentCommand builds the "cd <workDir> && claude <args...>" portion.
func buildAgentCommand(model string, systemArgs, customArgs []string,
	prompt, workDir string) string {

	args := []string{"claude"}
	args = append(args, quoteArgs(systemArgs)...)
	args = append(args, "--model", shellQuote(model))
	args = append(args, quoteArgs(customArgs)...)
	base := fmt.Sprintf("cd %s &&\n%s", shellQuote(workDir), strings.Join(args, " "))
	if prompt != "" {
		return base + " --append-system-prompt \\\n" + shellQuote(prompt)
	}
	return base
}

// buildCommand returns the complete shell command for launching an agent,
// including environment variable exports.
func buildCommand(model string, systemArgs, customArgs []string,
	prompt, sessionID, memberID string,
	authEnvs []string, userEnvs []domain.EnvSnapshot) string {

	workDir := PlaceholderMemberspace + "/project"
	cmd := buildAgentCommand(model, systemArgs, customArgs, prompt, workDir)
	env := buildEnv(sessionID, memberID, authEnvs, userEnvs)
	return buildEnvCommand(cmd, env)
}
