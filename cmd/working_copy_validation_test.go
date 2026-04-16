package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func TestValidateWorkingCopy_LeafTeam(t *testing.T) {
	agentName := "reviewer"
	base := t.TempDir()
	fs := filesystem.New()

	// Write a leaf team projection (no children).
	projection := &appworkspace.TeamProjection{
		Name:      agentName,
		AgentType: "claude",
		Command:   "claude",
	}
	if err := appworkspace.WriteTeamProjection(fs, appworkspace.TeamProjectionPath(base), projection); err != nil {
		t.Fatalf("WriteTeamProjection: %v", err)
	}

	// Create required files for the agent.
	agentBase := filepath.Join(base, agentName)
	required := []string{
		filepath.Join(agentBase, "CLAUDE.md"),
		filepath.Join(agentBase, ".clier", "work-log-protocol.md"),
		filepath.Join(agentBase, ".claude", "settings.local.json"),
		filepath.Join(agentBase, ".clier", appworkspace.TeamProtocolFileName(agentName)),
	}
	for _, path := range required {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}

	meta := &appworkspace.Manifest{
		Kind: string(api.KindTeam),
	}
	if err := validateWorkingCopy(base, meta); err != nil {
		t.Fatalf("validateWorkingCopy: %v", err)
	}
}

func TestValidateWorkingCopy_CodexLeafTeam(t *testing.T) {
	agentName := "coder"
	base := t.TempDir()
	fs := filesystem.New()

	// Write a leaf team projection for codex agent.
	projection := &appworkspace.TeamProjection{
		Name:      agentName,
		AgentType: "codex",
		Command:   "codex",
	}
	if err := appworkspace.WriteTeamProjection(fs, appworkspace.TeamProjectionPath(base), projection); err != nil {
		t.Fatalf("WriteTeamProjection: %v", err)
	}

	// Create required files for codex agent.
	agentBase := filepath.Join(base, agentName)
	required := []string{
		filepath.Join(agentBase, "AGENTS.md"),
		filepath.Join(agentBase, ".clier", "work-log-protocol.md"),
		filepath.Join(agentBase, ".clier", appworkspace.TeamProtocolFileName(agentName)),
	}
	for _, path := range required {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}

	meta := &appworkspace.Manifest{
		Kind: string(api.KindTeam),
	}
	if err := validateWorkingCopy(base, meta); err != nil {
		t.Fatalf("validateWorkingCopy (codex): %v", err)
	}
}

func TestValidateWorkingCopy_MissingFileFails(t *testing.T) {
	base := t.TempDir()
	fs := filesystem.New()

	// Write a leaf team projection but do NOT create any agent files.
	projection := &appworkspace.TeamProjection{
		Name:      "reviewer",
		AgentType: "claude",
		Command:   "claude",
	}
	if err := appworkspace.WriteTeamProjection(fs, appworkspace.TeamProjectionPath(base), projection); err != nil {
		t.Fatalf("WriteTeamProjection: %v", err)
	}

	meta := &appworkspace.Manifest{
		Kind: string(api.KindTeam),
	}
	if err := validateWorkingCopy(base, meta); err == nil {
		t.Fatalf("expected validation error for incomplete local clone")
	}
}
