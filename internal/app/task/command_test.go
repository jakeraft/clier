package task

import (
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	t.Run("Empty_ReturnsSingleQuotes", func(t *testing.T) {
		got := shellQuote("")
		if got != "''" {
			t.Errorf("got %q, want %q", got, "''")
		}
	})

	t.Run("Simple_WrapsInSingleQuotes", func(t *testing.T) {
		got := shellQuote("hello")
		if got != "'hello'" {
			t.Errorf("got %q, want %q", got, "'hello'")
		}
	})

	t.Run("WithSingleQuote_EscapesQuote", func(t *testing.T) {
		got := shellQuote("it's")
		want := `'it'\''s'`
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestConfigDirEnv(t *testing.T) {
	t.Run("ReturnsClaudeConfigDir", func(t *testing.T) {
		got := configDirEnv()
		want := "CLAUDE_CONFIG_DIR=" + PlaceholderMemberspace + "/.claude"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestAuthEnvs(t *testing.T) {
	t.Run("ReturnsCommandEnvWithPlaceholder", func(t *testing.T) {
		envs := authEnvs()

		if len(envs) != 1 {
			t.Fatalf("expected 1 env, got %d", len(envs))
		}
		want := "CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude
		if envs[0] != want {
			t.Errorf("got %q, want %q", envs[0], want)
		}
	})
}

func TestBuildEnv(t *testing.T) {
	t.Run("IncludesAllCategories", func(t *testing.T) {
		env := buildEnv("my-team", "reviewer", "task-1", "m1")

		envMap := make(map[string]string)
		for _, e := range env {
			k, v, _ := strings.Cut(e, "=")
			envMap[k] = v
		}

		for k, want := range map[string]string{
			"CLAUDE_CONFIG_DIR":       PlaceholderMemberspace + "/.claude",
			"CLIER_TASK_ID":           "task-1",
			"CLIER_MEMBER_ID":         "m1",
			"CLAUDE_CODE_OAUTH_TOKEN": PlaceholderAuthClaude,
			"GIT_AUTHOR_NAME":         "my-team/reviewer",
			"GIT_AUTHOR_EMAIL":        "noreply@clier.com",
			"GIT_COMMITTER_NAME":      "my-team/reviewer",
			"GIT_COMMITTER_EMAIL":     "noreply@clier.com",
		} {
			if envMap[k] != want {
				t.Errorf("%s = %q, want %q", k, envMap[k], want)
			}
		}
	})

	t.Run("NoUserEnvs_HasSystemAuthIdentity", func(t *testing.T) {
		env := buildEnv("my-team", "coder", "task-1", "m2")

		// system(3) + auth(1) + identity(4) = 8
		if len(env) != 8 {
			t.Errorf("expected 8 env vars, got %d", len(env))
		}
	})
}

func TestIdentityEnvs(t *testing.T) {
	t.Run("DerivedFromTeamAndMemberName", func(t *testing.T) {
		envs := identityEnvs("dev-team", "alice")

		envMap := make(map[string]string)
		for _, e := range envs {
			k, v, _ := strings.Cut(e, "=")
			envMap[k] = v
		}

		if envMap["GIT_AUTHOR_NAME"] != "dev-team/alice" {
			t.Errorf("GIT_AUTHOR_NAME = %q, want %q", envMap["GIT_AUTHOR_NAME"], "dev-team/alice")
		}
		if envMap["GIT_AUTHOR_EMAIL"] != "noreply@clier.com" {
			t.Errorf("GIT_AUTHOR_EMAIL = %q, want %q", envMap["GIT_AUTHOR_EMAIL"], "noreply@clier.com")
		}
		if envMap["GIT_COMMITTER_NAME"] != "dev-team/alice" {
			t.Errorf("GIT_COMMITTER_NAME = %q, want %q", envMap["GIT_COMMITTER_NAME"], "dev-team/alice")
		}
		if envMap["GIT_COMMITTER_EMAIL"] != "noreply@clier.com" {
			t.Errorf("GIT_COMMITTER_EMAIL = %q, want %q", envMap["GIT_COMMITTER_EMAIL"], "noreply@clier.com")
		}
	})
}

func TestBuildEnvCommand(t *testing.T) {
	t.Run("NoEnv_ReturnsCommandOnly", func(t *testing.T) {
		got := buildEnvCommand("claude --model opus", nil)
		if got != "claude --model opus" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("SingleEnv_PrependsExport", func(t *testing.T) {
		got := buildEnvCommand("claude", []string{"HOME=/tmp/run"})
		want := "export HOME='/tmp/run' &&\nclaude"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("MultipleEnv_ChainsExports", func(t *testing.T) {
		got := buildEnvCommand("claude", []string{"HOME=/tmp/run", "FOO=bar"})
		want := "export HOME='/tmp/run' &&\nexport FOO='bar' &&\nclaude"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestBuildCommand(t *testing.T) {
	t.Run("AllArgs_IncludesPlaceholders", func(t *testing.T) {
		cmd := buildCommand("claude-sonnet-4-6",
			[]string{"--dangerously-skip-permissions", "--verbose"},
			PlaceholderMemberspace+"/project",
			"my-team", "coder", "task-1", "m1")

		for _, want := range []string{
			"claude",
			"--model 'claude-sonnet-4-6'",
			"--dangerously-skip-permissions",
			"--verbose",
			"export CLAUDE_CONFIG_DIR='" + PlaceholderMemberspace + "/.claude'",
			"export CLIER_TASK_ID='task-1'",
			"export CLIER_MEMBER_ID='m1'",
			"export CLAUDE_CODE_OAUTH_TOKEN='" + PlaceholderAuthClaude + "'",
			"export GIT_AUTHOR_NAME='my-team/coder'",
			"cd '" + PlaceholderMemberspace + "/project'",
		} {
			if !strings.Contains(cmd, want) {
				t.Errorf("missing %q in:\n%s", want, cmd)
			}
		}

		// --append-system-prompt should NOT be present
		if strings.Contains(cmd, "--append-system-prompt") {
			t.Error("--append-system-prompt should not be in the command")
		}
	})

}
