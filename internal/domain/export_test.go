package domain

import "testing"

func TestExportFromSnapshot(t *testing.T) {
	snap := TeamSnapshot{
		TeamName:     "my-team",
		RootMemberID: "id-alice",
		Members: []MemberSnapshot{
			{
				MemberID:       "id-alice",
				MemberName:     "alice",
				Binary:         BinaryClaude,
				Model:          "claude-sonnet-4-6",
				CliProfileName: "claude-sonnet",
				SystemArgs:     []string{"--dangerously-skip-permissions"},
				CustomArgs:     []string{"--verbose"},
				DotConfig:      DotConfig{"key": "value"},
				SystemPrompts: []PromptSnapshot{
					{Name: "default", Prompt: "you are helpful"},
				},
				Envs: []EnvSnapshot{
					{Name: "greeting", Key: "GREETING", Value: "hello"},
				},
				GitRepo: &GitRepoSnapshot{
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
	if alice.Name != "alice" {
		t.Errorf("Members[0].Name = %q, want %q", alice.Name, "alice")
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
	if len(alice.Envs) != 1 || alice.Envs[0].Name != "greeting" || alice.Envs[0].Key != "GREETING" || alice.Envs[0].Value != "hello" {
		t.Errorf("alice.Envs = %v, unexpected", alice.Envs)
	}
	if alice.GitRepo == nil {
		t.Fatal("alice.GitRepo should not be nil")
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
		Members: []MemberSnapshot{
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

func TestTeamExport_Validate(t *testing.T) {
	validExport := func() TeamExport {
		return TeamExport{
			TeamName:       "my-team",
			RootMemberName: "alice",
			Members: []MemberExport{
				{
					Name: "alice",
					CliProfile: CliProfileExport{
						Name:       "claude-sonnet",
						Model:      "claude-sonnet-4-6",
						Binary:     BinaryClaude,
						SystemArgs: []string{},
						CustomArgs: []string{},
						DotConfig:  DotConfig{},
					},
					SystemPrompts: []PromptSnapshot{},
					GitRepo:       nil,
				},
				{
					Name: "bob",
					CliProfile: CliProfileExport{
						Name:       "claude-haiku",
						Model:      "claude-haiku-4-5-20251001",
						Binary:     BinaryClaude,
						SystemArgs: []string{},
						CustomArgs: []string{},
						DotConfig:  DotConfig{},
					},
					SystemPrompts: []PromptSnapshot{},
					GitRepo:       nil,
				},
			},
			Relations: []RelationExport{
				{From: "alice", To: "bob", Type: RelationLeader},
			},
		}
	}

	t.Run("valid export passes", func(t *testing.T) {
		e := validExport()
		if err := e.Validate(); err != nil {
			t.Errorf("Validate() returned error for valid export: %v", err)
		}
	})

	t.Run("empty team name", func(t *testing.T) {
		e := validExport()
		e.TeamName = ""
		if err := e.Validate(); err == nil {
			t.Error("Validate() should return error for empty team name")
		}
	})

	t.Run("unknown root member", func(t *testing.T) {
		e := validExport()
		e.RootMemberName = "unknown"
		if err := e.Validate(); err == nil {
			t.Error("Validate() should return error for unknown root member")
		}
	})

	t.Run("duplicate member names", func(t *testing.T) {
		e := validExport()
		e.Members[1].Name = "alice"
		if err := e.Validate(); err == nil {
			t.Error("Validate() should return error for duplicate member names")
		}
	})

	t.Run("unknown relation member", func(t *testing.T) {
		e := validExport()
		e.Relations = []RelationExport{
			{From: "alice", To: "unknown", Type: RelationLeader},
		}
		if err := e.Validate(); err == nil {
			t.Error("Validate() should return error for unknown relation member")
		}
	})

	t.Run("invalid relation type", func(t *testing.T) {
		e := validExport()
		e.Relations = []RelationExport{
			{From: "alice", To: "bob", Type: RelationType("invalid")},
		}
		if err := e.Validate(); err == nil {
			t.Error("Validate() should return error for invalid relation type")
		}
	})

	t.Run("EmptyMembers", func(t *testing.T) {
		e := TeamExport{
			TeamName:       "my-team",
			RootMemberName: "alice",
			Members:        []MemberExport{},
			Relations:      []RelationExport{},
		}
		if err := e.Validate(); err == nil {
			t.Error("Validate() should return error for empty members")
		}
	})

	t.Run("SelfReferencingRelation", func(t *testing.T) {
		e := validExport()
		e.Relations = []RelationExport{
			{From: "alice", To: "alice", Type: RelationLeader},
		}
		if err := e.Validate(); err == nil {
			t.Error("Validate() should return error for self-referencing relation")
		}
	})
}

func TestExportFromSnapshot_UnknownRootID(t *testing.T) {
	snap := TeamSnapshot{
		TeamName:     "my-team",
		RootMemberID: "nonexistent",
		Members: []MemberSnapshot{
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
