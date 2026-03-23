package sprint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/terminal"
)

// BuildCommand returns the full shell command to launch an agent,
// including environment variable exports.
// Result format: "export K='V' && ... && cd <workDir> && <binary> <args...>"
func BuildCommand(m domain.MemberSnapshot, workDir string, env []string) (command string, tempFiles []string, err error) {
	var cmd string
	var tf []string

	switch m.Binary {
	case domain.BinaryClaude:
		cmd = buildClaudeCommand(m, workDir)
	case domain.BinaryCodex:
		cmd, tf, err = buildCodexCommand(m, workDir)
		if err != nil {
			return "", nil, err
		}
	default:
		return "", nil, fmt.Errorf("unknown binary: %s", m.Binary)
	}

	return buildEnvCommand(cmd, env), tf, nil
}

func buildClaudeCommand(m domain.MemberSnapshot, workDir string) string {
	q := terminal.ShellQuote
	args := []string{string(m.Binary)}
	args = append(args, quoteArgs(m.SystemArgs)...)
	args = append(args, "--model", q(m.Model))
	args = append(args, "--session-id", q(m.MemberID))
	if m.ComposedPrompt != "" {
		args = append(args, "--append-system-prompt", q(m.ComposedPrompt))
	}
	args = append(args, quoteArgs(m.CustomArgs)...)
	return fmt.Sprintf("cd %s && %s", q(workDir), strings.Join(args, " "))
}

func buildCodexCommand(m domain.MemberSnapshot, workDir string) (string, []string, error) {
	q := terminal.ShellQuote
	instructionsFile := filepath.Join(os.TempDir(), fmt.Sprintf("clier-codex-instructions-%s.md", uuid.NewString()))
	if err := os.WriteFile(instructionsFile, []byte(m.ComposedPrompt), 0644); err != nil {
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
	q := terminal.ShellQuote
	parts := make([]string, 0, len(env)+1)
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		parts = append(parts, fmt.Sprintf("export %s=%s", k, q(v)))
	}
	parts = append(parts, command)
	return strings.Join(parts, " && ")
}

func quoteArgs(args []string) []string {
	q := terminal.ShellQuote
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = q(a)
	}
	return quoted
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

// WriteConfigs writes CLI-specific config files to the member's home directory.
func WriteConfigs(m domain.MemberSnapshot, memberHome, workDir string) error {
	switch m.Binary {
	case domain.BinaryClaude:
		return writeClaudeConfigs(m, memberHome, workDir)
	case domain.BinaryCodex:
		return writeCodexConfigs(m, memberHome, workDir)
	default:
		return fmt.Errorf("unknown binary: %s", m.Binary)
	}
}

func writeClaudeConfigs(m domain.MemberSnapshot, memberHome, workDir string) error {
	if len(m.DotConfig) > 0 {
		claudeDir := filepath.Join(memberHome, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			return fmt.Errorf("create .claude dir: %w", err)
		}
		data, err := json.MarshalIndent(m.DotConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal dotconfig: %w", err)
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644); err != nil {
			return fmt.Errorf("write settings.json: %w", err)
		}
	}

	trust := map[string]any{
		"projects": map[string]any{
			workDir: map[string]any{
				"hasTrustDialogAccepted":        true,
				"hasCompletedProjectOnboarding": true,
			},
		},
	}
	data, err := json.MarshalIndent(trust, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal trust config: %w", err)
	}
	return os.WriteFile(filepath.Join(memberHome, ".claude.json"), data, 0644)
}

func writeCodexConfigs(m domain.MemberSnapshot, memberHome, workDir string) error {
	codexDir := filepath.Join(memberHome, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return fmt.Errorf("create .codex dir: %w", err)
	}

	var b strings.Builder
	for k, v := range m.DotConfig {
		fmt.Fprintf(&b, "%s = %q\n", k, fmt.Sprint(v))
	}
	b.WriteString("\n[projects]\n")
	fmt.Fprintf(&b, "[projects.%q]\n", workDir)
	b.WriteString("trust_level = \"trusted\"\n")

	return os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(b.String()), 0644)
}
