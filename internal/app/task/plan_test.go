package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/adapter/db"
	agentrt "github.com/jakeraft/clier/internal/adapter/runtime"
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

	claudeMd, _ := resource.NewClaudeMd("test-md", "do things")
	if err := store.CreateClaudeMd(ctx, claudeMd); err != nil {
		t.Fatalf("CreateClaudeMd: %v", err)
	}

	claudeSettings, _ := resource.NewClaudeSettings("test-settings", `{"key":"val"}`)
	if err := store.CreateClaudeSettings(ctx, claudeSettings); err != nil {
		t.Fatalf("CreateClaudeSettings: %v", err)
	}

	root, _ := domain.NewMember("alice", "claude --dangerously-skip-permissions --model claude-sonnet-4-6",
		claudeMd.ID, nil, claudeSettings.ID, "https://example.com/repo.git")
	if err := store.CreateMember(ctx, root); err != nil {
		t.Fatalf("CreateMember root: %v", err)
	}

	worker, _ := domain.NewMember("bob", "claude --dangerously-skip-permissions --model claude-sonnet-4-6",
		claudeMd.ID, nil, claudeSettings.ID, "")
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

	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "/tmp/base", "/home/user", nil)

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
	if root.Command == "" {
		t.Error("root Command is empty")
	}
	if root.ClaudeMd == nil {
		t.Error("root ClaudeMd should not be nil")
	}
	if root.GitRepoURL == "" {
		t.Error("root GitRepoURL should not be empty")
	}
	if len(root.Relations.Workers) != 1 {
		t.Errorf("root Workers = %d, want 1", len(root.Relations.Workers))
	}

	worker := byID[workerTMID]
	if worker.Name != "bob" {
		t.Errorf("worker Name = %q, want bob", worker.Name)
	}
	if worker.GitRepoURL != "" {
		t.Error("worker GitRepoURL should be empty")
	}
	if len(worker.Relations.Leaders) != 1 {
		t.Errorf("worker Leaders = %d, want 1", len(worker.Relations.Leaders))
	}
}

func TestBuildPlans(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	team, rootTMID, workerTMID := createMinimalTeam(t, ctx, store)

	runtimes := map[string]AgentRuntime{"claude": &agentrt.ClaudeRuntime{}}
	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "/tmp/base", "/home/user", runtimes)

	resolved, err := svc.resolveTeam(ctx, team)
	if err != nil {
		t.Fatalf("resolveTeam: %v", err)
	}

	plans := buildPlans(resolved, "test-task", runtimes)

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
	if rootPlan.Workspace.GitRepoURL == "" {
		t.Error("root should have git repo URL")
	}

	workerPlan := planByTMID[workerTMID]
	if workerPlan.MemberName != "bob" {
		t.Errorf("worker MemberName = %q, want bob", workerPlan.MemberName)
	}
	if workerPlan.Workspace.GitRepoURL != "" {
		t.Error("worker should not have git repo URL")
	}
}
