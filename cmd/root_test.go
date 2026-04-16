package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewAgentRootCmd_StandaloneScope(t *testing.T) {
	root := newAgentRootCmd(false)

	if strings.Contains(root.Long, "`clier run tell`") {
		t.Fatalf("standalone agent help should not mention tell:\n%s", root.Long)
	}
	if !strings.Contains(root.Long, "Use `clier run note` to record a work log entry.") {
		t.Fatalf("standalone agent help should mention note:\n%s", root.Long)
	}
	if got := commandNames(root.Commands()); strings.Join(got, ",") != "run" {
		t.Fatalf("standalone agent commands = %v, want [run]", got)
	}

	run := root.Commands()[0]
	if !strings.Contains(run.Long, "Use `note` to record a work log entry.") {
		t.Fatalf("standalone run help should mention note:\n%s", run.Long)
	}
	if strings.Contains(run.Long, "Use `tell` to message another agent.") {
		t.Fatalf("standalone run help should not mention tell:\n%s", run.Long)
	}
	if got := commandNames(run.Commands()); strings.Join(got, ",") != "note" {
		t.Fatalf("standalone run commands = %v, want [note]", got)
	}
}

func TestNewAgentRootCmd_TeamScope(t *testing.T) {
	root := newAgentRootCmd(true)

	if !strings.Contains(root.Long, "Use `clier run tell` to message another agent.") {
		t.Fatalf("team agent help should mention tell:\n%s", root.Long)
	}
	if !strings.Contains(root.Long, "Use `clier run note` to record a work log entry.") {
		t.Fatalf("team agent help should mention note:\n%s", root.Long)
	}
	if got := commandNames(root.Commands()); strings.Join(got, ",") != "run" {
		t.Fatalf("team agent commands = %v, want [run]", got)
	}

	run := root.Commands()[0]
	if !strings.Contains(run.Long, "Use `tell` to message another agent.") {
		t.Fatalf("team-scoped run help should mention tell:\n%s", run.Long)
	}
	if !strings.Contains(run.Long, "Use `note` to record a work log entry.") {
		t.Fatalf("team-scoped run help should mention note:\n%s", run.Long)
	}
	got := strings.Join(commandNames(run.Commands()), ",")
	if got != "note,tell" && got != "tell,note" {
		t.Fatalf("team-scoped run commands = %v, want [tell note]", commandNames(run.Commands()))
	}
}

func TestParseOwnerName(t *testing.T) {
	t.Parallel()

	owner, name, err := parseOwnerName("jakeraft/todo-team")
	if err != nil {
		t.Fatalf("parseOwnerName: %v", err)
	}
	if owner != "jakeraft" || name != "todo-team" {
		t.Fatalf("got %q/%q", owner, name)
	}

	if _, _, err := parseOwnerName("todo-team"); err == nil {
		t.Fatal("expected missing owner to fail")
	}
}

func commandNames(cmds []*cobra.Command) []string {
	names := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		names = append(names, cmd.Name())
	}
	return names
}
