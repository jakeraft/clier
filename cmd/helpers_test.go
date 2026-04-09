package cmd

import (
	"os"
	"path/filepath"
	"testing"

	apprun "github.com/jakeraft/clier/internal/app/run"
)

func TestResolveRunPlanPath_SearchesCurrentWorkspaceAncestors(t *testing.T) {
	base := t.TempDir()
	runID := "42"
	plan := &apprun.RunPlan{RunID: runID, Session: "alpha-42"}
	if err := apprun.SavePlan(base, runID, plan); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	projectDir := filepath.Join(base, "member", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
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

func TestResolveRunPlanPath_PrefersCLIER_RUN_PLAN(t *testing.T) {
	base := t.TempDir()
	runID := "99"
	plan := &apprun.RunPlan{RunID: runID, Session: "alpha-99"}
	if err := apprun.SavePlan(base, runID, plan); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	planPath := apprun.PlanPath(base, runID)

	orig := os.Getenv("CLIER_RUN_PLAN")
	if err := os.Setenv("CLIER_RUN_PLAN", planPath); err != nil {
		t.Fatalf("Setenv: %v", err)
	}
	defer func() { _ = os.Setenv("CLIER_RUN_PLAN", orig) }()

	got, err := resolveRunPlanPath(runID)
	if err != nil {
		t.Fatalf("resolveRunPlanPath: %v", err)
	}
	if got != planPath {
		t.Fatalf("plan path = %q, want %q", got, planPath)
	}
}
