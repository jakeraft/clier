package session

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestSetAuth(t *testing.T) {
	t.Run("Claude_ReturnsCommandEnvWithPlaceholder", func(t *testing.T) {
		result := setAuth(domain.BinaryClaude)

		if len(result.CommandEnvs) != 1 {
			t.Fatalf("expected 1 env, got %d", len(result.CommandEnvs))
		}
		want := "CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude
		if result.CommandEnvs[0] != want {
			t.Errorf("got %q, want %q", result.CommandEnvs[0], want)
		}
		if len(result.Files) != 0 {
			t.Errorf("claude should not produce auth files, got %d", len(result.Files))
		}
	})

	t.Run("Codex_ReturnsAuthFileWithPlaceholder", func(t *testing.T) {
		result := setAuth(domain.BinaryCodex)

		if len(result.Files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(result.Files))
		}
		wantPath := PlaceholderMemberspace + "/.codex/auth.json"
		if result.Files[0].Path != wantPath {
			t.Errorf("path = %q, want %q", result.Files[0].Path, wantPath)
		}
		if result.Files[0].Content != PlaceholderAuthCodex {
			t.Errorf("content = %q, want %q", result.Files[0].Content, PlaceholderAuthCodex)
		}
		if len(result.CommandEnvs) != 0 {
			t.Errorf("codex should not produce command envs, got %d", len(result.CommandEnvs))
		}
	})

	t.Run("UnknownBinary_ReturnsEmpty", func(t *testing.T) {
		result := setAuth("unknown-cli")

		if len(result.CommandEnvs) != 0 {
			t.Errorf("expected no envs, got %d", len(result.CommandEnvs))
		}
		if len(result.Files) != 0 {
			t.Errorf("expected no files, got %d", len(result.Files))
		}
	})
}
