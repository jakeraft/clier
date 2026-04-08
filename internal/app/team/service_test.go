package team

import (
	"context"
	"fmt"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

// stubStore is an in-memory implementation of the team.Store interface.
type stubStore struct {
	claudeMds      map[string]resource.ClaudeMd
	skills         map[string]resource.Skill
	claudeSettings map[string]resource.ClaudeSettings
	members        map[string]domain.Member
	teams          map[string]domain.Team
}

func newStubStore() *stubStore {
	return &stubStore{
		claudeMds:      make(map[string]resource.ClaudeMd),
		skills:         make(map[string]resource.Skill),
		claudeSettings: make(map[string]resource.ClaudeSettings),
		members:        make(map[string]domain.Member),
		teams:          make(map[string]domain.Team),
	}
}

func (s *stubStore) GetTeam(_ context.Context, id string) (domain.Team, error) {
	t, ok := s.teams[id]
	if !ok {
		return domain.Team{}, fmt.Errorf("team not found: %s", id)
	}
	return t, nil
}

func (s *stubStore) GetMember(_ context.Context, id string) (domain.Member, error) {
	m, ok := s.members[id]
	if !ok {
		return domain.Member{}, fmt.Errorf("member not found: %s", id)
	}
	return m, nil
}

func (s *stubStore) CreateClaudeMd(_ context.Context, cm *resource.ClaudeMd) error {
	s.claudeMds[cm.ID] = *cm
	return nil
}
func (s *stubStore) CreateSkill(_ context.Context, sk *resource.Skill) error {
	s.skills[sk.ID] = *sk
	return nil
}
func (s *stubStore) CreateClaudeSettings(_ context.Context, st *resource.ClaudeSettings) error {
	s.claudeSettings[st.ID] = *st
	return nil
}
func (s *stubStore) CreateMember(_ context.Context, m *domain.Member) error {
	s.members[m.ID] = *m
	return nil
}
func (s *stubStore) CreateTeam(_ context.Context, t *domain.Team) error {
	s.teams[t.ID] = *t
	return nil
}
func (s *stubStore) UpdateClaudeMd(_ context.Context, cm *resource.ClaudeMd) error {
	s.claudeMds[cm.ID] = *cm
	return nil
}
func (s *stubStore) UpdateSkill(_ context.Context, sk *resource.Skill) error {
	s.skills[sk.ID] = *sk
	return nil
}
func (s *stubStore) UpdateClaudeSettings(_ context.Context, st *resource.ClaudeSettings) error {
	s.claudeSettings[st.ID] = *st
	return nil
}
func (s *stubStore) UpdateMember(_ context.Context, m *domain.Member) error {
	s.members[m.ID] = *m
	return nil
}
func (s *stubStore) UpdateTeam(_ context.Context, t *domain.Team) error {
	s.teams[t.ID] = *t
	return nil
}
func (s *stubStore) AddTeamMember(_ context.Context, teamID string, tm domain.TeamMember) error {
	t, ok := s.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}
	t.TeamMembers = append(t.TeamMembers, tm)
	s.teams[teamID] = t
	return nil
}
func (s *stubStore) AddTeamRelation(_ context.Context, teamID string, r domain.Relation) error {
	t, ok := s.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %s", teamID)
	}
	t.Relations = append(t.Relations, r)
	s.teams[teamID] = t
	return nil
}
func (s *stubStore) ReplaceTeamComposition(_ context.Context, t *domain.Team) error {
	s.teams[t.ID] = *t
	return nil
}

func createMinimalTeam(t *testing.T, ctx context.Context, store *stubStore) (string, string, string) {
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
	store := newStubStore()
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
