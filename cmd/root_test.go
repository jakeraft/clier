package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestApplyAgentHelp_ScopesStandaloneAgentDescriptions(t *testing.T) {
	root := &cobra.Command{
		Use:   "clier",
		Short: "root short",
		Long:  "root long",
	}
	run := &cobra.Command{
		Use:   "run",
		Short: "run short",
		Long:  "run long",
	}
	member := &cobra.Command{
		Use:   "member",
		Short: "member short",
		Long:  "member long",
	}
	root.AddCommand(run, member)

	applyAgentHelp(root, false)

	if strings.Contains(root.Long, "`clier run tell`") {
		t.Fatalf("standalone agent help should not mention tell:\n%s", root.Long)
	}
	if !strings.Contains(root.Long, "Use `clier run note` to record a work log entry.") {
		t.Fatalf("standalone agent help should mention note:\n%s", root.Long)
	}
	if !strings.Contains(run.Long, "Use `note` to record a work log entry.") {
		t.Fatalf("run long should describe note in standalone scope:\n%s", run.Long)
	}
	if strings.Contains(run.Long, "Use `tell` to send a message") {
		t.Fatalf("run long should not describe tell in standalone scope:\n%s", run.Long)
	}
	if member.Long != "member long" {
		t.Fatalf("non-run command help should remain unchanged, got %q", member.Long)
	}
}

func TestApplyAgentHelp_ScopesTeamAgentDescriptions(t *testing.T) {
	root := &cobra.Command{
		Use:   "clier",
		Short: "root short",
		Long:  "root long",
	}
	run := &cobra.Command{
		Use:   "run",
		Short: "run short",
		Long:  "run long",
	}
	root.AddCommand(run)

	applyAgentHelp(root, true)

	if !strings.Contains(root.Long, "Use `clier run tell` to message another team member.") {
		t.Fatalf("team agent help should mention tell:\n%s", root.Long)
	}
	if !strings.Contains(root.Long, "Use `clier run note` to record a work log entry.") {
		t.Fatalf("team agent help should mention note:\n%s", root.Long)
	}
	if !strings.Contains(run.Long, "Use `tell` to message another team member.") {
		t.Fatalf("run long should describe tell in team scope:\n%s", run.Long)
	}
	if !strings.Contains(run.Long, "Use `note` to record a work log entry.") {
		t.Fatalf("run long should describe note in team scope:\n%s", run.Long)
	}
}

func TestFilterAgentCommands_ScopesStandaloneToNoteOnly(t *testing.T) {
	root := &cobra.Command{Use: "clier"}
	run := &cobra.Command{Use: "run"}
	run.AddCommand(
		&cobra.Command{Use: "tell"},
		&cobra.Command{Use: "note"},
		&cobra.Command{Use: "attach"},
	)
	root.AddCommand(run, &cobra.Command{Use: "member"})

	filterAgentCommands(root, false)

	if len(root.Commands()) != 1 || root.Commands()[0].Name() != "run" {
		t.Fatalf("agent scope should only keep run, got %v", commandNames(root.Commands()))
	}
	if got := commandNames(root.Commands()[0].Commands()); strings.Join(got, ",") != "note" {
		t.Fatalf("standalone agent run commands = %v, want [note]", got)
	}
}

func TestFilterAgentCommands_ScopesTeamToTellAndNote(t *testing.T) {
	root := &cobra.Command{Use: "clier"}
	run := &cobra.Command{Use: "run"}
	run.AddCommand(
		&cobra.Command{Use: "tell"},
		&cobra.Command{Use: "note"},
		&cobra.Command{Use: "attach"},
	)
	root.AddCommand(run, &cobra.Command{Use: "member"})

	filterAgentCommands(root, true)

	if len(root.Commands()) != 1 || root.Commands()[0].Name() != "run" {
		t.Fatalf("agent scope should only keep run, got %v", commandNames(root.Commands()))
	}
	if got := commandNames(root.Commands()[0].Commands()); strings.Join(got, ",") != "note,tell" && strings.Join(got, ",") != "tell,note" {
		t.Fatalf("team agent run commands = %v, want [tell note]", got)
	}
}

func commandNames(cmds []*cobra.Command) []string {
	names := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		names = append(names, cmd.Name())
	}
	return names
}
