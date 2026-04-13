package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/filesystem"
	apprun "github.com/jakeraft/clier/internal/app/run"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func TestResolveRunPlanPath_SearchesCurrentWorkspaceAncestors(t *testing.T) {
	base := t.TempDir()
	runID := "42"
	plan := &apprun.RunPlan{RunID: runID, Session: "alpha-42"}
	if err := apprun.SavePlan(base, runID, plan); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if err := appworkspace.SaveManifest(filesystem.New(), base, &appworkspace.Manifest{
		Kind:  string(api.KindMember),
		Owner: "jakeraft",
		Name:  "tech-lead",
	}); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	repoDir := filepath.Join(base, "member")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	got, err := resolveRunPlanPath(runID)
	if err != nil {
		t.Fatalf("resolveRunPlanPath: %v", err)
	}
	want, err := filepath.EvalSymlinks(apprun.PlanPath(base, runID))
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if got != want {
		t.Fatalf("plan path = %q, want %q", got, want)
	}
}

func TestBuildMemberEnv_OmitsTeamIDForStandaloneRuns(t *testing.T) {
	env := buildMemberEnv("run-1", 11, nil, "tech-lead")

	if env["CLIER_TEAM_ID"] != "" {
		t.Fatalf("CLIER_TEAM_ID should be omitted for standalone runs, got %q", env["CLIER_TEAM_ID"])
	}
	if env["CLIER_RUN_ID"] != "run-1" {
		t.Fatalf("CLIER_RUN_ID = %q, want run-1", env["CLIER_RUN_ID"])
	}
	if env["CLIER_MEMBER_ID"] != "11" {
		t.Fatalf("CLIER_MEMBER_ID = %q, want 11", env["CLIER_MEMBER_ID"])
	}
}

func TestBuildMemberEnv_SetsTeamIDForTeamRuns(t *testing.T) {
	teamID := int64(22)
	env := buildMemberEnv("run-1", 11, &teamID, "coder")

	if env["CLIER_TEAM_ID"] != strconv.FormatInt(teamID, 10) {
		t.Fatalf("CLIER_TEAM_ID = %q, want %d", env["CLIER_TEAM_ID"], teamID)
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
