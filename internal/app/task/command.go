package task

import (
	"fmt"
	"strings"
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

// systemEnvs returns clier infrastructure environment variables.
func systemEnvs(taskID, memberID string) []string {
	return []string{
		configDirEnv(),
		"CLIER_TASK_ID=" + taskID,
		"CLIER_MEMBER_ID=" + memberID,
	}
}

// authEnvs returns authentication environment variables for the Claude CLI.
func authEnvs() []string {
	return []string{"CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude}
}

// identityEnvs returns git identity environment variables derived from the team and member name.
func identityEnvs(teamName, memberName string) []string {
	name := teamName + "/" + memberName
	email := "noreply@clier.com"
	return []string{
		"GIT_AUTHOR_NAME=" + name,
		"GIT_AUTHOR_EMAIL=" + email,
		"GIT_COMMITTER_NAME=" + name,
		"GIT_COMMITTER_EMAIL=" + email,
	}
}

// buildEnv assembles the full set of environment variables for a member command.
func buildEnv(teamName, memberName, taskID, memberID string) []string {
	var env []string
	env = append(env, systemEnvs(taskID, memberID)...)
	env = append(env, authEnvs()...)
	env = append(env, identityEnvs(teamName, memberName)...)
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
// No --append-system-prompt — instructions go into CLAUDE.md.
func buildAgentCommand(model string, args []string, workDir string) string {
	parts := []string{"claude"}
	parts = append(parts, quoteArgs(args)...)
	parts = append(parts, "--model", shellQuote(model))
	return fmt.Sprintf("cd %s &&\n%s", shellQuote(workDir), strings.Join(parts, " "))
}

// buildCommand returns the complete shell command for launching an agent.
func buildCommand(model string, args []string, workDir, teamName, memberName, taskID, memberID string) string {
	cmd := buildAgentCommand(model, args, workDir)
	env := buildEnv(teamName, memberName, taskID, memberID)
	return buildEnvCommand(cmd, env)
}
