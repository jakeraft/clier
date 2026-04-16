package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
