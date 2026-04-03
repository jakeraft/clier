package sprint

import (
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// BuildMemberPrompt generates the full prompt for a member by combining
// all system prompts into a single string.
func BuildMemberPrompt(team domain.TeamSnapshot, memberID string) (string, error) {
	member, ok := team.FindMember(memberID)
	if !ok {
		return "", fmt.Errorf("member %q not found in team %q", memberID, team.TeamName)
	}

	var parts []string
	for _, sp := range member.SystemPrompts {
		parts = append(parts, sp.Prompt)
	}

	return strings.Join(parts, "\n\n"), nil
}
