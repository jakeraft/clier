package session

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/adapter/db"
	"github.com/jakeraft/clier/internal/domain"
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
// and a leader relation. Returns the team and both TeamMember IDs.
func createMinimalTeam(t *testing.T, ctx context.Context, store *db.Store) (domain.Team, string, string) {
	t.Helper()

	sp, _ := domain.NewSystemPrompt("test-prompt", "do things")
	if err := store.CreateSystemPrompt(ctx, sp); err != nil {
		t.Fatalf("CreateSystemPrompt: %v", err)
	}

	profile, _ := domain.NewCliProfileRaw("test-profile", "claude-sonnet-4-6", domain.BinaryClaude,
		[]string{"--dangerously-skip-permissions"}, []string{}, domain.DotConfig{"key": "val"})
	if err := store.CreateCliProfile(ctx, profile); err != nil {
		t.Fatalf("CreateCliProfile: %v", err)
	}

	repo, _ := domain.NewGitRepo("test-repo", "https://example.com/repo.git")
	if err := store.CreateGitRepo(ctx, repo); err != nil {
		t.Fatalf("CreateGitRepo: %v", err)
	}

	root, _ := domain.NewMember("alice", profile.ID, []string{sp.ID}, repo.ID, nil)
	if err := store.CreateMember(ctx, root); err != nil {
		t.Fatalf("CreateMember root: %v", err)
	}

	worker, _ := domain.NewMember("bob", profile.ID, []string{sp.ID}, "", nil)
	if err := store.CreateMember(ctx, worker); err != nil {
		t.Fatalf("CreateMember worker: %v", err)
	}

	team, _ := domain.NewTeam("test-team", root.ID, "alice")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	workerTM := domain.TeamMember{ID: uuid.NewString(), MemberID: worker.ID, Name: "bob"}
	if err := store.AddTeamMember(ctx, team.ID, workerTM); err != nil {
		t.Fatalf("AddTeamMember: %v", err)
	}
	rel := domain.Relation{From: team.RootTeamMemberID, To: workerTM.ID, Type: domain.RelationLeader}
	if err := store.AddTeamRelation(ctx, team.ID, rel); err != nil {
		t.Fatalf("AddTeamRelation: %v", err)
	}

	got, err := store.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	return got, team.RootTeamMemberID, workerTM.ID
}

func TestService_BuildPlan(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	team, rootTMID, workerTMID := createMinimalTeam(t, ctx, store)

	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "/tmp/base", "/home/user")

	plan, err := svc.buildPlan(ctx, team)
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}

	if len(plan) != 2 {
		t.Fatalf("plan = %d members, want 2", len(plan))
	}

	// Verify both team members are in the plan.
	planByTMID := make(map[string]domain.MemberPlan)
	for _, p := range plan {
		planByTMID[p.TeamMemberID] = p
	}

	rootPlan, ok := planByTMID[rootTMID]
	if !ok {
		t.Fatal("root team member not found in plan")
	}
	if rootPlan.MemberName != "alice" {
		t.Errorf("root MemberName = %q, want alice", rootPlan.MemberName)
	}
	if rootPlan.Terminal.Command == "" {
		t.Error("root Terminal.Command is empty")
	}
	if rootPlan.Workspace.GitRepo == nil {
		t.Error("root should have git repo")
	}

	workerPlan, ok := planByTMID[workerTMID]
	if !ok {
		t.Fatal("worker team member not found in plan")
	}
	if workerPlan.MemberName != "bob" {
		t.Errorf("worker MemberName = %q, want bob", workerPlan.MemberName)
	}
	if workerPlan.Workspace.GitRepo != nil {
		t.Error("worker should not have git repo")
	}
}
