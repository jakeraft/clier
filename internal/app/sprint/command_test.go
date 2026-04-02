package sprint

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

	t.Run("WithSpaces_PreservesSpaces", func(t *testing.T) {
		got := shellQuote("hello world")
		if got != "'hello world'" {
			t.Errorf("got %q, want %q", got, "'hello world'")
		}
	})

	t.Run("WithSingleQuote_EscapesQuote", func(t *testing.T) {
		got := shellQuote("it's")
		want := `'it'\''s'`
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("WithMultipleSingleQuotes_EscapesAll", func(t *testing.T) {
		got := shellQuote("a'b'c")
		want := `'a'\''b'\''c'`
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("WithSpecialChars_NoEscaping", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			in   string
			want string
		}{
			{"DoubleQuote", `say "hi"`, `'say "hi"'`},
			{"Dollar", "$HOME", "'$HOME'"},
			{"Backtick", "`cmd`", "'`cmd`'"},
			{"Backslash", `a\b`, `'a\b'`},
			{"Newline", "a\nb", "'a\nb'"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				got := shellQuote(tc.in)
				if got != tc.want {
					t.Errorf("shellQuote(%q) = %q, want %q", tc.in, got, tc.want)
				}
			})
		}
	})
}

func TestBuildCommand(t *testing.T) {
	t.Run("Claude", func(t *testing.T) {
		t.Run("AllArgs_IncludesModelSessionPromptAndCustom", func(t *testing.T) {
			// given: Claude member with all args
			m := domain.MemberSnapshot{
				MemberID:   "m1",
				Binary:     domain.BinaryClaude,
				Model:      "claude-sonnet-4-6",
				SystemArgs: []string{"--dangerously-skip-permissions"},
				CustomArgs: []string{"--verbose"},
			}

			// when
			cmd, err := BuildCommand(m, "you are a coder", "/work", "sprint-1", "/home/m1")
			if err != nil {
				t.Fatalf("BuildCommand: %v", err)
			}

			// then: command is
			//   export HOME='/home/m1' &&
			//   export CLIER_SPRINT_ID='sprint-1' && export CLIER_MEMBER_ID='m1' &&
			//   cd '/work' && claude '--dangerously-skip-permissions' --model 'claude-sonnet-4-6'
			//     --session-id 'm1' --append-system-prompt '...' '--verbose'
			for _, want := range []string{
				"claude",
				"--model 'claude-sonnet-4-6'",
				"--session-id 'm1'",
				"--dangerously-skip-permissions",
				"--verbose",
				"--append-system-prompt",
				"export CLAUDE_CONFIG_DIR='/home/m1/.claude'",
				"export CLIER_SPRINT_ID='sprint-1'",
				"export CLIER_MEMBER_ID='m1'",
			} {
				if !strings.Contains(cmd, want) {
					t.Errorf("missing %q in:\n%s", want, cmd)
				}
			}
		})

	})

	t.Run("Codex", func(t *testing.T) {
		t.Run("WithPrompt_UsesDeveloperInstructions", func(t *testing.T) {
			// given: Codex member with prompt
			m := domain.MemberSnapshot{
				MemberID:   "m2",
				Binary:     domain.BinaryCodex,
				Model:      "gpt-5.4",
				SystemArgs: []string{},
				CustomArgs: []string{},
			}

			// when
			cmd, err := BuildCommand(m, "you are a coder", "/work", "sprint-1", "/home/m2")
			if err != nil {
				t.Fatalf("BuildCommand: %v", err)
			}

			// then: uses developer_instructions inline, no file written
			if !strings.Contains(cmd, "developer_instructions=") {
				t.Errorf("should use developer_instructions: %s", cmd)
			}
			if strings.Contains(cmd, "model_instructions_file") {
				t.Errorf("should NOT use model_instructions_file: %s", cmd)
			}
		})
	})
}

