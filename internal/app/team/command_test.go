package team

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
	t.Run("Claude_ReturnsClaudeConfigDir", func(t *testing.T) {
		got := configDirEnv(domain.BinaryClaude)
		want := "CLAUDE_CONFIG_DIR=" + PlaceholderMemberspace + "/.claude"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("Codex_ReturnsCodexHome", func(t *testing.T) {
		got := configDirEnv(domain.BinaryCodex)
		want := "CODEX_HOME=" + PlaceholderMemberspace + "/.codex"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("Unknown_FallsBackToHOME", func(t *testing.T) {
		got := configDirEnv("unknown")
		want := "HOME=" + PlaceholderMemberspace
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestBuildEnv(t *testing.T) {
	t.Run("Claude_WithAuth_IncludesAllEnvVars", func(t *testing.T) {
		authEnvs := []string{"CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude}
		userEnvs := []domain.EnvSnapshot{
			{Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
		}

		env := buildEnv(domain.BinaryClaude, "session-1", "m1", authEnvs, userEnvs)

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

	t.Run("Codex_NoAuth_SystemVarsOnly", func(t *testing.T) {
		env := buildEnv(domain.BinaryCodex, "session-1", "m2", nil, nil)

		envMap := make(map[string]string)
		for _, e := range env {
			k, v, _ := strings.Cut(e, "=")
			envMap[k] = v
		}

		if envMap["CODEX_HOME"] != PlaceholderMemberspace+"/.codex" {
			t.Errorf("CODEX_HOME = %q", envMap["CODEX_HOME"])
		}
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
	t.Run("Claude_AllArgs_IncludesPlaceholders", func(t *testing.T) {
		authEnvs := []string{"CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude}

		cmd, err := buildCommand(
			domain.BinaryClaude, "claude-sonnet-4-6",
			[]string{"--dangerously-skip-permissions"}, []string{"--verbose"},
			"you are a coder", "session-1", "m1",
			authEnvs, nil,
		)
		if err != nil {
			t.Fatalf("buildCommand: %v", err)
		}

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

	t.Run("Codex_WithPrompt_UsesDeveloperInstructions", func(t *testing.T) {
		cmd, err := buildCommand(
			domain.BinaryCodex, "gpt-5.4",
			nil, nil,
			"you are a coder", "session-1", "m2",
			nil, nil,
		)
		if err != nil {
			t.Fatalf("buildCommand: %v", err)
		}

		if !strings.Contains(cmd, "developer_instructions=") {
			t.Errorf("should use developer_instructions: %s", cmd)
		}
		if !strings.Contains(cmd, "cd '"+PlaceholderMemberspace+"/project'") {
			t.Errorf("should cd to memberspace/project: %s", cmd)
		}
	})

	t.Run("UnknownBinary_ReturnsError", func(t *testing.T) {
		_, err := buildCommand("unknown", "m1", nil, nil, "", "session-1", "m1", nil, nil)
		if err == nil {
			t.Error("expected error for unknown binary")
		}
	})

	t.Run("WithUserEnvs_BakedIntoCommand", func(t *testing.T) {
		userEnvs := []domain.EnvSnapshot{
			{Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
			{Key: "SSH_AUTH_SOCK", Value: "/tmp/ssh.sock"},
		}

		cmd, err := buildCommand(
			domain.BinaryClaude, "opus",
			nil, nil, "", "session-1", "m1",
			nil, userEnvs,
		)
		if err != nil {
			t.Fatalf("buildCommand: %v", err)
		}

		if !strings.Contains(cmd, "export GITHUB_TOKEN='ghp_xxx'") {
			t.Errorf("missing GITHUB_TOKEN in:\n%s", cmd)
		}
		if !strings.Contains(cmd, "export SSH_AUTH_SOCK='/tmp/ssh.sock'") {
			t.Errorf("missing SSH_AUTH_SOCK in:\n%s", cmd)
		}
	})
}
