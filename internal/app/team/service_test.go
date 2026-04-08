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
	claudeMds      map[int64]resource.ClaudeMd
	skills         map[int64]resource.Skill
	claudeSettings map[int64]resource.ClaudeSettings
	members        map[int64]domain.Member
	teams          map[int64]domain.Team
	nextID         int64
}

func newStubStore() *stubStore {
	return &stubStore{
		claudeMds:      make(map[int64]resource.ClaudeMd),
		skills:         make(map[int64]resource.Skill),
		claudeSettings: make(map[int64]resource.ClaudeSettings),
		members:        make(map[int64]domain.Member),
		teams:          make(map[int64]domain.Team),
		nextID:         1,
	}
}

func (s *stubStore) genID() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *stubStore) GetTeam(_ context.Context, id int64) (domain.Team, error) {
	t, ok := s.teams[id]
	if !ok {
		return domain.Team{}, fmt.Errorf("team not found: %d", id)
	}
	return t, nil
}

func (s *stubStore) GetMember(_ context.Context, id int64) (domain.Member, error) {
	m, ok := s.members[id]
	if !ok {
		return domain.Member{}, fmt.Errorf("member not found: %d", id)
	}
	return m, nil
}

func (s *stubStore) CreateClaudeMd(_ context.Context, cm *resource.ClaudeMd) error {
	if cm.ID == 0 {
		cm.ID = s.genID()
	}
	s.claudeMds[cm.ID] = *cm
	return nil
}
func (s *stubStore) CreateSkill(_ context.Context, sk *resource.Skill) error {
	if sk.ID == 0 {
		sk.ID = s.genID()
	}
	s.skills[sk.ID] = *sk
	return nil
}
func (s *stubStore) CreateClaudeSettings(_ context.Context, st *resource.ClaudeSettings) error {
	if st.ID == 0 {
		st.ID = s.genID()
	}
	s.claudeSettings[st.ID] = *st
	return nil
}
func (s *stubStore) CreateMember(_ context.Context, m *domain.Member) error {
	if m.ID == 0 {
		m.ID = s.genID()
	}
	s.members[m.ID] = *m
	return nil
}
func (s *stubStore) CreateTeam(_ context.Context, t *domain.Team) error {
	if t.ID == 0 {
		t.ID = s.genID()
	}
	// Assign IDs to team members that don't have one (simulates server behavior).
	for i := range t.TeamMembers {
		if t.TeamMembers[i].ID == 0 {
			t.TeamMembers[i].ID = s.genID()
		}
	}
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
func (s *stubStore) AddTeamMember(_ context.Context, teamID int64, tm domain.TeamMember) error {
	t, ok := s.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %d", teamID)
	}
	t.TeamMembers = append(t.TeamMembers, tm)
	s.teams[teamID] = t
	return nil
}
func (s *stubStore) AddTeamRelation(_ context.Context, teamID int64, r domain.Relation) error {
	t, ok := s.teams[teamID]
	if !ok {
		return fmt.Errorf("team not found: %d", teamID)
	}
	t.Relations = append(t.Relations, r)
	s.teams[teamID] = t
	return nil
}
func (s *stubStore) ReplaceTeamComposition(_ context.Context, t *domain.Team) error {
	s.teams[t.ID] = *t
	return nil
}

func createMinimalTeam(t *testing.T, ctx context.Context, store *stubStore) (int64, int64, int64) {
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
		&claudeMd.ID, nil, &settings.ID, "https://example.com/repo.git")
	if err := store.CreateMember(ctx, root); err != nil {
		t.Fatalf("CreateMember root: %v", err)
	}

	worker, _ := domain.NewMember("bob", "claude --dangerously-skip-permissions --model claude-sonnet-4-6",
		&claudeMd.ID, nil, &settings.ID, "")
	if err := store.CreateMember(ctx, worker); err != nil {
		t.Fatalf("CreateMember worker: %v", err)
	}

	team, _ := domain.NewTeam("test-team", root.ID, "alice")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	// team.TeamMembers[0] now has a server-assigned ID from CreateTeam.
	rootTMID := team.TeamMembers[0].ID

	workerTMID := store.genID()
	workerTM := domain.TeamMember{ID: workerTMID, MemberID: worker.ID, Name: "bob"}
	if err := store.AddTeamMember(ctx, team.ID, workerTM); err != nil {
		t.Fatalf("AddTeamMember: %v", err)
	}

	rel := domain.Relation{FromTeamMemberID: rootTMID, ToTeamMemberID: workerTM.ID}
	if err := store.AddTeamRelation(ctx, team.ID, rel); err != nil {
		t.Fatalf("AddTeamRelation: %v", err)
	}

	// Update root team member ID on the stored team.
	stored := store.teams[team.ID]
	stored.RootTeamMemberID = &rootTMID
	store.teams[team.ID] = stored

	return team.ID, rootTMID, workerTM.ID
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
