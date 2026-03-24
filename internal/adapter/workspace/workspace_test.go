package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

type stubAuth struct {
	err error
}

func (s *stubAuth) CopyAuthTo(_ domain.CliBinary, _ string) error {
	return s.err
}

func TestPrepare(t *testing.T) {
	t.Run("CreatesSprintAndMemberDirs", func(t *testing.T) {
		baseDir := t.TempDir()
		ws := New(baseDir, &stubAuth{})

		snapshot := domain.TeamSnapshot{
			TeamName:     "team-1",
			RootMemberID: "m1",
			Members: []domain.MemberSnapshot{
				{MemberID: "m1", MemberName: "alice", Binary: domain.BinaryClaude},
				{MemberID: "m2", MemberName: "bob", Binary: domain.BinaryClaude},
			},
		}

		dirs, err := ws.Prepare(context.Background(), "sprint-1", snapshot)
		if err != nil {
			t.Fatalf("Prepare: %v", err)
		}

		if len(dirs) != 2 {
			t.Fatalf("expected 2 member dirs, got %d", len(dirs))
		}

		for _, id := range []string{"m1", "m2"} {
			dir, ok := dirs[id]
			if !ok {
				t.Errorf("missing dir for member %s", id)
				continue
			}
			if _, err := os.Stat(dir.Home); err != nil {
				t.Errorf("member home not created: %v", err)
			}
			if _, err := os.Stat(dir.WorkDir); err != nil {
				t.Errorf("member workdir not created: %v", err)
			}
		}
	})

	t.Run("Failure_CleansUpSprintDir", func(t *testing.T) {
		baseDir := t.TempDir()
		ws := New(baseDir, &stubAuth{err: fmt.Errorf("auth failed")})

		snapshot := domain.TeamSnapshot{
			TeamName:     "team-1",
			RootMemberID: "m1",
			Members: []domain.MemberSnapshot{
				{MemberID: "m1", MemberName: "alice", Binary: domain.BinaryClaude},
			},
		}

		_, err := ws.Prepare(context.Background(), "sprint-1", snapshot)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		sprintDir := filepath.Join(baseDir, "sprint-1")
		if _, err := os.Stat(sprintDir); !os.IsNotExist(err) {
			t.Error("sprint dir should be cleaned up on failure")
		}
	})
}

func TestWriteConfigs(t *testing.T) {
	t.Run("Claude/WritesSettingsAndTrust", func(t *testing.T) {
		home := t.TempDir()
		workDir := filepath.Join(home, "project")

		m := domain.MemberSnapshot{
			Binary:    domain.BinaryClaude,
			DotConfig: domain.DotConfig{"skipDangerousModePermissionPrompt": true},
		}

		if err := writeConfigs(m, home, workDir); err != nil {
			t.Fatalf("writeConfigs: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
		if err != nil {
			t.Fatalf("read settings.json: %v", err)
		}
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err != nil {
			t.Fatalf("parse settings.json: %v", err)
		}
		if settings["skipDangerousModePermissionPrompt"] != true {
			t.Errorf("settings missing skipDangerousModePermissionPrompt")
		}

		data, err = os.ReadFile(filepath.Join(home, ".claude.json"))
		if err != nil {
			t.Fatalf("read .claude.json: %v", err)
		}
		var trust map[string]any
		if err := json.Unmarshal(data, &trust); err != nil {
			t.Fatalf("parse .claude.json: %v", err)
		}
		projects := trust["projects"].(map[string]any)
		if _, ok := projects[workDir]; !ok {
			t.Errorf(".claude.json missing project entry for %s", workDir)
		}
	})

	t.Run("Claude/EmptyDotConfig_StillWritesFiles", func(t *testing.T) {
		home := t.TempDir()
		workDir := filepath.Join(home, "project")

		m := domain.MemberSnapshot{
			Binary:    domain.BinaryClaude,
			DotConfig: domain.DotConfig{},
		}

		if err := writeConfigs(m, home, workDir); err != nil {
			t.Fatalf("writeConfigs: %v", err)
		}

		if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); err != nil {
			t.Errorf("settings.json should always be created: %v", err)
		}
		if _, err := os.Stat(filepath.Join(home, ".claude.json")); err != nil {
			t.Errorf(".claude.json should always be created: %v", err)
		}
	})

	t.Run("Codex/WritesConfigToml", func(t *testing.T) {
		home := t.TempDir()
		workDir := filepath.Join(home, "project")

		m := domain.MemberSnapshot{
			Binary:    domain.BinaryCodex,
			DotConfig: domain.DotConfig{"sandbox_mode": "danger-full-access"},
		}

		if err := writeConfigs(m, home, workDir); err != nil {
			t.Fatalf("writeConfigs: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
		if err != nil {
			t.Fatalf("read config.toml: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "sandbox_mode") {
			t.Errorf("config.toml missing sandbox_mode: %s", content)
		}
		if !strings.Contains(content, "trust_level") {
			t.Errorf("config.toml missing trust_level: %s", content)
		}
	})
}

func TestCleanup(t *testing.T) {
	t.Run("RemovesSprintDir", func(t *testing.T) {
		baseDir := t.TempDir()
		ws := New(baseDir, nil)

		sprintDir := filepath.Join(baseDir, "sprint-1", "member-1")
		if err := os.MkdirAll(sprintDir, 0755); err != nil {
			t.Fatalf("create dir: %v", err)
		}

		if err := ws.Cleanup("sprint-1"); err != nil {
			t.Fatalf("Cleanup: %v", err)
		}

		if _, err := os.Stat(filepath.Join(baseDir, "sprint-1")); !os.IsNotExist(err) {
			t.Error("sprint dir should be removed")
		}
	})
}
