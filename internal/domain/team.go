package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RelationType string

const (
	RelationLeader RelationType = "leader"
	RelationPeer   RelationType = "peer"
)

type Relation struct {
	From string
	To   string
	Type RelationType
}

type MemberRelations struct {
	Leaders []string
	Workers []string
	Peers   []string
}

type Team struct {
	ID           string
	Name         string
	RootMemberID string
	MemberIDs    []string
	Relations    []Relation
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewTeam(name, rootMemberID string) (*Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("team name must not be empty")
	}

	now := time.Now()
	return &Team{
		ID:           uuid.NewString(),
		Name:         name,
		RootMemberID: rootMemberID,
		MemberIDs:    []string{rootMemberID},
		Relations:    []Relation{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (t *Team) Update(name *string, rootMemberID *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return fmt.Errorf("team name must not be empty")
		}
		t.Name = trimmed
	}
	if rootMemberID != nil {
		if !t.hasMember(*rootMemberID) {
			return fmt.Errorf("root member must be in team: %s: %w", *rootMemberID, ErrNotFound)
		}
		t.RootMemberID = *rootMemberID
	}
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Team) AddMember(memberID string) error {
	if t.hasMember(memberID) {
		return fmt.Errorf("member already in team: %s: %w", memberID, ErrConflict)
	}
	t.MemberIDs = append(t.MemberIDs, memberID)
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Team) RemoveMember(memberID string) error {
	if t.RootMemberID == memberID {
		return fmt.Errorf("cannot remove root member: %s", memberID)
	}
	idx := t.memberIndex(memberID)
	if idx == -1 {
		return fmt.Errorf("member not in team: %s: %w", memberID, ErrNotFound)
	}
	t.MemberIDs = append(t.MemberIDs[:idx], t.MemberIDs[idx+1:]...)

	// Remove all relations involving this member.
	filtered := t.Relations[:0]
	for _, r := range t.Relations {
		if r.From != memberID && r.To != memberID {
			filtered = append(filtered, r)
		}
	}
	t.Relations = filtered
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Team) AddRelation(r Relation) error {
	if r.From == r.To {
		return fmt.Errorf("cannot create relation to self")
	}
	if !t.hasMember(r.From) {
		return fmt.Errorf("member not in team: %s: %w", r.From, ErrNotFound)
	}
	if !t.hasMember(r.To) {
		return fmt.Errorf("member not in team: %s: %w", r.To, ErrNotFound)
	}

	// Check duplicate.
	for _, existing := range t.Relations {
		if existing.From == r.From && existing.To == r.To && existing.Type == r.Type {
			return fmt.Errorf("duplicate relation: %s → %s (%s): %w", r.From, r.To, r.Type, ErrConflict)
		}
	}

	// Leader uniqueness: each member can have at most one leader.
	if r.Type == RelationLeader {
		for _, existing := range t.Relations {
			if existing.To == r.To && existing.Type == RelationLeader {
				return fmt.Errorf("member %s already has a leader: %w", r.To, ErrConflict)
			}
		}
	}

	t.Relations = append(t.Relations, r)
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Team) RemoveRelation(from, to string, relType RelationType) error {
	for i, r := range t.Relations {
		if r.From == from && r.To == to && r.Type == relType {
			t.Relations = append(t.Relations[:i], t.Relations[i+1:]...)
			t.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("relation not found: %w", ErrNotFound)
}

func (t *Team) GetMemberRelations(memberID string) MemberRelations {
	var leaders, workers, peers []string

	for _, r := range t.Relations {
		switch r.Type {
		case RelationLeader:
			if r.To == memberID {
				leaders = append(leaders, r.From)
			}
			if r.From == memberID {
				workers = append(workers, r.To)
			}
		case RelationPeer:
			if r.From == memberID {
				peers = append(peers, r.To)
			} else if r.To == memberID {
				peers = append(peers, r.From)
			}
		}
	}

	return MemberRelations{
		Leaders: leaders,
		Workers: workers,
		Peers:   peers,
	}
}

func (t *Team) GetDisconnectedWarnings() []string {
	connected := make(map[string]bool)
	connected[t.RootMemberID] = true
	for _, r := range t.Relations {
		connected[r.From] = true
		connected[r.To] = true
	}

	var warnings []string
	for _, id := range t.MemberIDs {
		if !connected[id] {
			warnings = append(warnings, fmt.Sprintf("Member %s has no relations. Connect with 'team relation add' if needed.", id))
		}
	}
	return warnings
}

func (t *Team) hasMember(memberID string) bool {
	return t.memberIndex(memberID) != -1
}

func (t *Team) memberIndex(memberID string) int {
	for i, id := range t.MemberIDs {
		if id == memberID {
			return i
		}
	}
	return -1
}
