package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func TestValidateWorkingCopy_AgentTeam(t *testing.T) {
	agentName := "reviewer"
	agentOwner := "jakeraft"
	agentID := agentOwner + "/" + agentName
	base := t.TempDir()

	// Create required files for the agent.
	agentBase := filepath.Join(base, filepath.FromSlash(appworkspace.AgentWorkspaceLocalPath(agentOwner, agentName)))
	required := []string{
		filepath.Join(agentBase, "CLAUDE.md"),
		filepath.Join(agentBase, ".clier", "work-log-protocol.md"),
		filepath.Join(agentBase, ".clier", appworkspace.TeamProtocolFileName(agentID)),
	}
	for _, path := range required {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}

	meta := &appworkspace.Manifest{
		Kind:  string(api.KindTeam),
		Owner: agentOwner,
		Name:  agentName,
		Teams: []appworkspace.StoredTeamState{{
			Owner:    agentOwner,
			Name:     agentName,
			LocalDir: appworkspace.AgentWorkspaceLocalPath(agentOwner, agentName),
			Projection: appworkspace.TeamProjection{
				Name:      agentName,
				AgentType: "claude",
				Command:   "claude",
			},
		}},
	}
	if err := validateWorkingCopy(base, meta); err != nil {
		t.Fatalf("validateWorkingCopy: %v", err)
	}
}

func TestValidateWorkingCopy_MissingFileFails(t *testing.T) {
	base := t.TempDir()

	meta := &appworkspace.Manifest{
		Kind:  string(api.KindTeam),
		Owner: "jakeraft",
		Name:  "reviewer",
		Teams: []appworkspace.StoredTeamState{{
			Owner:    "jakeraft",
			Name:     "reviewer",
			LocalDir: appworkspace.AgentWorkspaceLocalPath("jakeraft", "reviewer"),
			Projection: appworkspace.TeamProjection{
				Name:      "reviewer",
				AgentType: "claude",
				Command:   "claude",
			},
		}},
	}
	if err := validateWorkingCopy(base, meta); err == nil {
		t.Fatalf("expected validation error for incomplete local clone")
	}
}
