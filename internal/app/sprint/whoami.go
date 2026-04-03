package sprint

import (
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// MemberRef is a lightweight member reference for position output.
type MemberRef struct {
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name"`
}

// SprintPosition describes a member's position within a sprint team.
type SprintPosition struct {
	SprintID string      `json:"sprint_id"`
	TeamName string      `json:"team_name"`
	Me       MemberRef   `json:"me"`
	Leaders  []MemberRef `json:"leaders"`
	Workers  []MemberRef `json:"workers"`
	Peers    []MemberRef `json:"peers"`
}

// BuildPosition builds a SprintPosition for the given member from a team snapshot.
func BuildPosition(team domain.TeamSnapshot, sprintID, memberID string) (SprintPosition, error) {
	if memberID == domain.UserMemberID {
		workers := make([]MemberRef, 0, len(team.Members))
		for _, m := range team.Members {
			workers = append(workers, MemberRef{MemberID: m.MemberID, MemberName: m.MemberName})
		}
		return SprintPosition{
			SprintID: sprintID,
			TeamName: team.TeamName,
			Me:       MemberRef{MemberID: domain.UserMemberID, MemberName: "user"},
			Leaders:  []MemberRef{},
			Workers:  workers,
			Peers:    []MemberRef{},
		}, nil
	}

	member, ok := team.FindMember(memberID)
	if !ok {
		return SprintPosition{}, fmt.Errorf("member %q not found in team %q", memberID, team.TeamName)
	}

	toRefs := func(ids []string) []MemberRef {
		refs := make([]MemberRef, 0, len(ids))
		for _, id := range ids {
			refs = append(refs, MemberRef{MemberID: id, MemberName: team.MemberName(id)})
		}
		return refs
	}

	return SprintPosition{
		SprintID: sprintID,
		TeamName: team.TeamName,
		Me:       MemberRef{MemberID: member.MemberID, MemberName: member.MemberName},
		Leaders:  toRefs(member.Relations.Leaders),
		Workers:  toRefs(member.Relations.Workers),
		Peers:    toRefs(member.Relations.Peers),
	}, nil
}
