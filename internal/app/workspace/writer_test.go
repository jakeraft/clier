package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
	"github.com/jakeraft/clier/internal/domain"
)

func TestLocalSettingsContent_CodexReturnsEmpty(t *testing.T) {
	profile, err := domain.ProfileFor("codex")
	if err != nil {
		t.Fatalf("ProfileFor: %v", err)
	}
	content, err := localSettingsContent(profile)
	if err != nil {
		t.Fatalf("localSettingsContent: %v", err)
	}
	if content != "{}" {
		t.Fatalf("codex local settings = %q, want %q", content, "{}")
	}
}

func TestLocalSettingsContent_UsesHomeClaudePath(t *testing.T) {
	homeDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", homeDir); err != nil {
		t.Fatalf("Setenv HOME: %v", err)
	}
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	profile, profileErr := domain.ProfileFor("claude")
	if profileErr != nil {
		t.Fatalf("ProfileFor: %v", profileErr)
	}
	content, err := localSettingsContent(profile)
	if err != nil {
		t.Fatalf("localSettingsContent: %v", err)
	}

	var payload struct {
		Excludes []string `json:"claudeMdExcludes"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	want := filepath.ToSlash(filepath.Join(homeDir, ".claude")) + "/**"
	if len(payload.Excludes) != 1 || payload.Excludes[0] != want {
		t.Fatalf("excludes = %v, want [%q]", payload.Excludes, want)
	}
}

func TestMaterializeAgent_WritesSkillsUnderOwnerAndName(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	writer := NewWriter(filesystem.New(), nil, map[string]*api.ResolvedResource{
		"alice/reviewer": {
			OwnerName: "alice",
			Name:      "reviewer",
			Snapshot:  []byte(`{"content":"# reviewer skill"}`),
		},
	})

	err := writer.MaterializeAgent(base, &TeamProjection{
		Name:      "coder",
		AgentType: "claude",
		Skills: []ResourceRefProjection{{
			Owner: "alice",
			Name:  "reviewer",
		}},
	}, "jakeraft/coder")
	if err != nil {
		t.Fatalf("MaterializeAgent: %v", err)
	}

	skillPath := filepath.Join(base, ".claude", "skills", "alice", "reviewer", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", skillPath, err)
	}
	if string(data) != "# reviewer skill" {
		t.Fatalf("skill content = %q", string(data))
	}
}
