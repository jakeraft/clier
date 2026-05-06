package runner

import "strings"

// shellEscape POSIX-quotes a string for use as a single shell argv item.
// Strings containing only safe characters pass through unmodified — the
// command line stays readable in `clier run view` and tmux scrollback.
func shellEscape(s string) string {
	if s == "" {
		return "''"
	}
	if isSafe(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func isSafe(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9':
			continue
		}
		switch r {
		case '/', '_', '-', '.', ':', '=', '@', ',':
			continue
		}
		return false
	}
	return true
}

// joinCommandLine builds the full single-line invocation the CLI sends via
// tmux send-keys: the verbatim server-supplied `command` followed by each
// `args[]` token after per-item POSIX shell escape (ADR-0002 §9).
//
// `command` is intentionally NOT escaped — it is the team author's literal
// shell expression (which may include vendor flags or env prefixes the
// author wrote) and the server emits it verbatim. `args` items ARE
// escaped so the receiving shell parses each as exactly one argv token,
// no matter what bytes the server packed in.
func joinCommandLine(command string, args []string) string {
	if len(args) == 0 {
		return command
	}
	var b strings.Builder
	b.WriteString(command)
	for _, a := range args {
		b.WriteByte(' ')
		b.WriteString(shellEscape(a))
	}
	return b.String()
}
