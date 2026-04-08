package run

import (
	"context"
	"testing"

	"github.com/google/uuid"
	agentrt "github.com/jakeraft/clier/internal/adapter/runtime"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

type fullStubStore struct {
	stubStore
	members        map[string]domain.Member
	claudeMds      map[string]resource.ClaudeMd
	skills         map[string]resource.Skill
	claudeSettings map[string]resource.ClaudeSettings
	teams          map[string]domain.Team
}

func newFullStubStore() *fullStubStore {
	return &fullStubStore{
		members:        make(map[string]domain.Member),
		claudeMds:      make(map[string]resource.ClaudeMd),
		skills:         make(map[string]resource.Skill),
		claudeSettings: make(map[string]resource.ClaudeSettings),
		teams:          make(map[string]domain.Team),
	}
}

func (s *fullStubStore) GetMember(_ context.Context, id string) (domain.Member, error) {
	m, ok := s.members[id]
	if !ok {
		return domain.Member{}, errNotFound("member", id)
	}
	return m, nil
}

func (s *fullStubStore) GetClaudeMd(_ context.Context, id string) (resource.ClaudeMd, error) {
	cm, ok := s.claudeMds[id]
	if !ok {
		return resource.ClaudeMd{}, errNotFound("claude_md", id)
	}
	return cm, nil
}

func (s *fullStubStore) GetSkill(_ context.Context, id string) (resource.Skill, error) {
	sk, ok := s.skills[id]
	if !ok {
		return resource.Skill{}, errNotFound("skill", id)
	}
	return sk, nil
}

func (s *fullStubStore) GetClaudeSettings(_ context.Context, id string) (resource.ClaudeSettings, error) {
	cs, ok := s.claudeSettings[id]
	if !ok {
		return resource.ClaudeSettings{}, errNotFound("claude_settings", id)
	}
	return cs, nil
}

func (s *fullStubStore) GetTeam(_ context.Context, id string) (domain.Team, error) {
	t, ok := s.teams[id]
	if !ok {
		return domain.Team{}, errNotFound("team", id)
	}
	return t, nil
}

func errNotFound(entity, id string) error {
	return domain.ErrNotFound{Entity: entity, ID: id}
}

func createMinimalTeam(t *testing.T, store *fullStubStore) (domain.Team, string, string) {
	t.Helper()

	claudeMd, _ := resource.NewClaudeMd("test-md", "do things")
	store.claudeMds[claudeMd.ID] = *claudeMd

	claudeSettings, _ := resource.NewClaudeSettings("test-settings", `{"key":"val"}`)
	store.claudeSettings[claudeSettings.ID] = *claudeSettings

	root, _ := domain.NewMember("alice", "claude --dangerously-skip-permissions --model claude-sonnet-4-6",
		claudeMd.ID, nil, claudeSettings.ID, "https://example.com/repo.git")
	store.members[root.ID] = *root

	worker, _ := domain.NewMember("bob", "claude --dangerously-skip-permissions --model claude-sonnet-4-6",
		claudeMd.ID, nil, claudeSettings.ID, "")
	store.members[worker.ID] = *worker

	team, _ := domain.NewTeam("test-team", root.ID, "alice")
	workerTM := domain.TeamMember{ID: uuid.NewString(), MemberID: worker.ID, Name: "bob"}
	team.TeamMembers = append(team.TeamMembers, workerTM)
	rel := domain.Relation{From: team.RootTeamMemberID, To: workerTM.ID}
	team.Relations = append(team.Relations, rel)
	store.teams[team.ID] = *team

	return *team, team.RootTeamMemberID, workerTM.ID
}

func TestResolveTeam(t *testing.T) {
	ctx := context.Background()
	store := newFullStubStore()
	team, rootTMID, workerTMID := createMinimalTeam(t, store)

	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "/tmp/base", nil)

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
	store := newFullStubStore()
	team, rootTMID, workerTMID := createMinimalTeam(t, store)

	runtimes := map[string]AgentRuntime{"claude": &agentrt.ClaudeRuntime{}}
	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "/tmp/base", runtimes)

	resolved, err := svc.resolveTeam(ctx, team)
	if err != nil {
		t.Fatalf("resolveTeam: %v", err)
	}

	plans := buildPlans(resolved, "/tmp/base/workspaces", "test-run", runtimes)

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
