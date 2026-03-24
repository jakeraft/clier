package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// Workspace manages sprint member filesystem environments.
type Workspace struct {
	baseDir       string
	copyAuth      func(binary domain.CliBinary, destHome string) error
	getCredential func(host string) (string, error)
}

func New(baseDir string, copyAuth func(domain.CliBinary, string) error, getCredential func(string) (string, error)) *Workspace {
	return &Workspace{baseDir: baseDir, copyAuth: copyAuth, getCredential: getCredential}
}

// PrepareMember creates the member's isolated workspace: directories, auth, configs, and git repo.
func (w *Workspace) PrepareMember(ctx context.Context, sprintID string, m domain.MemberSnapshot) (memberHome, workDir string, err error) {
	memberHome = filepath.Join(w.baseDir, sprintID, m.MemberID)
	workDir = filepath.Join(memberHome, "project")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", "", fmt.Errorf("create workspace: %w", err)
	}

	if err := w.copyAuth(m.Binary, memberHome); err != nil {
		return "", "", fmt.Errorf("copy auth: %w", err)
	}

	if err := writeConfigs(m, memberHome, workDir); err != nil {
		return "", "", fmt.Errorf("write configs: %w", err)
	}

	if err := w.setupGit(ctx, m, workDir); err != nil {
		return "", "", fmt.Errorf("setup git: %w", err)
	}

	return memberHome, workDir, nil
}

// Cleanup removes all workspace files for a sprint.
func (w *Workspace) Cleanup(sprintID string) error {
	return os.RemoveAll(filepath.Join(w.baseDir, sprintID))
}

func (w *Workspace) setupGit(ctx context.Context, m domain.MemberSnapshot, workDir string) error {
	if m.GitRepo == nil {
		return exec.CommandContext(ctx, "git", "init", workDir).Run()
	}

	cloneURL := m.GitRepo.URL
	if host := extractHost(cloneURL); host != "" {
		if token, err := w.getCredential(host); err == nil {
			cloneURL = injectCredential(cloneURL, token)
		}
	}

	if err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", cloneURL, workDir).Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", m.GitRepo.URL, err)
	}
	return nil
}

func writeConfigs(m domain.MemberSnapshot, memberHome, workDir string) error {
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

func extractHost(gitURL string) string {
	u, err := url.Parse(gitURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

func injectCredential(gitURL, token string) string {
	u, err := url.Parse(gitURL)
	if err != nil {
		return gitURL
	}
	u.User = url.UserPassword("x-access-token", token)
	return u.String()
}

