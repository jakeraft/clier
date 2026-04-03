package domain

import "testing"

func TestExportFromSnapshot(t *testing.T) {
	snap := TeamSnapshot{
		TeamID:       "team-001",
		TeamName:     "my-team",
		RootMemberID: "id-alice",
		Members: []TeamMemberSnapshot{
			{
				MemberID:       "id-alice",
				MemberName:     "alice",
				Binary:         BinaryClaude,
				Model:          "claude-sonnet-4-6",
				CliProfileID:   "prof-alice",
				CliProfileName: "claude-sonnet",
				SystemArgs:     []string{"--dangerously-skip-permissions"},
				CustomArgs:     []string{"--verbose"},
				DotConfig:      DotConfig{"key": "value"},
				SystemPrompts: []PromptSnapshot{
					{ID: "sp-1", Name: "default", Prompt: "you are helpful"},
				},
				Envs: []EnvSnapshot{
					{ID: "env-1", Name: "greeting", Key: "GREETING", Value: "hello"},
				},
				GitRepo: &GitRepoSnapshot{
					ID:   "repo-1",
					Name: "my-repo",
					URL:  "https://github.com/example/repo",
				},
				Relations: MemberRelations{
					Leaders: []string{},
					Workers: []string{"id-bob"},
					Peers:   []string{},
				},
			},
			{
				MemberID:       "id-bob",
				MemberName:     "bob",
				Binary:         BinaryClaude,
				Model:          "claude-haiku-4-5-20251001",
				CliProfileID:   "prof-bob",
				CliProfileName: "claude-haiku",
				SystemArgs:     []string{"--dangerously-skip-permissions"},
				CustomArgs:     []string{},
				DotConfig:      DotConfig{},
				SystemPrompts:  []PromptSnapshot{},
				GitRepo:        nil,
				Relations: MemberRelations{
					Leaders: []string{"id-alice"},
					Workers: []string{},
					Peers:   []string{},
				},
			},
		},
	}

	export, err := ExportFromSnapshot(snap)
	if err != nil {
		t.Fatalf("ExportFromSnapshot() returned unexpected error: %v", err)
	}

	// TeamID
	if export.TeamID != "team-001" {
		t.Errorf("TeamID = %q, want %q", export.TeamID, "team-001")
	}

	// TeamName
	if export.TeamName != "my-team" {
		t.Errorf("TeamName = %q, want %q", export.TeamName, "my-team")
	}

	// RootMemberName
	if export.RootMemberName != "alice" {
		t.Errorf("RootMemberName = %q, want %q", export.RootMemberName, "alice")
	}

	// Members count
	if len(export.Members) != 2 {
		t.Fatalf("Members count = %d, want 2", len(export.Members))
	}

	// Alice member
	alice := export.Members[0]
	if alice.ID != "id-alice" {
		t.Errorf("Members[0].ID = %q, want %q", alice.ID, "id-alice")
	}
	if alice.Name != "alice" {
		t.Errorf("Members[0].Name = %q, want %q", alice.Name, "alice")
	}
	if alice.CliProfile.ID != "prof-alice" {
		t.Errorf("alice.CliProfile.ID = %q, want %q", alice.CliProfile.ID, "prof-alice")
	}
	if alice.CliProfile.Name != "claude-sonnet" {
		t.Errorf("alice.CliProfile.Name = %q, want %q", alice.CliProfile.Name, "claude-sonnet")
	}
	if alice.CliProfile.Model != "claude-sonnet-4-6" {
		t.Errorf("alice.CliProfile.Model = %q, want %q", alice.CliProfile.Model, "claude-sonnet-4-6")
	}
	if alice.CliProfile.Binary != BinaryClaude {
		t.Errorf("alice.CliProfile.Binary = %q, want %q", alice.CliProfile.Binary, BinaryClaude)
	}
	if len(alice.CliProfile.SystemArgs) != 1 || alice.CliProfile.SystemArgs[0] != "--dangerously-skip-permissions" {
		t.Errorf("alice.CliProfile.SystemArgs = %v, unexpected", alice.CliProfile.SystemArgs)
	}
	if len(alice.CliProfile.CustomArgs) != 1 || alice.CliProfile.CustomArgs[0] != "--verbose" {
		t.Errorf("alice.CliProfile.CustomArgs = %v, unexpected", alice.CliProfile.CustomArgs)
	}
	if len(alice.SystemPrompts) != 1 || alice.SystemPrompts[0].Name != "default" {
		t.Errorf("alice.SystemPrompts = %v, unexpected", alice.SystemPrompts)
	}
	if alice.SystemPrompts[0].ID != "sp-1" {
		t.Errorf("alice.SystemPrompts[0].ID = %q, want %q", alice.SystemPrompts[0].ID, "sp-1")
	}
	if len(alice.Envs) != 1 || alice.Envs[0].Name != "greeting" || alice.Envs[0].Key != "GREETING" || alice.Envs[0].Value != "hello" {
		t.Errorf("alice.Envs = %v, unexpected", alice.Envs)
	}
	if alice.Envs[0].ID != "env-1" {
		t.Errorf("alice.Envs[0].ID = %q, want %q", alice.Envs[0].ID, "env-1")
	}
	if alice.GitRepo == nil {
		t.Fatal("alice.GitRepo should not be nil")
	}
	if alice.GitRepo.ID != "repo-1" {
		t.Errorf("alice.GitRepo.ID = %q, want %q", alice.GitRepo.ID, "repo-1")
	}
	if alice.GitRepo.Name != "my-repo" {
		t.Errorf("alice.GitRepo.Name = %q, want %q", alice.GitRepo.Name, "my-repo")
	}
	if alice.GitRepo.URL != "https://github.com/example/repo" {
		t.Errorf("alice.GitRepo.URL = %q, want %q", alice.GitRepo.URL, "https://github.com/example/repo")
	}

	// Bob member
	bob := export.Members[1]
	if bob.Name != "bob" {
		t.Errorf("Members[1].Name = %q, want %q", bob.Name, "bob")
	}
	if bob.CliProfile.Name != "claude-haiku" {
		t.Errorf("bob.CliProfile.Name = %q, want %q", bob.CliProfile.Name, "claude-haiku")
	}
	if bob.GitRepo != nil {
		t.Errorf("bob.GitRepo should be nil, got %v", bob.GitRepo)
	}

	// Relations: exactly 1 leader relation (alice -> bob)
	if len(export.Relations) != 1 {
		t.Fatalf("Relations count = %d, want 1", len(export.Relations))
	}
	rel := export.Relations[0]
	if rel.From != "alice" || rel.To != "bob" || rel.Type != RelationLeader {
		t.Errorf("Relations[0] = %+v, want {From:alice To:bob Type:leader}", rel)
	}
}

