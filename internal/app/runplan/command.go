package runplan

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

var q = shellQuote

// shellQuote wraps a string in single quotes, escaping embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// quoteArgs quotes each element of args.
func quoteArgs(args []string) []string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = q(a)
	}
	return quoted
}

// configDirEnv returns the env-var assignment that controls where each CLI
// stores its dotfiles, using PlaceholderMemberspace as the base path.
func configDirEnv(binary domain.CliBinary) string {
	switch binary {
	case domain.BinaryClaude:
		return "CLAUDE_CONFIG_DIR=" + PlaceholderMemberspace + "/.claude"
	case domain.BinaryCodex:
		return "CODEX_HOME=" + PlaceholderMemberspace + "/.codex"
	default:
		return "HOME=" + PlaceholderMemberspace
	}
}

// buildEnv assembles the full set of environment variables for a member command.
func buildEnv(binary domain.CliBinary, sessionID, memberID string,
	authEnvs []string, userEnvs []domain.EnvSnapshot) []string {

	env := []string{
		configDirEnv(binary),
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
		parts = append(parts, fmt.Sprintf("export %s=%s", k, q(v)))
	}
	parts = append(parts, command)
	return strings.Join(parts, " &&\n")
}

// buildClaudeCommand builds the "cd <workDir> && claude <args...>" portion.
func buildClaudeCommand(binary domain.CliBinary, model string, systemArgs, customArgs []string,
	prompt, workDir string) string {

	args := []string{string(binary)}
	args = append(args, quoteArgs(systemArgs)...)
	args = append(args, "--model", q(model))
	args = append(args, quoteArgs(customArgs)...)
	base := fmt.Sprintf("cd %s &&\n%s", q(workDir), strings.Join(args, " "))
	if prompt != "" {
		return base + " --append-system-prompt \\\n" + q(prompt)
	}
	return base
}

// buildCodexCommand builds the "cd <workDir> && codex <args...>" portion.
func buildCodexCommand(binary domain.CliBinary, model string, systemArgs, customArgs []string,
	prompt, workDir string) string {

	args := []string{string(binary)}
	args = append(args, quoteArgs(systemArgs)...)
	args = append(args, "--model", q(model))
	args = append(args, quoteArgs(customArgs)...)
	base := fmt.Sprintf("cd %s &&\n%s", q(workDir), strings.Join(args, " "))
	if prompt != "" {
		return base + " -c \\\ndeveloper_instructions=" + q(prompt)
	}
	return base
}

// buildCommand returns the complete shell command for launching an agent,
// including environment variable exports.
func buildCommand(binary domain.CliBinary, model string, systemArgs, customArgs []string,
	prompt, sessionID, memberID string,
	authEnvs []string, userEnvs []domain.EnvSnapshot) (string, error) {

	workDir := PlaceholderMemberspace + "/project"

	var cmd string
	switch binary {
	case domain.BinaryClaude:
		cmd = buildClaudeCommand(binary, model, systemArgs, customArgs, prompt, workDir)
	case domain.BinaryCodex:
		cmd = buildCodexCommand(binary, model, systemArgs, customArgs, prompt, workDir)
	default:
		return "", fmt.Errorf("unknown binary: %s", binary)
	}

	env := buildEnv(binary, sessionID, memberID, authEnvs, userEnvs)
	return buildEnvCommand(cmd, env), nil
}
