package cmd

import (
	"strings"
	"testing"
)

func TestBuildAgentEnv_OmitsTeamNameForStandaloneRuns(t *testing.T) {
	env := buildAgentEnv("run-1", "jakeraft/tech-lead", "")

	if env["CLIER_TEAM_NAME"] != "" {
		t.Fatalf("CLIER_TEAM_NAME should be omitted for standalone runs, got %q", env["CLIER_TEAM_NAME"])
	}
	if env["CLIER_RUN_ID"] != "run-1" {
		t.Fatalf("CLIER_RUN_ID = %q, want run-1", env["CLIER_RUN_ID"])
	}
	if env["CLIER_AGENT_NAME"] != "jakeraft/tech-lead" {
		t.Fatalf("CLIER_AGENT_NAME = %q, want jakeraft/tech-lead", env["CLIER_AGENT_NAME"])
	}
}

func TestBuildAgentEnv_SetsTeamNameForTeamRuns(t *testing.T) {
	env := buildAgentEnv("run-1", "jakeraft/coder", "jakeraft/my-team")

	if env["CLIER_TEAM_NAME"] != "jakeraft/my-team" {
		t.Fatalf("CLIER_TEAM_NAME = %q, want jakeraft/my-team", env["CLIER_TEAM_NAME"])
	}
}

func TestBuildFullCommand_QuotesShellSensitiveValues(t *testing.T) {
	command := buildFullCommand(map[string]string{
		"GIT_AUTHOR_NAME": "O'Brien",
	}, "claude --dangerously-skip-permissions", "/tmp/owner's/workspace")

	if !strings.Contains(command, "export GIT_AUTHOR_NAME='O'\"'\"'Brien'") {
		t.Fatalf("expected quoted env value, got %q", command)
	}
	if !strings.Contains(command, "cd '/tmp/owner'\"'\"'s/workspace'") {
		t.Fatalf("expected quoted cwd, got %q", command)
	}
}
