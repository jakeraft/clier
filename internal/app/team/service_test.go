package team

import (
	"context"
	"testing"

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

// createMinimalTeam creates a team with 2 members and a leader relation.
// Returns (teamID, rootMemberID, workerMemberID).
func createMinimalTeam(t *testing.T, ctx context.Context, store *db.Store) (string, string, string) {
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

	team, _ := domain.NewTeam("test-team", root.ID)
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if err := store.AddTeamMember(ctx, team.ID, worker.ID); err != nil {
		t.Fatalf("AddTeamMember: %v", err)
	}
	rel := domain.Relation{From: root.ID, To: worker.ID, Type: domain.RelationLeader}
	if err := store.AddTeamRelation(ctx, team.ID, rel); err != nil {
		t.Fatalf("AddTeamRelation: %v", err)
	}

	return team.ID, root.ID, worker.ID
}

func TestService_Export(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	teamID, _, _ := createMinimalTeam(t, ctx, store)

	svc := New(store)
	export, err := svc.Export(ctx, teamID)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if export.TeamName != "test-team" {
		t.Errorf("TeamName = %q, want %q", export.TeamName, "test-team")
	}
	if export.RootMemberName != "alice" {
		t.Errorf("RootMemberName = %q, want %q", export.RootMemberName, "alice")
	}
	if len(export.Members) != 2 {
		t.Fatalf("Members = %d, want 2", len(export.Members))
	}

	// alice should have git repo, bob should not
	findMember := func(name string) domain.MemberExport {
		for _, m := range export.Members {
			if m.Name == name {
				return m
			}
		}
		t.Fatalf("member %q not found", name)
		return domain.MemberExport{}
	}
	alice := findMember("alice")
	bob := findMember("bob")

	if alice.GitRepo == nil {
		t.Error("alice.GitRepo should not be nil")
	}
	if bob.GitRepo != nil {
		t.Error("bob.GitRepo should be nil")
	}
	if alice.CliProfile.Model != "claude-sonnet-4-6" {
		t.Errorf("alice.CliProfile.Model = %q, want %q", alice.CliProfile.Model, "claude-sonnet-4-6")
	}

	if len(export.Relations) != 1 {
		t.Fatalf("Relations = %d, want 1", len(export.Relations))
	}
	if export.Relations[0].From != "alice" || export.Relations[0].To != "bob" {
		t.Errorf("Relation = %v, want alice→bob", export.Relations[0])
	}
}

func TestService_Import(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	svc := New(store)

	export := domain.TeamExport{
		TeamName:       "imported-team",
		RootMemberName: "alice",
		Members: []domain.MemberExport{
			{
				Name: "alice",
				CliProfile: domain.CliProfileExport{
					Name:       "sonnet-profile",
					Model:      "claude-sonnet-4-6",
					Binary:     domain.BinaryClaude,
					SystemArgs: []string{"--dangerously-skip-permissions"},
					CustomArgs: []string{},
					DotConfig:  domain.DotConfig{"key": "val"},
				},
				SystemPrompts: []domain.PromptSnapshot{
					{Name: "shared-prompt", Prompt: "do things"},
					{Name: "alice-prompt", Prompt: "lead the team"},
				},
				Envs: []domain.EnvSnapshot{
					{Name: "shared-env", Key: "GREETING", Value: "hello"},
					{Name: "alice-env", Key: "ALICE_KEY", Value: "alice-val"},
				},
				GitRepo: &domain.GitRepoSnapshot{Name: "my-repo", URL: "https://example.com/repo.git"},
			},
			{
				Name: "bob",
				CliProfile: domain.CliProfileExport{
					Name:       "sonnet-profile", // same profile name as alice — should dedup
					Model:      "claude-sonnet-4-6",
					Binary:     domain.BinaryClaude,
					SystemArgs: []string{"--dangerously-skip-permissions"},
					CustomArgs: []string{},
					DotConfig:  domain.DotConfig{"key": "val"},
				},
				SystemPrompts: []domain.PromptSnapshot{
					{Name: "shared-prompt", Prompt: "do things"}, // same as alice — should dedup
				},
				Envs: []domain.EnvSnapshot{
					{Name: "shared-env", Key: "GREETING", Value: "hello"}, // same as alice — should dedup
				},
				GitRepo: nil,
			},
		},
		Relations: []domain.RelationExport{
			{From: "alice", To: "bob", Type: domain.RelationLeader},
		},
	}

	team, err := svc.Import(ctx, export)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if team.Name != "imported-team" {
		t.Errorf("team.Name = %q, want %q", team.Name, "imported-team")
	}
	if len(team.MemberIDs) != 2 {
		t.Errorf("MemberIDs = %d, want 2", len(team.MemberIDs))
	}
	if len(team.Relations) != 1 {
		t.Fatalf("Relations = %d, want 1", len(team.Relations))
	}
	if team.Relations[0].Type != domain.RelationLeader {
		t.Errorf("Relation.Type = %s, want leader", team.Relations[0].Type)
	}

	// Verify round-trip: export the imported team, compare
	reExport, err := svc.Export(ctx, team.ID)
	if err != nil {
		t.Fatalf("re-Export: %v", err)
	}
	if reExport.TeamName != export.TeamName {
		t.Errorf("round-trip TeamName = %q, want %q", reExport.TeamName, export.TeamName)
	}
	if reExport.RootMemberName != export.RootMemberName {
		t.Errorf("round-trip RootMemberName = %q, want %q", reExport.RootMemberName, export.RootMemberName)
	}
	if len(reExport.Members) != len(export.Members) {
		t.Errorf("round-trip Members = %d, want %d", len(reExport.Members), len(export.Members))
	}
	if len(reExport.Relations) != len(export.Relations) {
		t.Errorf("round-trip Relations = %d, want %d", len(reExport.Relations), len(export.Relations))
	}
}

func TestService_Import_ReimportSkipsExisting(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	svc := New(store)

	export := domain.TeamExport{
		TeamName:       "reimport-team",
		RootMemberName: "alice",
		Members: []domain.MemberExport{
			{
				Name: "alice",
				CliProfile: domain.CliProfileExport{
					Name:       "sonnet-profile",
					Model:      "claude-sonnet-4-6",
					Binary:     domain.BinaryClaude,
					SystemArgs: []string{},
					CustomArgs: []string{},
					DotConfig:  domain.DotConfig{},
				},
				SystemPrompts: []domain.PromptSnapshot{
					{Name: "test-prompt", Prompt: "do things"},
				},
				GitRepo: nil,
			},
		},
		Relations: []domain.RelationExport{},
	}

	// First import: creates everything
	team1, err := svc.Import(ctx, export)
	if err != nil {
		t.Fatalf("first Import: %v", err)
	}

	// Export to get UUIDs
	exported, err := svc.Export(ctx, team1.ID)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Re-import the exported JSON (now with UUIDs)
	team2, err := svc.Import(ctx, exported)
	if err != nil {
		t.Fatalf("re-Import: %v", err)
	}

	// Same team ID — no duplicate created
	if team2.ID != team1.ID {
		t.Errorf("re-import created new team: got %s, want %s", team2.ID, team1.ID)
	}
}

