package team

import (
	"context"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/db"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

func setupTestStore(t *testing.T) *db.Store {
	t.Helper()
	store, err := db.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// createMinimalTeam creates a team with 2 team members (alice=root, bob=worker)
// and a leader relation. Returns (teamID, rootTeamMemberID, workerTeamMemberID).
func createMinimalTeam(t *testing.T, ctx context.Context, store *db.Store) (string, string, string) {
	t.Helper()

	claudeMd, _ := resource.NewClaudeMd("test-md", "do things")
	if err := store.CreateClaudeMd(ctx, claudeMd); err != nil {
		t.Fatalf("CreateClaudeMd: %v", err)
	}

	settings, _ := resource.NewClaudeSettings("test-settings", `{"key":"val"}`)
	if err := store.CreateClaudeSettings(ctx, settings); err != nil {
		t.Fatalf("CreateClaudeSettings: %v", err)
	}

	root, _ := domain.NewMember("alice", "claude --dangerously-skip-permissions --model claude-sonnet-4-6",
		claudeMd.ID, nil, settings.ID, "https://example.com/repo.git")
	if err := store.CreateMember(ctx, root); err != nil {
		t.Fatalf("CreateMember root: %v", err)
	}

	worker, _ := domain.NewMember("bob", "claude --dangerously-skip-permissions --model claude-sonnet-4-6",
		claudeMd.ID, nil, settings.ID, "")
	if err := store.CreateMember(ctx, worker); err != nil {
		t.Fatalf("CreateMember worker: %v", err)
	}

	team, _ := domain.NewTeam("test-team", root.ID, "alice")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	workerTM := domain.TeamMember{ID: generateID(), MemberID: worker.ID, Name: "bob"}
	if err := store.AddTeamMember(ctx, team.ID, workerTM); err != nil {
		t.Fatalf("AddTeamMember: %v", err)
	}
	rel := domain.Relation{From: team.RootTeamMemberID, To: workerTM.ID}
	if err := store.AddTeamRelation(ctx, team.ID, rel); err != nil {
		t.Fatalf("AddTeamRelation: %v", err)
	}

	return team.ID, team.RootTeamMemberID, workerTM.ID
}

func TestService_ImportTeam(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	teamID, _, _ := createMinimalTeam(t, ctx, store)

	// Fetch the team and re-import it (upsert path).
	svc := New(store)
	team, err := store.GetTeam(ctx, teamID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}

	if err := svc.ImportTeam(ctx, &team); err != nil {
		t.Fatalf("ImportTeam: %v", err)
	}

	got, err := store.GetTeam(ctx, teamID)
	if err != nil {
		t.Fatalf("GetTeam after import: %v", err)
	}
	if got.Name != team.Name {
		t.Errorf("Name = %q, want %q", got.Name, team.Name)
	}
}
