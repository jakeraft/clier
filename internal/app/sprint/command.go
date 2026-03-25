package sprint

import (
	"fmt"
	"os"
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
func BuildCommand(m domain.MemberSnapshot, prompt, workDir, sprintID, memberHome, dataDir string) (string, error) {
	var cmd string

	switch m.Binary {
	case domain.BinaryClaude:
		cmd = buildClaudeCommand(m, prompt, workDir)
	case domain.BinaryCodex:
		var err error
		cmd, err = buildCodexCommand(m, prompt, workDir, memberHome)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unknown binary: %s", m.Binary)
	}

	env := buildEnv(m, sprintID, memberHome, dataDir)
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

func buildCodexCommand(m domain.MemberSnapshot, prompt, workDir, memberHome string) (string, error) {
	instructionsFile := filepath.Join(memberHome, "codex-instructions.md")
	if err := os.WriteFile(instructionsFile, []byte(prompt), 0644); err != nil {
		return "", fmt.Errorf("write codex instructions: %w", err)
	}

	args := []string{string(m.Binary)}
	args = append(args, quoteArgs(m.SystemArgs)...)
	args = append(args, "--model", q(m.Model))
	args = append(args, "-c", "model_instructions_file="+q(instructionsFile))
	args = append(args, quoteArgs(m.CustomArgs)...)
	return fmt.Sprintf("cd %s && %s", q(workDir), strings.Join(args, " ")), nil
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

func buildEnv(m domain.MemberSnapshot, sprintID, memberHome, dataDir string) []string {
	env := []string{
		"HOME=" + memberHome,
		"CLIER_DATA_DIR=" + dataDir,
		"CLIER_SPRINT_ID=" + sprintID,
		"CLIER_MEMBER_ID=" + m.MemberID,
	}
	for _, e := range m.Environments {
		env = append(env, e.Key+"="+e.Value)
	}
	return env
}
