package sprint

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

var q = shellQuote

// shellQuote wraps a string in single quotes, escaping embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// BuildCommand returns the full shell command to launch an agent,
// including environment variable exports.
// Result format: "export K='V' && ... && cd <workDir> && <binary> <args...>"
func BuildCommand(m domain.MemberSnapshot, prompt, workDir, sprintID, memberHome string) (string, error) {
	var cmd string

	switch m.Binary {
	case domain.BinaryClaude:
		cmd = buildClaudeCommand(m, prompt, workDir)
	case domain.BinaryCodex:
		cmd = buildCodexCommand(m, prompt, workDir)
	default:
		return "", fmt.Errorf("unknown binary: %s", m.Binary)
	}

	env := buildEnv(m, sprintID, memberHome)
	return buildEnvCommand(cmd, env), nil
}

func buildClaudeCommand(m domain.MemberSnapshot, prompt, workDir string) string {
	args := []string{string(m.Binary)}
	args = append(args, quoteArgs(m.SystemArgs)...)
	args = append(args, "--model", q(m.Model))
	args = append(args, "--session-id", q(m.MemberID))
	if prompt != "" {
		args = append(args, "--append-system-prompt", q(prompt))
	}
	args = append(args, quoteArgs(m.CustomArgs)...)
	return fmt.Sprintf("cd %s && %s", q(workDir), strings.Join(args, " "))
}

func buildCodexCommand(m domain.MemberSnapshot, prompt, workDir string) string {
	args := []string{string(m.Binary)}
	args = append(args, quoteArgs(m.SystemArgs)...)
	args = append(args, "--model", q(m.Model))
	if prompt != "" {
		args = append(args, "-c", "developer_instructions="+q(prompt))
	}
	args = append(args, quoteArgs(m.CustomArgs)...)
	return fmt.Sprintf("cd %s && %s", q(workDir), strings.Join(args, " "))
}

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
	return strings.Join(parts, " && ")
}

func quoteArgs(args []string) []string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = q(a)
	}
	return quoted
}

// configDirEnv returns the environment variable assignment that controls
// where each CLI stores its dotfiles, avoiding a full HOME override.
func configDirEnv(binary domain.CliBinary, memberHome string) string {
	switch binary {
	case domain.BinaryClaude:
		return "CLAUDE_CONFIG_DIR=" + filepath.Join(memberHome, ".claude")
	case domain.BinaryCodex:
		return "CODEX_HOME=" + filepath.Join(memberHome, ".codex")
	default:
		return "HOME=" + memberHome
	}
}

func buildEnv(m domain.MemberSnapshot, sprintID, memberHome string) []string {
	env := []string{
		configDirEnv(m.Binary, memberHome),
		"CLIER_SPRINT_ID=" + sprintID,
		"CLIER_MEMBER_ID=" + m.MemberID,
	}
	for _, e := range m.Envs {
		env = append(env, e.Key+"="+e.Value)
	}
	return env
}
