package cmd

import (
	"testing"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func TestCollectRunnableAgents_WalksNestedChildrenRecursively(t *testing.T) {
	t.Parallel()

	state := &appworkspace.Manifest{
		Owner: "jakeraft",
		Name:  "root-team",
		Teams: []appworkspace.StoredTeamState{
			{
				Owner:   "jakeraft",
				Name:    "root-team",
				Version: 1,
				Projection: appworkspace.TeamProjection{
					Name:      "root-team",
					AgentType: "manager",
					Children: []appworkspace.ChildProjection{{
						Owner: "alice",
						Name:  "lead",
					}},
				},
			},
			{
				Owner:   "alice",
				Name:    "lead",
				Version: 1,
				Projection: appworkspace.TeamProjection{
					Name:      "lead",
					AgentType: "manager",
					Children: []appworkspace.ChildProjection{{
						Owner: "bob",
						Name:  "coder",
					}},
				},
			},
			{
				Owner:    "bob",
				Name:     "coder",
				Version:  1,
				LocalDir: appworkspace.AgentWorkspaceLocalPath("bob", "coder"),
				Projection: appworkspace.TeamProjection{
					Name:      "coder",
					AgentType: "codex",
					Command:   "codex",
				},
			},
		},
	}

	agents, err := collectRunnableAgents(state)
	if err != nil {
		t.Fatalf("collectRunnableAgents: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("len(agents) = %d, want 1", len(agents))
	}
	if agents[0].ID != "bob/coder" {
		t.Fatalf("agent ID = %q, want %q", agents[0].ID, "bob/coder")
	}
	if agents[0].LocalBase != "bob.coder" {
		t.Fatalf("localBase = %q, want %q", agents[0].LocalBase, "bob.coder")
	}
}
