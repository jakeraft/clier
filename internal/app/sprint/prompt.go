package sprint

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// BuildMemberPrompt generates the full prompt for a member by combining
// all system prompts (including the built-in team protocol) into a single string.
func BuildMemberPrompt(team domain.TeamSnapshot, memberID string) (string, error) {
	member, ok := findMember(team.Members, memberID)
	if !ok {
		return "", fmt.Errorf("member %q not found in team %q", memberID, team.TeamName)
	}

	var parts []string
	for _, sp := range member.SystemPrompts {
		parts = append(parts, sp.Prompt)
	}

	return strings.Join(parts, "\n\n"), nil
}

func findMember(members []domain.MemberSnapshot, memberID string) (domain.MemberSnapshot, bool) {
	for _, m := range members {
		if m.MemberID == memberID {
			return m, true
		}
	}
	return domain.MemberSnapshot{}, false
}
