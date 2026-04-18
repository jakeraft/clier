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

func TestSplitResourceID(t *testing.T) {
	t.Parallel()

	owner, name, err := splitResourceID("jakeraft/todo-team")
	if err != nil {
		t.Fatalf("splitResourceID: %v", err)
	}
	if owner != "jakeraft" || name != "todo-team" {
		t.Fatalf("got %q/%q", owner, name)
	}

	if _, _, err := splitResourceID("todo-team"); err == nil {
		t.Fatal("expected missing owner to fail")
	}
}

func TestSplitVersionedResourceID(t *testing.T) {
	t.Parallel()

	owner, name, version, err := splitVersionedResourceID("jakeraft/todo-team@7")
	if err != nil {
		t.Fatalf("splitVersionedResourceID: %v", err)
	}
	if owner != "jakeraft" || name != "todo-team" {
		t.Fatalf("got %q/%q", owner, name)
	}
	if version == nil || *version != 7 {
		t.Fatalf("version = %v, want 7", version)
	}

	owner, name, version, err = splitVersionedResourceID("jakeraft/todo-team")
	if err != nil {
		t.Fatalf("splitVersionedResourceID without version: %v", err)
	}
	if owner != "jakeraft" || name != "todo-team" || version != nil {
		t.Fatalf("got %q/%q@%v, want owner/name with nil version", owner, name, version)
	}

	if _, _, _, err := splitVersionedResourceID("jakeraft/todo-team@0"); err == nil {
		t.Fatal("expected non-positive version to fail")
	}
}

func TestSplitVersionedResourceID_OrgOwnerWithoutVersion(t *testing.T) {
	t.Parallel()

	owner, name, version, err := splitVersionedResourceID("@clier/hello-claude")
	if err != nil {
		t.Fatalf("splitVersionedResourceID org owner without version: %v", err)
	}
	if owner != "@clier" || name != "hello-claude" || version != nil {
		t.Fatalf("got %q/%q@%v, want org owner/name with nil version", owner, name, version)
	}
}

func TestSplitVersionedResourceID_OrgOwnerWithVersion(t *testing.T) {
	t.Parallel()

	owner, name, version, err := splitVersionedResourceID("@clier/hello-claude@7")
	if err != nil {
		t.Fatalf("splitVersionedResourceID org owner with version: %v", err)
	}
	if owner != "@clier" || name != "hello-claude" {
		t.Fatalf("got %q/%q", owner, name)
	}
	if version == nil || *version != 7 {
		t.Fatalf("version = %v, want 7", version)
	}
}

func commandNames(cmds []*cobra.Command) []string {
	names := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		names = append(names, cmd.Name())
	}
	return names
}
