package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Relation struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type MemberRelations struct {
	Leaders []string `json:"leaders"`
	Workers []string `json:"workers"`
}

func (r MemberRelations) IsConnectedTo(memberID string) bool {
	return slices.Contains(r.Leaders, memberID) ||
		slices.Contains(r.Workers, memberID)
}

// TeamMember is an instance of a Member spec within a Team.
// The same MemberID (spec reference) can appear in multiple TeamMembers.
type TeamMember struct {
	ID       string `json:"id"`        // unique instance ID (UUID)
	MemberID string `json:"member_id"` // spec reference (Member.ID)
	Name     string `json:"name"`      // display name
}

type Team struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	RootTeamMemberID string       `json:"root_team_member_id"`
	TeamMembers      []TeamMember `json:"team_members"`
	Relations        []Relation   `json:"relations"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

func NewTeam(name, memberID, memberName string) (*Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("team name must not be empty")
	}
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return nil, errors.New("root member id must not be empty")
	}
	memberName = strings.TrimSpace(memberName)
	if memberName == "" {
		return nil, errors.New("root member name must not be empty")
	}

	rootTeamMember := TeamMember{
		ID:       uuid.NewString(),
		MemberID: memberID,
		Name:     memberName,
	}

	now := time.Now()
	return &Team{
		ID:               uuid.NewString(),
		Name:             name,
		RootTeamMemberID: rootTeamMember.ID,
		TeamMembers:      []TeamMember{rootTeamMember},
		Relations:        []Relation{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (t *Team) Update(name *string, rootTeamMemberID *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("team name must not be empty")
		}
		t.Name = trimmed
	}
	if rootTeamMemberID != nil {
		if !t.hasTeamMember(*rootTeamMemberID) {
			return fmt.Errorf("root member must be in team: %s", *rootTeamMemberID)
		}
		t.RootTeamMemberID = *rootTeamMemberID
	}
	t.UpdatedAt = time.Now()
	return nil
}

// ReplaceComposition replaces the team's name, root, members, and relations atomically.
// Validates all invariants: name non-empty, root in members, relation types, no self-relations, etc.
func (t *Team) ReplaceComposition(name string, rootTeamMemberID string, teamMembers []TeamMember, relations []Relation) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("team name must not be empty")
	}
	rootTeamMemberID = strings.TrimSpace(rootTeamMemberID)
	if rootTeamMemberID == "" {
		return errors.New("root team member id must not be empty")
	}

	// Build a temporary team to leverage existing validation.
	tmp := &Team{
		ID:               t.ID,
		Name:             name,
		RootTeamMemberID: rootTeamMemberID,
		TeamMembers:      []TeamMember{},
		Relations:        []Relation{},
	}

	// Add all team members with validation.
	for _, tm := range teamMembers {
		if strings.TrimSpace(tm.ID) == "" {
			return errors.New("team member id must not be empty")
		}
		if strings.TrimSpace(tm.Name) == "" {
			return errors.New("team member name must not be empty")
		}
		tmp.TeamMembers = append(tmp.TeamMembers, tm)
	}

	// Verify root is among team members.
	if !tmp.hasTeamMember(rootTeamMemberID) {
		return fmt.Errorf("root team member not found in team members: %s", rootTeamMemberID)
	}

	for _, r := range relations {
		if err := tmp.AddRelation(r); err != nil {
			return err
		}
	}

	t.Name = tmp.Name
	t.RootTeamMemberID = tmp.RootTeamMemberID
	t.TeamMembers = tmp.TeamMembers
	t.Relations = tmp.Relations
	t.UpdatedAt = time.Now()
	return nil
}

// AddTeamMember creates a new TeamMember instance for the given member spec.
// The same memberID can be added multiple times (duplicate spec is allowed).
func (t *Team) AddTeamMember(memberID, name string) (*TeamMember, error) {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return nil, errors.New("member id must not be empty")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("member name must not be empty")
	}

	tm := TeamMember{
		ID:       uuid.NewString(),
		MemberID: memberID,
		Name:     name,
	}
	t.TeamMembers = append(t.TeamMembers, tm)
	t.UpdatedAt = time.Now()
	return &tm, nil
}

// RemoveTeamMember removes a team member by its unique TeamMember.ID.
// Cannot remove the root team member.
// Also removes all relations involving this team member.
func (t *Team) RemoveTeamMember(teamMemberID string) error {
	if t.RootTeamMemberID == teamMemberID {
		return fmt.Errorf("cannot remove root team member: %s", teamMemberID)
	}
	idx := t.teamMemberIndex(teamMemberID)
	if idx == -1 {
		return fmt.Errorf("team member not in team: %s", teamMemberID)
	}
	t.TeamMembers = append(t.TeamMembers[:idx], t.TeamMembers[idx+1:]...)

	// Remove all relations involving this team member.
	filtered := make([]Relation, 0, len(t.Relations))
	for _, r := range t.Relations {
		if r.From != teamMemberID && r.To != teamMemberID {
			filtered = append(filtered, r)
		}
	}
	t.Relations = filtered
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Team) AddRelation(r Relation) error {
	if r.From == r.To {
		return errors.New("cannot create relation to self")
	}
	if !t.hasTeamMember(r.From) {
		return fmt.Errorf("team member not in team: %s", r.From)
	}
	if !t.hasTeamMember(r.To) {
		return fmt.Errorf("team member not in team: %s", r.To)
	}

	// Check duplicate.
	for _, existing := range t.Relations {
		if existing.From == r.From && existing.To == r.To {
			return fmt.Errorf("duplicate relation: %s -> %s", r.From, r.To)
		}
	}

	// Leader uniqueness: each member can have at most one leader.
	for _, existing := range t.Relations {
		if existing.To == r.To {
			return fmt.Errorf("member %s already has a leader", r.To)
		}
		if existing.From == r.To && existing.To == r.From {
			return fmt.Errorf("mutual leader cycle: %s and %s", r.From, r.To)
		}
	}

	t.Relations = append(t.Relations, r)
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Team) RemoveRelation(from, to string) error {
	for i, r := range t.Relations {
		if r.From == from && r.To == to {
			t.Relations = append(t.Relations[:i], t.Relations[i+1:]...)
			t.UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("relation not found")
}

func (t *Team) MemberRelations(teamMemberID string) MemberRelations {
	leaders := []string{}
	workers := []string{}

	for _, r := range t.Relations {
		if r.To == teamMemberID {
			leaders = append(leaders, r.From)
		}
		if r.From == teamMemberID {
			workers = append(workers, r.To)
		}
	}

	return MemberRelations{
		Leaders: leaders,
		Workers: workers,
	}
}

// FindTeamMember looks up a TeamMember by its unique ID.
func (t *Team) FindTeamMember(teamMemberID string) (*TeamMember, bool) {
	for i, tm := range t.TeamMembers {
		if tm.ID == teamMemberID {
			return &t.TeamMembers[i], true
		}
	}
	return nil, false
}

func (t *Team) hasTeamMember(teamMemberID string) bool {
	return t.teamMemberIndex(teamMemberID) != -1
}

func (t *Team) teamMemberIndex(teamMemberID string) int {
	for i, tm := range t.TeamMembers {
		if tm.ID == teamMemberID {
			return i
		}
	}
	return -1
}


