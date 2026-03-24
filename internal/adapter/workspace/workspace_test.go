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

func (s *stubAuth) CheckAuthReady(_ domain.CliBinary) error {
	return s.err
}

func (s *stubAuth) CopyAuthTo(_ domain.CliBinary, _ string) error {
	return s.err
}

func TestWorkspace(t *testing.T) {
	t.Run("Prepare", func(t *testing.T) {
		t.Run("ValidTeam_CreatesSprintAndMemberDirs", func(t *testing.T) {
			// Given: a team with 2 Claude members (alice, bob)
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

			// When: Prepare is called
			dirs, err := ws.Prepare(context.Background(), "sprint-1", snapshot)
			if err != nil {
				t.Fatalf("Prepare: %v", err)
			}

			// Then: returns 2 member dirs with the following structure:
			//   {baseDir}/sprint-1/
			//   ├── m1/                  ← alice's Home
			//   │   ├── .claude/
			//   │   │   └── settings.json
			//   │   ├── .claude.json
			//   │   └── project/         ← alice's WorkDir
			//   └── m2/                  ← bob's Home
			//       ├── .claude/
			//       │   └── settings.json
			//       ├── .claude.json
			//       └── project/         ← bob's WorkDir
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

		t.Run("AuthFailure_CleansUpSprintDir", func(t *testing.T) {
			// Given: auth that always fails
			baseDir := t.TempDir()
			ws := New(baseDir, &stubAuth{err: fmt.Errorf("auth failed")})

			snapshot := domain.TeamSnapshot{
				TeamName:     "team-1",
				RootMemberID: "m1",
				Members: []domain.MemberSnapshot{
					{MemberID: "m1", MemberName: "alice", Binary: domain.BinaryClaude},
				},
			}

			// When: Prepare fails
			_, err := ws.Prepare(context.Background(), "sprint-1", snapshot)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Then: entire sprint directory is rolled back (nothing left on disk)
			//   {baseDir}/sprint-1/  ← should NOT exist
			sprintDir := filepath.Join(baseDir, "sprint-1")
			if _, err := os.Stat(sprintDir); !os.IsNotExist(err) {
				t.Error("sprint dir should be cleaned up on failure")
			}
		})
	})

	t.Run("Cleanup", func(t *testing.T) {
		t.Run("ExistingDir_RemovesSprintDir", func(t *testing.T) {
			// Given: an existing sprint directory with nested member dirs
			baseDir := t.TempDir()
			ws := New(baseDir, nil)

			//   {baseDir}/sprint-1/
			//   └── member-1/
			sprintDir := filepath.Join(baseDir, "sprint-1", "member-1")
			if err := os.MkdirAll(sprintDir, 0755); err != nil {
				t.Fatalf("create dir: %v", err)
			}

			// When: Cleanup is called
			if err := ws.Cleanup("sprint-1"); err != nil {
				t.Fatalf("Cleanup: %v", err)
			}

			// Then: entire sprint directory is removed
			//   {baseDir}/sprint-1/  ← should NOT exist
			if _, err := os.Stat(filepath.Join(baseDir, "sprint-1")); !os.IsNotExist(err) {
				t.Error("sprint dir should be removed")
			}
		})
	})
}

func TestWriteConfigs(t *testing.T) {
	t.Run("Claude", func(t *testing.T) {
		t.Run("WithDotConfig_WritesSettingsAndTrust", func(t *testing.T) {
			// Given: a Claude member with DotConfig
			home := t.TempDir()
			workDir := filepath.Join(home, "project")

			m := domain.MemberSnapshot{
				Binary:    domain.BinaryClaude,
				DotConfig: domain.DotConfig{"skipDangerousModePermissionPrompt": true},
			}

			// When: writeConfigs is called
			if err := writeConfigs(m, home, workDir); err != nil {
				t.Fatalf("writeConfigs: %v", err)
			}

			// Then: creates the following files:
			//   {home}/
			//   ├── .claude/
			//   │   └── settings.json   ← contains DotConfig values
			//   └── .claude.json        ← trust config for workDir
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

		t.Run("EmptyDotConfig_StillWritesFiles", func(t *testing.T) {
			// Given: a Claude member with empty DotConfig
			home := t.TempDir()
			workDir := filepath.Join(home, "project")

			m := domain.MemberSnapshot{
				Binary:    domain.BinaryClaude,
				DotConfig: domain.DotConfig{},
			}

			// When: writeConfigs is called
			if err := writeConfigs(m, home, workDir); err != nil {
				t.Fatalf("writeConfigs: %v", err)
			}

			// Then: both files are still created (settings.json will be empty object)
			//   {home}/
			//   ├── .claude/
			//   │   └── settings.json   ← {} (empty but present)
			//   └── .claude.json        ← trust config
			if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); err != nil {
				t.Errorf("settings.json should always be created: %v", err)
			}
			if _, err := os.Stat(filepath.Join(home, ".claude.json")); err != nil {
				t.Errorf(".claude.json should always be created: %v", err)
			}
		})
	})

	t.Run("Codex", func(t *testing.T) {
		t.Run("WithDotConfig_WritesConfigToml", func(t *testing.T) {
			// Given: a Codex member with DotConfig
			home := t.TempDir()
			workDir := filepath.Join(home, "project")

			m := domain.MemberSnapshot{
				Binary:    domain.BinaryCodex,
				DotConfig: domain.DotConfig{"sandbox_mode": "danger-full-access"},
			}

			// When: writeConfigs is called
			if err := writeConfigs(m, home, workDir); err != nil {
				t.Fatalf("writeConfigs: %v", err)
			}

			// Then: creates codex config with DotConfig values and trust:
			//   {home}/
			//   └── .codex/
			//       └── config.toml     ← contains sandbox_mode + trust_level
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
	})

	t.Run("Claude/ClaudeMdExcludes_TildeExpandedToAbsolutePath", func(t *testing.T) {
		home := t.TempDir()
		workDir := filepath.Join(home, "project")

		m := domain.MemberSnapshot{
			Binary: domain.BinaryClaude,
			DotConfig: domain.DotConfig{
				"claudeMdExcludes": []string{"~/.claude/**"},
			},
		}

		if err := writeConfigs(m, home, workDir); err != nil {
			t.Fatalf("writeConfigs: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
		if err != nil {
			t.Fatalf("read settings.json: %v", err)
		}

		content := string(data)
		if strings.Contains(content, "~/") {
			t.Errorf("tilde should be expanded in settings.json:\n%s", content)
		}
		realHome, _ := os.UserHomeDir()
		if !strings.Contains(content, realHome) {
			t.Errorf("settings.json should contain real home %s:\n%s", realHome, content)
		}
	})
}

func TestExpandTildePaths(t *testing.T) {
	home, _ := os.UserHomeDir()

	t.Run("ExpandsTilde", func(t *testing.T) {
		input := []byte(`{"paths": ["~/.claude/**"]}`)
		got := string(expandTildePaths(input))
		want := fmt.Sprintf(`{"paths": ["%s/.claude/**"]}`, home)
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("NoTilde_Unchanged", func(t *testing.T) {
		input := []byte(`{"paths": ["/absolute/path"]}`)
		got := string(expandTildePaths(input))
		if got != string(input) {
			t.Errorf("should not change: got %s", got)
		}
	})
}
