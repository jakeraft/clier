package sprint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
)

// shellQuote wraps a string in single quotes, escaping embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// BuildCommand returns the full shell command to launch an agent,
// including environment variable exports.
// Result format: "export K='V' && ... && cd <workDir> && <binary> <args...>"
func BuildCommand(m domain.MemberSnapshot, prompt, workDir string, env []string) (command string, tempFiles []string, err error) {
	var cmd string
	var tf []string

	switch m.Binary {
	case domain.BinaryClaude:
		cmd = buildClaudeCommand(m, prompt, workDir)
	case domain.BinaryCodex:
		cmd, tf, err = buildCodexCommand(m, prompt, workDir)
		if err != nil {
			return "", nil, err
		}
	default:
		return "", nil, fmt.Errorf("unknown binary: %s", m.Binary)
	}

	return buildEnvCommand(cmd, env), tf, nil
}

func buildClaudeCommand(m domain.MemberSnapshot, prompt, workDir string) string {
	q := shellQuote
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

func buildCodexCommand(m domain.MemberSnapshot, prompt, workDir string) (string, []string, error) {
	q := shellQuote
	instructionsFile := filepath.Join(os.TempDir(), fmt.Sprintf("clier-codex-instructions-%s.md", uuid.NewString()))
	if err := os.WriteFile(instructionsFile, []byte(prompt), 0644); err != nil {
		return "", nil, fmt.Errorf("write codex instructions: %w", err)
	}

	args := []string{string(m.Binary)}
	args = append(args, quoteArgs(m.SystemArgs)...)
	args = append(args, "--model", q(m.Model))
	args = append(args, "-c", fmt.Sprintf("model_instructions_file=%s", q(instructionsFile)))
	args = append(args, quoteArgs(m.CustomArgs)...)
	return fmt.Sprintf("cd %s && %s", q(workDir), strings.Join(args, " ")), []string{instructionsFile}, nil
}

func buildEnvCommand(command string, env []string) string {
	if len(env) == 0 {
		return command
	}
	q := shellQuote
	parts := make([]string, 0, len(env)+1)
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		parts = append(parts, fmt.Sprintf("export %s=%s", k, q(v)))
	}
	parts = append(parts, command)
	return strings.Join(parts, " && ")
}

func quoteArgs(args []string) []string {
	q := shellQuote
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = q(a)
	}
	return quoted
}

func cleanupTempFiles(files []string) {
	for _, f := range files {
		_ = os.Remove(f)
	}
}

// BuildEnv returns environment variables for the agent process.
func BuildEnv(m domain.MemberSnapshot, sprintID, memberHome string) []string {
	env := []string{
		"HOME=" + memberHome,
		"CLIER_SPRINT_ID=" + sprintID,
		"CLIER_MEMBER_ID=" + m.MemberID,
	}
	for _, e := range m.Environments {
		env = append(env, e.Key+"="+e.Value)
	}
	return env
}
