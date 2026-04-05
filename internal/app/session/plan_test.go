package session

import (
	"context"
	"testing"

	"github.com/google/uuid"
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

func createMinimalTeam(t *testing.T, ctx context.Context, store *db.Store) (domain.Team, string, string) {
	t.Helper()

	sp, _ := resource.NewSystemPrompt("test-prompt", "do things")
	if err := store.CreateSystemPrompt(ctx, sp); err != nil {
		t.Fatalf("CreateSystemPrompt: %v", err)
	}

	profile, _ := resource.NewCliProfileRaw("test-profile", "claude-sonnet-4-6", resource.BinaryClaude,
		[]string{"--dangerously-skip-permissions"}, []string{}, resource.DotConfig{"key": "val"})
	if err := store.CreateCliProfile(ctx, profile); err != nil {
		t.Fatalf("CreateCliProfile: %v", err)
	}

	repo, _ := resource.NewGitRepo("test-repo", "https://example.com/repo.git")
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
	rel := domain.Relation{From: team.RootTeamMemberID, To: workerTM.ID}
	if err := store.AddTeamRelation(ctx, team.ID, rel); err != nil {
		t.Fatalf("AddTeamRelation: %v", err)
	}

	got, err := store.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	return got, team.RootTeamMemberID, workerTM.ID
}

func TestResolveTeam(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	team, rootTMID, workerTMID := createMinimalTeam(t, ctx, store)

	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "/tmp/base", "/home/user")

	resolved, err := svc.resolveTeam(ctx, team)
	if err != nil {
		t.Fatalf("resolveTeam: %v", err)
	}

	if len(resolved.Members) != 2 {
		t.Fatalf("resolved %d members, want 2", len(resolved.Members))
	}

	byID := make(map[string]domain.ResolvedMember)
	for _, rm := range resolved.Members {
		byID[rm.TeamMemberID] = rm
	}

	root := byID[rootTMID]
	if root.Name != "alice" {
		t.Errorf("root Name = %q, want alice", root.Name)
	}
	if root.Profile.Model == "" {
		t.Error("root Profile.Model is empty")
	}
	if len(root.Prompts) != 1 {
		t.Errorf("root Prompts = %d, want 1", len(root.Prompts))
	}
	if root.Repo == nil {
		t.Error("root Repo should not be nil")
	}
	if len(root.Relations.Workers) != 1 {
		t.Errorf("root Workers = %d, want 1", len(root.Relations.Workers))
	}

	worker := byID[workerTMID]
	if worker.Name != "bob" {
		t.Errorf("worker Name = %q, want bob", worker.Name)
	}
	if worker.Repo != nil {
		t.Error("worker Repo should be nil")
	}
	if len(worker.Relations.Leaders) != 1 {
		t.Errorf("worker Leaders = %d, want 1", len(worker.Relations.Leaders))
	}
}

func TestBuildPlans(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	team, rootTMID, workerTMID := createMinimalTeam(t, ctx, store)

	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "/tmp/base", "/home/user")

	resolved, err := svc.resolveTeam(ctx, team)
	if err != nil {
		t.Fatalf("resolveTeam: %v", err)
	}

	plans, err := buildPlans(resolved, "test-session")
	if err != nil {
		t.Fatalf("buildPlans: %v", err)
	}

	if len(plans) != 2 {
		t.Fatalf("plans = %d members, want 2", len(plans))
	}

	planByTMID := make(map[string]domain.MemberPlan)
	for _, p := range plans {
		planByTMID[p.TeamMemberID] = p
	}

	rootPlan := planByTMID[rootTMID]
	if rootPlan.MemberName != "alice" {
		t.Errorf("root MemberName = %q, want alice", rootPlan.MemberName)
	}
	if rootPlan.Terminal.Command == "" {
		t.Error("root Terminal.Command is empty")
	}
	if rootPlan.Workspace.GitRepo == nil {
		t.Error("root should have git repo")
	}

	workerPlan := planByTMID[workerTMID]
	if workerPlan.MemberName != "bob" {
		t.Errorf("worker MemberName = %q, want bob", workerPlan.MemberName)
	}
	if workerPlan.Workspace.GitRepo != nil {
		t.Error("worker should not have git repo")
	}
}
