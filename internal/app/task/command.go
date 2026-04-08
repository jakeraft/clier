package task

import (
	"fmt"
	"strings"
)

// shellQuote wraps a string in single quotes, escaping embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// systemEnvs returns clier infrastructure environment variables.
func systemEnvs(rt AgentRuntime, memberspace, taskID, memberID string) []string {
	return []string{
		rt.ConfigDirEnv(memberspace),
		"CLIER_TASK_ID=" + taskID,
		"CLIER_MEMBER_ID=" + memberID,
	}
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
func buildEnv(rt AgentRuntime, memberspace, teamName, memberName, taskID, memberID, authPlaceholder string) []string {
	var env []string
	env = append(env, systemEnvs(rt, memberspace, taskID, memberID)...)
	env = append(env, rt.AuthEnvs(authPlaceholder)...)
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

// buildAgentCommand builds the "cd <workDir> && <command>" portion.
// The command string contains the binary and all CLI flags (e.g. "claude --dangerously-skip-permissions").
// No --append-system-prompt — instructions go into the instruction file.
func buildAgentCommand(command string, workDir string) string {
	return fmt.Sprintf("cd %s &&\n%s", shellQuote(workDir), command)
}

// buildCommand returns the complete shell command for launching an agent.
func buildCommand(rt AgentRuntime, command, workDir, memberspace, teamName, memberName, taskID, memberID, authPlaceholder string) string {
	cmd := buildAgentCommand(command, workDir)
	env := buildEnv(rt, memberspace, teamName, memberName, taskID, memberID, authPlaceholder)
	return buildEnvCommand(cmd, env)
}