func TestBuildEnvCommand(t *testing.T) {
	t.Run("NoEnv_ReturnsCommandOnly", func(t *testing.T) {
		got := buildEnvCommand("claude --model opus", nil)
		if got != "claude --model opus" {
			t.Errorf("got %q, want %q", got, "claude --model opus")
		}
	})

	t.Run("SingleEnv_PrependsExport", func(t *testing.T) {
		got := buildEnvCommand("claude", []string{"HOME=/tmp/sprint"})
		want := "export HOME='/tmp/sprint' && claude"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("MultipleEnv_ChainsExports", func(t *testing.T) {
		got := buildEnvCommand("claude", []string{"HOME=/tmp/sprint", "FOO=bar"})
		want := "export HOME='/tmp/sprint' && export FOO='bar' && claude"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("ValueWithSingleQuote_EscapesQuote", func(t *testing.T) {
		got := buildEnvCommand("claude", []string{"MSG=it's fine"})
		want := `export MSG='it'\''s fine' && claude`
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("ValueWithEquals_SplitsOnFirstEquals", func(t *testing.T) {
		got := buildEnvCommand("claude", []string{"OPTS=key=value"})
		want := "export OPTS='key=value' && claude"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("EmptyValue_ExportsEmptyString", func(t *testing.T) {
		got := buildEnvCommand("claude", []string{"EMPTY="})
		want := "export EMPTY='' && claude"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestBuildEnv(t *testing.T) {
	t.Run("Claude_UsesClaudeConfigDir", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID: "m1",
			Binary:   domain.BinaryClaude,
		}

		env := buildEnv(m, "sprint-1", "/home/m1")

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			envMap[parts[0]] = parts[1]
		}

		for k, want := range map[string]string{
			"CLAUDE_CONFIG_DIR": "/home/m1/.claude",
			"CLIER_SPRINT_ID":   "sprint-1",
			"CLIER_MEMBER_ID":   "m1",
		} {
			if envMap[k] != want {
				t.Errorf("%s = %q, want %q", k, envMap[k], want)
			}
		}
		if _, ok := envMap["HOME"]; ok {
			t.Error("HOME should not be set for claude")
		}
	})

	t.Run("Codex_UsesCodexHome", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID: "m1",
			Binary:   domain.BinaryCodex,
		}

		env := buildEnv(m, "sprint-1", "/home/m1")

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			envMap[parts[0]] = parts[1]
		}

		if envMap["CODEX_HOME"] != "/home/m1/.codex" {
			t.Errorf("CODEX_HOME = %q, want %q", envMap["CODEX_HOME"], "/home/m1/.codex")
		}
		if _, ok := envMap["HOME"]; ok {
			t.Error("HOME should not be set for codex")
		}
	})

	t.Run("WithEnvs_AppendsCustomEnvVars", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID: "m1",
			Binary:   domain.BinaryClaude,
			Envs: []domain.EnvSnapshot{
				{Name: "github-token", Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
				{Name: "ssh-sock", Key: "SSH_AUTH_SOCK", Value: "/tmp/ssh.sock"},
			},
		}

		env := buildEnv(m, "sprint-1", "/home/m1")

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			envMap[parts[0]] = parts[1]
		}

		// config dir var present
		if envMap["CLAUDE_CONFIG_DIR"] != "/home/m1/.claude" {
			t.Errorf("CLAUDE_CONFIG_DIR = %q, want %q", envMap["CLAUDE_CONFIG_DIR"], "/home/m1/.claude")
		}

		// custom vars appended
		if envMap["GITHUB_TOKEN"] != "ghp_xxx" {
			t.Errorf("GITHUB_TOKEN = %q, want %q", envMap["GITHUB_TOKEN"], "ghp_xxx")
		}
		if envMap["SSH_AUTH_SOCK"] != "/tmp/ssh.sock" {
			t.Errorf("SSH_AUTH_SOCK = %q, want %q", envMap["SSH_AUTH_SOCK"], "/tmp/ssh.sock")
		}
	})

	t.Run("NoEnvs_OnlySystemVars", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID: "m1",
			Binary:   domain.BinaryClaude,
			Envs:     nil,
		}

		env := buildEnv(m, "sprint-1", "/home/m1")

		if len(env) != 3 {
			t.Errorf("env length = %d, want 3", len(env))
		}
	})
}
