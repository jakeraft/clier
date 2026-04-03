package domain

import (
	"errors"
	"fmt"
	"slices"
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
	From string       `json:"from"`
	To   string       `json:"to"`
	Type RelationType `json:"type"`
}

type MemberRelations struct {
	Leaders []string `json:"leaders"`
	Workers []string `json:"workers"`
	Peers   []string `json:"peers"`
}

func (r MemberRelations) IsConnectedTo(memberID string) bool {
	return slices.Contains(r.Leaders, memberID) ||
		slices.Contains(r.Workers, memberID) ||
		slices.Contains(r.Peers, memberID)
}

type Team struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	RootMemberID string           `json:"root_member_id"`
	MemberIDs    []string         `json:"member_ids"`
	Relations    []Relation       `json:"relations"`
	Plan         []MemberSessionPlan `json:"plan"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

func NewTeam(name, rootMemberID string) (*Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("team name must not be empty")
	}
	rootMemberID = strings.TrimSpace(rootMemberID)
	if rootMemberID == "" {
		return nil, errors.New("root member id must not be empty")
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
			return errors.New("team name must not be empty")
		}
		t.Name = trimmed
	}
	if rootMemberID != nil {
		if !t.hasMember(*rootMemberID) {
			return fmt.Errorf("root member must be in team: %s", *rootMemberID)
		}
		t.RootMemberID = *rootMemberID
	}
	t.UpdatedAt = time.Now()
	return nil
}

// ReplaceComposition replaces the team's name, root, members, and relations atomically.
// Validates all invariants: name non-empty, root in members, relation types, no self-relations, etc.
func (t *Team) ReplaceComposition(name string, rootMemberID string, memberIDs []string, relations []Relation) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("team name must not be empty")
	}
	rootMemberID = strings.TrimSpace(rootMemberID)
	if rootMemberID == "" {
		return errors.New("root member id must not be empty")
	}

	// Build a temporary team to leverage existing AddMember/AddRelation validation.
	tmp := &Team{
		ID:           t.ID,
		Name:         name,
		RootMemberID: rootMemberID,
		MemberIDs:    []string{rootMemberID},
		Relations:    []Relation{},
	}
	for _, id := range memberIDs {
		if id == rootMemberID {
			continue
		}
		if err := tmp.AddMember(id); err != nil {
			return err
		}
	}
	for _, r := range relations {
		if err := tmp.AddRelation(r); err != nil {
			return err
		}
	}

	t.Name = tmp.Name
	t.RootMemberID = tmp.RootMemberID
	t.MemberIDs = tmp.MemberIDs
	t.Relations = tmp.Relations
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Team) AddMember(memberID string) error {
	if t.hasMember(memberID) {
		return fmt.Errorf("member already in team: %s", memberID)
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
		return fmt.Errorf("member not in team: %s", memberID)
	}
	t.MemberIDs = append(t.MemberIDs[:idx], t.MemberIDs[idx+1:]...)

	// Remove all relations involving this member.
	filtered := make([]Relation, 0, len(t.Relations))
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
	switch r.Type {
	case RelationLeader, RelationPeer:
	default:
		return fmt.Errorf("invalid relation type: %s (must be leader or peer)", r.Type)
	}
	if r.From == r.To {
		return errors.New("cannot create relation to self")
	}
	if !t.hasMember(r.From) {
		return fmt.Errorf("member not in team: %s", r.From)
	}
	if !t.hasMember(r.To) {
		return fmt.Errorf("member not in team: %s", r.To)
	}

	// Check duplicate (peer relations are bidirectional).
	for _, existing := range t.Relations {
		if existing.From == r.From && existing.To == r.To && existing.Type == r.Type {
			return fmt.Errorf("duplicate relation: %s → %s (%s)", r.From, r.To, r.Type)
		}
		if r.Type == RelationPeer && existing.Type == RelationPeer &&
			existing.From == r.To && existing.To == r.From {
			return fmt.Errorf("duplicate relation: %s → %s (%s)", r.From, r.To, r.Type)
		}
	}

	// Leader uniqueness: each member can have at most one leader.
	// Leader cycle: A→B leader and B→A leader is not allowed.
	if r.Type == RelationLeader {
		for _, existing := range t.Relations {
			if existing.To == r.To && existing.Type == RelationLeader {
				return fmt.Errorf("member %s already has a leader", r.To)
			}
			if existing.Type == RelationLeader &&
				existing.From == r.To && existing.To == r.From {
				return fmt.Errorf("mutual leader cycle: %s and %s", r.From, r.To)
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
	return errors.New("relation not found")
}

func (t *Team) MemberRelations(memberID string) MemberRelations {
	leaders := []string{}
	workers := []string{}
	peers := []string{}

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
