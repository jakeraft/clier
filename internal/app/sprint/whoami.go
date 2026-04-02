package sprint

import (
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// MemberRef is a lightweight member reference for context output.
type MemberRef struct {
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name"`
}

// SprintContext describes a member's position within a sprint team.
type SprintContext struct {
	SprintID string      `json:"sprint_id"`
	TeamName string      `json:"team_name"`
	Me       MemberRef   `json:"me"`
	Leaders  []MemberRef `json:"leaders"`
	Workers  []MemberRef `json:"workers"`
	Peers    []MemberRef `json:"peers"`
}

// BuildContext builds a SprintContext for the given member from a team snapshot.
func BuildContext(snapshot domain.TeamSnapshot, sprintID, memberID string) (SprintContext, error) {
	nameOf := make(map[string]string, len(snapshot.Members))
	for _, m := range snapshot.Members {
		nameOf[m.MemberID] = m.MemberName
	}

	// User caller: sees all members as workers.
	if memberID == domain.UserMemberID {
		workers := make([]MemberRef, 0, len(snapshot.Members))
		for _, m := range snapshot.Members {
			workers = append(workers, MemberRef{MemberID: m.MemberID, MemberName: m.MemberName})
		}
		return SprintContext{
			SprintID: sprintID,
			TeamName: snapshot.TeamName,
			Me:       MemberRef{MemberID: domain.UserMemberID, MemberName: "user"},
			Leaders:  []MemberRef{},
			Workers:  workers,
			Peers:    []MemberRef{},
		}, nil
	}

	member, ok := findMember(snapshot.Members, memberID)
	if !ok {
		return SprintContext{}, fmt.Errorf("member %q not found in team %q", memberID, snapshot.TeamName)
	}

	toRefs := func(ids []string) []MemberRef {
		refs := make([]MemberRef, 0, len(ids))
		for _, id := range ids {
			refs = append(refs, MemberRef{MemberID: id, MemberName: nameOf[id]})
		}
		return refs
	}

	return SprintContext{
		SprintID: sprintID,
		TeamName: snapshot.TeamName,
		Me:       MemberRef{MemberID: member.MemberID, MemberName: member.MemberName},
		Leaders:  toRefs(member.Relations.Leaders),
		Workers:  toRefs(member.Relations.Workers),
		Peers:    toRefs(member.Relations.Peers),
	}, nil
}
