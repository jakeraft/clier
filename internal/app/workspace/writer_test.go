package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestLocalSettingsContent_UsesHomeClaudePath(t *testing.T) {
	homeDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", homeDir); err != nil {
		t.Fatalf("Setenv HOME: %v", err)
	}
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	profile := domain.ProfileFor("claude")
	content, err := localSettingsContent(profile)
	if err != nil {
		t.Fatalf("localSettingsContent: %v", err)
	}

	var payload struct {
		ClaudeMdExcludes []string `json:"claudeMdExcludes"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	want := filepath.ToSlash(filepath.Join(homeDir, ".claude")) + "/**"
	if len(payload.ClaudeMdExcludes) != 1 || payload.ClaudeMdExcludes[0] != want {
		t.Fatalf("claudeMdExcludes = %v, want [%q]", payload.ClaudeMdExcludes, want)
	}
}
