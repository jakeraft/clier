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
			m := domain.MemberSnapshot{
				MemberID:   "m1",
				Binary:     domain.BinaryClaude,
				Model:      "claude-sonnet-4-6",
				SystemArgs: []string{"--dangerously-skip-permissions"},
				CustomArgs: []string{"--verbose"},
			}
			cmd, tempFiles, err := BuildCommand(m, "you are a coder", "/work", nil)
			if err != nil {
				t.Fatalf("BuildCommand: %v", err)
			}
			if len(tempFiles) != 0 {
				t.Errorf("claude should have no temp files, got %v", tempFiles)
			}
			if !strings.Contains(cmd, "claude") {
				t.Errorf("command should contain binary: %s", cmd)
			}
			if !strings.Contains(cmd, "--model 'claude-sonnet-4-6'") {
				t.Errorf("command should contain model: %s", cmd)
			}
			if !strings.Contains(cmd, "--session-id 'm1'") {
				t.Errorf("command should contain session-id: %s", cmd)
			}
			if !strings.Contains(cmd, "--dangerously-skip-permissions") {
				t.Errorf("command should contain system args: %s", cmd)
			}
			if !strings.Contains(cmd, "--verbose") {
				t.Errorf("command should contain custom args: %s", cmd)
			}
			if !strings.Contains(cmd, "--append-system-prompt") {
				t.Errorf("command should contain prompt: %s", cmd)
			}
			if !strings.HasPrefix(cmd, "cd ") {
				t.Errorf("command should start with cd: %s", cmd)
			}
		})

		t.Run("WithEnv_PrependsExports", func(t *testing.T) {
			m := domain.MemberSnapshot{
				MemberID: "m1",
				Binary:   domain.BinaryClaude,
				Model:    "claude-sonnet-4-6",
			}
			env := []string{"HOME=/tmp/sprint", "FOO=bar"}
			cmd, _, err := BuildCommand(m, "", "/work", env)
			if err != nil {
				t.Fatalf("BuildCommand: %v", err)
			}
			if !strings.HasPrefix(cmd, "export HOME=") {
				t.Errorf("command should start with env exports: %s", cmd)
			}
			if !strings.Contains(cmd, "export FOO='bar'") {
				t.Errorf("command should contain FOO export: %s", cmd)
			}
			if !strings.Contains(cmd, "cd '/work'") {
				t.Errorf("command should contain cd: %s", cmd)
			}
		})
	})

	t.Run("Codex", func(t *testing.T) {
		t.Run("WithPrompt_WritesInstructionsFile", func(t *testing.T) {
			m := domain.MemberSnapshot{
				MemberID:   "m2",
				Binary:     domain.BinaryCodex,
				Model:      "gpt-5.4",
				SystemArgs: []string{},
				CustomArgs: []string{},
			}
			cmd, tempFiles, err := BuildCommand(m, "you are a coder", "/work", nil)
			if err != nil {
				t.Fatalf("BuildCommand: %v", err)
			}
			if len(tempFiles) != 1 {
				t.Fatalf("codex should have 1 temp file, got %d", len(tempFiles))
			}
			defer os.Remove(tempFiles[0])

			if !strings.Contains(cmd, "model_instructions_file=") {
				t.Errorf("command should contain instructions file: %s", cmd)
			}

			data, err := os.ReadFile(tempFiles[0])
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
	t.Run("WithCustomEnv_IncludesAllVars", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID: "m1",
			Environments: []domain.EnvironmentSnapshot{
				{Key: "API_KEY", Value: "secret"},
			},
		}
		env := BuildEnv(m, "sprint-1", "/home/m1")

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			envMap[parts[0]] = parts[1]
		}

		if envMap["HOME"] != "/home/m1" {
			t.Errorf("HOME = %q, want /home/m1", envMap["HOME"])
		}
		if envMap["CLIER_SPRINT_ID"] != "sprint-1" {
			t.Errorf("CLIER_SPRINT_ID = %q, want sprint-1", envMap["CLIER_SPRINT_ID"])
		}
		if envMap["CLIER_MEMBER_ID"] != "m1" {
			t.Errorf("CLIER_MEMBER_ID = %q, want m1", envMap["CLIER_MEMBER_ID"])
		}
		if envMap["API_KEY"] != "secret" {
			t.Errorf("API_KEY = %q, want secret", envMap["API_KEY"])
		}
	})
}
