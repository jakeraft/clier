package task

import (
	"testing"
)

func TestBuildAuthEnvs(t *testing.T) {
	t.Run("ReturnsCommandEnvWithPlaceholder", func(t *testing.T) {
		envs := buildAuthEnvs()

		if len(envs) != 1 {
			t.Fatalf("expected 1 env, got %d", len(envs))
		}
		want := "CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude
		if envs[0] != want {
			t.Errorf("got %q, want %q", envs[0], want)
		}
	})
}
