package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestMember(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			m, err := NewMember("alice", "profile-1", []string{"prompt-1"}, "repo-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(m.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", m.ID)
			}
			if m.Name != "alice" {
				t.Errorf("Name = %q, want %q", m.Name, "alice")
			}
			if m.CliProfileID != "profile-1" {
				t.Errorf("CliProfileID = %q, want %q", m.CliProfileID, "profile-1")
			}
			if len(m.SystemPromptIDs) != 1 || m.SystemPromptIDs[0] != "prompt-1" {
				t.Errorf("SystemPromptIDs = %v, want [prompt-1]", m.SystemPromptIDs)
			}
			if m.GitRepoID != "repo-1" {
				t.Errorf("GitRepoID = %q, want %q", m.GitRepoID, "repo-1")
			}
		})

		t.Run("NoOptionalFields_DefaultsToEmpty", func(t *testing.T) {
			m, err := NewMember("bob", "profile-1", nil, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(m.SystemPromptIDs) != 0 {
				t.Errorf("SystemPromptIDs = %v, want []", m.SystemPromptIDs)
			}
			if m.GitRepoID != "" {
				t.Errorf("GitRepoID = %q, want empty", m.GitRepoID)
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			_, err := NewMember("", "profile-1", nil, "")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyCliProfileID_ReturnsError", func(t *testing.T) {
			_, err := NewMember("name", "  ", nil, "")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("ValidFields_ChangesAllFields", func(t *testing.T) {
			m, _ := NewMember("old", "profile-1", nil, "")
			name := "new"
			profileID := "profile-2"
			prompts := []string{"prompt-1"}
			repoID := "repo-1"
			if err := m.Update(&name, &profileID, &prompts, &repoID); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.Name != "new" {
				t.Errorf("Name = %q, want %q", m.Name, "new")
			}
			if m.CliProfileID != "profile-2" {
				t.Errorf("CliProfileID = %q, want %q", m.CliProfileID, "profile-2")
			}
			if len(m.SystemPromptIDs) != 1 {
				t.Errorf("SystemPromptIDs = %v, want [prompt-1]", m.SystemPromptIDs)
			}
			if m.GitRepoID != "repo-1" {
				t.Errorf("GitRepoID = %q, want %q", m.GitRepoID, "repo-1")
			}
		})

		t.Run("ClearGitRepoID_SetsEmpty", func(t *testing.T) {
			m, _ := NewMember("name", "profile-1", nil, "repo-1")
			empty := ""
			if err := m.Update(nil, nil, nil, &empty); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.GitRepoID != "" {
				t.Errorf("GitRepoID = %q, want empty", m.GitRepoID)
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			m, _ := NewMember("valid", "profile-1", nil, "")
			name := ""
			if err := m.Update(&name, nil, nil, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyCliProfileID_ReturnsError", func(t *testing.T) {
			m, _ := NewMember("valid", "profile-1", nil, "")
			profileID := "  "
			if err := m.Update(nil, &profileID, nil, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
