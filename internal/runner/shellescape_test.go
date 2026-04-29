package runner

import "testing"

func TestShellEscape(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "''"},
		{"plain", "plain"},
		{"with-dash_underscore.dot:colon=eq@at,comma/slash", "with-dash_underscore.dot:colon=eq@at,comma/slash"},
		{"has space", "'has space'"},
		{"has 'apostrophe'", `'has '\''apostrophe'\'''`},
		{"newline\nin", "'newline\nin'"},
		{"multi-line\n# Team Protocol\n", "'multi-line\n# Team Protocol\n'"},
	}
	for _, tc := range cases {
		got := shellEscape(tc.in)
		if got != tc.want {
			t.Errorf("shellEscape(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestJoinCommandLine(t *testing.T) {
	t.Run("no args is verbatim", func(t *testing.T) {
		got := joinCommandLine("CLIER_AGENT= claude --foo", nil)
		want := "CLIER_AGENT= claude --foo"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("single arg with whitespace gets quoted", func(t *testing.T) {
		got := joinCommandLine("claude", []string{"--append-system-prompt", "# Team Protocol\nhello"})
		want := "claude --append-system-prompt '# Team Protocol\nhello'"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("safe arg passes through", func(t *testing.T) {
		got := joinCommandLine("codex", []string{"-c", "developer_instructions=hello"})
		want := "codex -c developer_instructions=hello"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
