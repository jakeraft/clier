package team

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
)

func generateID() string { return uuid.NewString() }

// ImportTeam validates that all MemberIDs referenced in TeamMembers exist,
// then creates or updates the team in the store.
func (s *Service) ImportTeam(ctx context.Context, t *domain.Team) error {
	// Validate that every TeamMember.MemberID exists in the DB.
	for _, tm := range t.TeamMembers {
		if _, err := s.store.GetMember(ctx, tm.MemberID); err != nil {
			return fmt.Errorf("member %q (id=%s) not found: %w", tm.Name, tm.MemberID, err)
		}
	}

	// Try update first; if not found, create.
	if existing, err := s.store.GetTeam(ctx, t.ID); err == nil {
		if err := existing.ReplaceComposition(t.Name, t.RootTeamMemberID, t.TeamMembers, t.Relations); err != nil {
			return fmt.Errorf("validate team composition: %w", err)
		}
		return s.store.ReplaceTeamComposition(ctx, &existing)
	}

	return s.store.CreateTeam(ctx, t)
}
