package session

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
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

func TestBuildEnv(t *testing.T) {
	t.Run("WithAuth_IncludesAllEnvVars", func(t *testing.T) {
		authEnvs := []string{"CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude}
		userEnvs := []domain.Env{
			{Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
		}

		env := buildEnv("session-1", "m1", authEnvs, userEnvs)

		envMap := make(map[string]string)
		for _, e := range env {
			k, v, _ := strings.Cut(e, "=")
			envMap[k] = v
		}

		for k, want := range map[string]string{
			"CLAUDE_CONFIG_DIR":       PlaceholderMemberspace + "/.claude",
			"CLIER_SESSION_ID":        "session-1",
			"CLIER_MEMBER_ID":         "m1",
			"CLAUDE_CODE_OAUTH_TOKEN": PlaceholderAuthClaude,
			"GITHUB_TOKEN":            "ghp_xxx",
		} {
			if envMap[k] != want {
				t.Errorf("%s = %q, want %q", k, envMap[k], want)
			}
		}
	})

	t.Run("NoAuth_SystemVarsOnly", func(t *testing.T) {
		env := buildEnv("session-1", "m2", nil, nil)

		if len(env) != 3 {
			t.Errorf("expected 3 env vars, got %d", len(env))
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
		authEnvs := []string{"CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude}

		profile := domain.CliProfile{
			Model:      "claude-sonnet-4-6",
			SystemArgs: []string{"--dangerously-skip-permissions"},
			CustomArgs: []string{"--verbose"},
		}
		cmd := buildCommand(profile, "you are a coder", "session-1", "m1", authEnvs, nil)

		for _, want := range []string{
			"claude",
			"--model 'claude-sonnet-4-6'",
			"--dangerously-skip-permissions",
			"--verbose",
			"--append-system-prompt",
			"export CLAUDE_CONFIG_DIR='" + PlaceholderMemberspace + "/.claude'",
			"export CLIER_SESSION_ID='session-1'",
			"export CLIER_MEMBER_ID='m1'",
			"export CLAUDE_CODE_OAUTH_TOKEN='" + PlaceholderAuthClaude + "'",
			"cd '" + PlaceholderMemberspace + "/project'",
		} {
			if !strings.Contains(cmd, want) {
				t.Errorf("missing %q in:\n%s", want, cmd)
			}
		}
	})

	t.Run("WithUserEnvs_BakedIntoCommand", func(t *testing.T) {
		userEnvs := []domain.Env{
			{Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
			{Key: "SSH_AUTH_SOCK", Value: "/tmp/ssh.sock"},
		}

		profile := domain.CliProfile{Model: "opus"}
		cmd := buildCommand(profile, "", "session-1", "m1", nil, userEnvs)

		if !strings.Contains(cmd, "export GITHUB_TOKEN='ghp_xxx'") {
			t.Errorf("missing GITHUB_TOKEN in:\n%s", cmd)
		}
		if !strings.Contains(cmd, "export SSH_AUTH_SOCK='/tmp/ssh.sock'") {
			t.Errorf("missing SSH_AUTH_SOCK in:\n%s", cmd)
		}
	})
}