func TestExportFromSnapshot_PeerDedup(t *testing.T) {
	snap := TeamSnapshot{
		TeamName:     "peer-team",
		RootMemberID: "id-alice",
		Members: []TeamMemberSnapshot{
			{
				MemberID:       "id-alice",
				MemberName:     "alice",
				Binary:         BinaryClaude,
				Model:          "claude-sonnet-4-6",
				CliProfileName: "claude-sonnet",
				SystemArgs:     []string{},
				CustomArgs:     []string{},
				DotConfig:      DotConfig{},
				SystemPrompts:  []PromptSnapshot{},
				GitRepo:        nil,
				Relations: MemberRelations{
					Leaders: []string{},
					Workers: []string{},
					Peers:   []string{"id-bob"},
				},
			},
			{
				MemberID:       "id-bob",
				MemberName:     "bob",
				Binary:         BinaryClaude,
				Model:          "claude-haiku-4-5-20251001",
				CliProfileName: "claude-haiku",
				SystemArgs:     []string{},
				CustomArgs:     []string{},
				DotConfig:      DotConfig{},
				SystemPrompts:  []PromptSnapshot{},
				GitRepo:        nil,
				Relations: MemberRelations{
					Leaders: []string{},
					Workers: []string{},
					Peers:   []string{"id-alice"},
				},
			},
		},
	}

	export, err := ExportFromSnapshot(snap)
	if err != nil {
		t.Fatalf("ExportFromSnapshot() returned unexpected error: %v", err)
	}

	// Bidirectional peer should produce exactly 1 relation
	if len(export.Relations) != 1 {
		t.Fatalf("Relations count = %d, want 1", len(export.Relations))
	}

	rel := export.Relations[0]
	if rel.From != "alice" || rel.To != "bob" || rel.Type != RelationPeer {
		t.Errorf("Relations[0] = %+v, want {From:alice To:bob Type:peer}", rel)
	}
}

func TestExportFromSnapshot_UnknownRootID(t *testing.T) {
	snap := TeamSnapshot{
		TeamName:     "my-team",
		RootMemberID: "nonexistent",
		Members: []TeamMemberSnapshot{
			{
				MemberID:       "id-alice",
				MemberName:     "alice",
				Binary:         BinaryClaude,
				Model:          "claude-sonnet-4-6",
				CliProfileName: "claude-sonnet",
				SystemArgs:     []string{},
				CustomArgs:     []string{},
				DotConfig:      DotConfig{},
				SystemPrompts:  []PromptSnapshot{},
				GitRepo:        nil,
				Relations: MemberRelations{
					Leaders: []string{},
					Workers: []string{},
					Peers:   []string{},
				},
			},
		},
	}

	_, err := ExportFromSnapshot(snap)
	if err == nil {
		t.Error("ExportFromSnapshot() should return error for unknown root member ID")
	}
}
