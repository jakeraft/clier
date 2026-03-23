package sprint

import (
	"os"
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
				"export HOME='/home/m1'",
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
		t.Run("WithPrompt_WritesInstructionsInMemberHome", func(t *testing.T) {
			// given: Codex member with temp home dir
			memberHome := t.TempDir()
			m := domain.MemberSnapshot{
				MemberID:   "m2",
				Binary:     domain.BinaryCodex,
				Model:      "gpt-5.4",
				SystemArgs: []string{},
				CustomArgs: []string{},
			}

			// when
			cmd, err := BuildCommand(m, "you are a coder", "/work", "sprint-1", memberHome)
			if err != nil {
				t.Fatalf("BuildCommand: %v", err)
			}

			// then: instructions file is in member home
			instructionsFile := memberHome + "/codex-instructions.md"
			if !strings.Contains(cmd, "model_instructions_file=") {
				t.Errorf("command should contain instructions file: %s", cmd)
			}

			data, err := os.ReadFile(instructionsFile)
			if err != nil {
				t.Fatalf("read instructions file: %v", err)
			}
			if string(data) != "you are a coder" {
				t.Errorf("instructions content = %q, want %q", string(data), "you are a coder")
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
	t.Run("SystemVars_IncludesAllVars", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID: "m1",
		}

		env := buildEnv(m, "sprint-1", "/home/m1")

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			envMap[parts[0]] = parts[1]
		}

		for k, want := range map[string]string{
			"HOME":            "/home/m1",
			"CLIER_SPRINT_ID": "sprint-1",
			"CLIER_MEMBER_ID": "m1",
		} {
			if envMap[k] != want {
				t.Errorf("%s = %q, want %q", k, envMap[k], want)
			}
		}
	})

	t.Run("WithEnvs_AppendsCustomEnvVars", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID: "m1",
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

		// system vars still present
		if envMap["HOME"] != "/home/m1" {
			t.Errorf("HOME = %q, want %q", envMap["HOME"], "/home/m1")
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
			Envs:     nil,
		}

		env := buildEnv(m, "sprint-1", "/home/m1")

		if len(env) != 3 {
			t.Errorf("env length = %d, want 3", len(env))
		}
	})
}
