package sprint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
)

// BuildCommand returns the shell command to launch an agent.
// Result format: "cd <workDir> && <binary> <args...>"
func BuildCommand(m domain.MemberSnapshot, workDir string) (command string, tempFiles []string) {
	switch m.Binary {
	case domain.BinaryClaude:
		return buildClaudeCommand(m, workDir), nil
	case domain.BinaryCodex:
		return buildCodexCommand(m, workDir)
	default:
		return "", nil
	}
}

func buildClaudeCommand(m domain.MemberSnapshot, workDir string) string {
	args := []string{string(m.Binary)}
	args = append(args, m.SystemArgs...)
	args = append(args, "--model", m.Model)
	args = append(args, "--session-id", m.MemberID)
	if m.ComposedPrompt != "" {
		args = append(args, "--append-system-prompt", shellQuote(m.ComposedPrompt))
	}
	args = append(args, m.CustomArgs...)
	return fmt.Sprintf("cd %s && %s", shellQuote(workDir), strings.Join(args, " "))
}

func buildCodexCommand(m domain.MemberSnapshot, workDir string) (string, []string) {
	instructionsFile := filepath.Join(os.TempDir(), fmt.Sprintf("clier-codex-instructions-%s.md", uuid.NewString()))
	_ = os.WriteFile(instructionsFile, []byte(m.ComposedPrompt), 0644)

	args := []string{string(m.Binary)}
	args = append(args, m.SystemArgs...)
	args = append(args, "--model", m.Model)
	args = append(args, "-c", fmt.Sprintf("model_instructions_file=%s", instructionsFile))
	args = append(args, m.CustomArgs...)
	return fmt.Sprintf("cd %s && %s", shellQuote(workDir), strings.Join(args, " ")), []string{instructionsFile}
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
	// .claude/settings.json — dotConfig
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

	// .claude.json — trust config
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
	if err := os.WriteFile(filepath.Join(memberHome, ".claude.json"), data, 0644); err != nil {
		return fmt.Errorf("write .claude.json: %w", err)
	}

	return nil
}

func writeCodexConfigs(m domain.MemberSnapshot, memberHome, workDir string) error {
	// .codex/config.toml — dotConfig + trust
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

	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("write config.toml: %w", err)
	}

	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
