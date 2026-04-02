package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
	toml "github.com/pelletier/go-toml/v2"
)

// AuthCopier copies CLI auth files to a destination home directory.
type AuthCopier interface {
	Check(binary domain.CliBinary) error
	CopyTo(binary domain.CliBinary, destHome string) error
}

// Workspace manages sprint member filesystem environments.
type Workspace struct {
	baseDir string
	auth    AuthCopier
}

func New(baseDir string, auth AuthCopier) *Workspace {
	return &Workspace{baseDir: baseDir, auth: auth}
}

// Prepare creates the sprint directory and sets up isolated environments for all members.
func (w *Workspace) Prepare(ctx context.Context, sprintID string, snapshot domain.SprintSnapshot) error {
	if len(snapshot.Members) == 0 {
		return nil
	}

	sprintDir := filepath.Join(w.baseDir, sprintID)
	if err := os.MkdirAll(sprintDir, 0755); err != nil {
		return fmt.Errorf("create sprint dir: %w", err)
	}

	// Cleanup entire sprint directory on any failure
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(sprintDir)
		}
	}()

	// Preflight: check auth for all required binaries before creating anything.
	checked := make(map[domain.CliBinary]bool)
	for _, m := range snapshot.Members {
		if !checked[m.Binary] {
			checked[m.Binary] = true
			if err := w.auth.Check(m.Binary); err != nil {
				return err
			}
		}
	}

	for _, m := range snapshot.Members {
		if err := w.prepareMember(ctx, m); err != nil {
			return fmt.Errorf("prepare member %s: %w", m.MemberName, err)
		}
	}

	success = true
	return nil
}

// Cleanup removes all workspace files for a sprint.
func (w *Workspace) Cleanup(sprintID string) error {
	return os.RemoveAll(filepath.Join(w.baseDir, sprintID))
}

func (w *Workspace) prepareMember(ctx context.Context, m domain.SprintMemberSnapshot) error {
	if err := os.MkdirAll(m.WorkDir, 0755); err != nil {
		return fmt.Errorf("create member dir: %w", err)
	}

	if err := w.auth.CopyTo(m.Binary, m.Home); err != nil {
		return fmt.Errorf("copy auth: %w", err)
	}

	if err := writeConfigs(m, m.Home, m.WorkDir); err != nil {
		return fmt.Errorf("write configs: %w", err)
	}

	if err := w.setupGit(ctx, m, m.WorkDir); err != nil {
		return fmt.Errorf("setup git: %w", err)
	}

	return nil
}

func (w *Workspace) setupGit(ctx context.Context, m domain.SprintMemberSnapshot, workDir string) error {
	if m.GitRepo == nil {
		return exec.CommandContext(ctx, "git", "init", workDir).Run()
	}
	if err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", m.GitRepo.URL, workDir).Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", m.GitRepo.URL, err)
	}
	return nil
}

func writeConfigs(m domain.SprintMemberSnapshot, memberHome, workDir string) error {
	switch m.Binary {
	case domain.BinaryClaude:
		return writeClaudeConfigs(m, memberHome, workDir)
	case domain.BinaryCodex:
		return writeCodexConfigs(m, memberHome, workDir)
	default:
		return fmt.Errorf("unknown binary: %s", m.Binary)
	}
}

func writeClaudeConfigs(m domain.SprintMemberSnapshot, memberHome, workDir string) error {
	claudeDir := filepath.Join(memberHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("create .claude dir: %w", err)
	}

	data, err := json.MarshalIndent(m.DotConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	data = expandTildePaths(data)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}

	// CLAUDE_CONFIG_DIR points to .claude/, so .claude.json lives inside it.
	trust := map[string]any{
		"hasCompletedOnboarding": true,
		"projects": map[string]any{
			workDir: map[string]any{
				"hasTrustDialogAccepted":        true,
				"hasCompletedProjectOnboarding": true,
			},
		},
	}
	data, err = json.MarshalIndent(trust, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal trust config: %w", err)
	}
	return os.WriteFile(filepath.Join(claudeDir, ".claude.json"), data, 0600)
}

func expandTildePaths(data []byte) []byte {
	home, err := os.UserHomeDir()
	if err != nil {
		return data
	}
	return []byte(strings.ReplaceAll(string(data), "~/", home+"/"))
}

func writeCodexConfigs(m domain.SprintMemberSnapshot, memberHome, workDir string) error {
	codexDir := filepath.Join(memberHome, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return fmt.Errorf("create .codex dir: %w", err)
	}

	config := make(map[string]any, len(m.DotConfig))
	maps.Copy(config, m.DotConfig)
	config["projects"] = map[string]any{
		workDir: map[string]any{
			"trust_level": "trusted",
		},
	}

	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config.toml: %w", err)
	}
	return os.WriteFile(filepath.Join(codexDir, "config.toml"), data, 0644)
}
