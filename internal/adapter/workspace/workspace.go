package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/app/sprint"
	"github.com/jakeraft/clier/internal/domain"
	toml "github.com/pelletier/go-toml/v2"
)

// AuthCopier copies CLI auth files to a destination home directory.
type AuthCopier interface {
	CheckAuthReady(binary domain.CliBinary) error
	CopyAuthTo(binary domain.CliBinary, destHome string) error
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
func (w *Workspace) Prepare(ctx context.Context, sprintID string, snapshot domain.TeamSnapshot) (map[string]sprint.MemberDir, error) {
	sprintDir := filepath.Join(w.baseDir, sprintID)
	if err := os.MkdirAll(sprintDir, 0755); err != nil {
		return nil, fmt.Errorf("create sprint dir: %w", err)
	}

	// Cleanup entire sprint directory on any failure
	success := false
	defer func() {
		if !success {
			os.RemoveAll(sprintDir)
		}
	}()

	// Preflight: check auth for all required binaries before creating anything.
	checked := make(map[domain.CliBinary]bool)
	for _, m := range snapshot.Members {
		if !checked[m.Binary] {
			checked[m.Binary] = true
			if err := w.auth.CheckAuthReady(m.Binary); err != nil {
				return nil, err
			}
		}
	}

	dirs := make(map[string]sprint.MemberDir, len(snapshot.Members))
	for _, m := range snapshot.Members {
		dir, err := w.prepareMember(ctx, sprintDir, m)
		if err != nil {
			return nil, fmt.Errorf("prepare member %s: %w", m.MemberName, err)
		}
		dirs[m.MemberID] = dir
	}

	success = true
	return dirs, nil
}

// Cleanup removes all workspace files for a sprint.
func (w *Workspace) Cleanup(sprintID string) error {
	return os.RemoveAll(filepath.Join(w.baseDir, sprintID))
}

func (w *Workspace) prepareMember(ctx context.Context, sprintDir string, m domain.MemberSnapshot) (sprint.MemberDir, error) {
	memberHome := filepath.Join(sprintDir, m.MemberID)
	workDir := filepath.Join(memberHome, "project")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		return sprint.MemberDir{}, fmt.Errorf("create member dir: %w", err)
	}

	if err := w.auth.CopyAuthTo(m.Binary, memberHome); err != nil {
		return sprint.MemberDir{}, fmt.Errorf("copy auth: %w", err)
	}

	if err := writeConfigs(m, memberHome, workDir); err != nil {
		return sprint.MemberDir{}, fmt.Errorf("write configs: %w", err)
	}

	if err := w.setupGit(ctx, m, workDir); err != nil {
		return sprint.MemberDir{}, fmt.Errorf("setup git: %w", err)
	}

	return sprint.MemberDir{Home: memberHome, WorkDir: workDir}, nil
}

func (w *Workspace) setupGit(ctx context.Context, m domain.MemberSnapshot, workDir string) error {
	if m.GitRepo == nil {
		return exec.CommandContext(ctx, "git", "init", workDir).Run()
	}
	if err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", m.GitRepo.URL, workDir).Run(); err != nil {
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
	return os.WriteFile(filepath.Join(memberHome, ".claude.json"), data, 0644)
}

func expandTildePaths(data []byte) []byte {
	home, err := os.UserHomeDir()
	if err != nil {
		return data
	}
	return []byte(strings.ReplaceAll(string(data), "~/", home+"/"))
}

func writeCodexConfigs(m domain.MemberSnapshot, memberHome, workDir string) error {
	codexDir := filepath.Join(memberHome, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return fmt.Errorf("create .codex dir: %w", err)
	}

	config := make(map[string]any)
	for k, v := range m.DotConfig {
		config[k] = v
	}
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
