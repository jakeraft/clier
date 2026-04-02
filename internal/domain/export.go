package domain

import (
	"errors"
	"fmt"
)

type TeamExport struct {
	TeamName       string           `json:"team_name"`
	RootMemberName string           `json:"root_member_name"`
	Members        []MemberExport   `json:"members"`
	Relations      []RelationExport `json:"relations"`
}

type MemberExport struct {
	Name          string           `json:"name"`
	CliProfile    CliProfileExport `json:"cli_profile"`
	SystemPrompts []PromptSnapshot `json:"system_prompts"`
	Envs          []EnvSnapshot    `json:"envs"`
	GitRepo       *GitRepoSnapshot `json:"git_repo"`
}

type CliProfileExport struct {
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	Binary     CliBinary `json:"binary"`
	SystemArgs []string  `json:"system_args"`
	CustomArgs []string  `json:"custom_args"`
	DotConfig  DotConfig `json:"dot_config"`
}

type RelationExport struct {
	From string       `json:"from"`
	To   string       `json:"to"`
	Type RelationType `json:"type"`
}

// ExportFromSnapshot converts a TeamSnapshot (UUID-based) into a TeamExport (name-based).
func ExportFromSnapshot(snap TeamSnapshot) (TeamExport, error) {
	// 1. Build idToName map
	idToName := make(map[string]string, len(snap.Members))
	for _, m := range snap.Members {
		idToName[m.MemberID] = m.MemberName
	}

	// 2. Check RootMemberID exists
	rootMemberName, ok := idToName[snap.RootMemberID]
	if !ok {
		return TeamExport{}, fmt.Errorf("root member ID not found in snapshot: %s", snap.RootMemberID)
	}

	// 3. Convert each MemberSnapshot to MemberExport
	members := make([]MemberExport, 0, len(snap.Members))
	for _, m := range snap.Members {
		members = append(members, MemberExport{
			Name: m.MemberName,
			CliProfile: CliProfileExport{
				Name:       m.CliProfileName,
				Model:      m.Model,
				Binary:     m.Binary,
				SystemArgs: m.SystemArgs,
				CustomArgs: m.CustomArgs,
				DotConfig:  m.DotConfig,
			},
			SystemPrompts: m.SystemPrompts,
			Envs:          m.Envs,
			GitRepo:       m.GitRepo,
		})
	}

	// 4. Extract relations from MemberSnapshot.Relations
	var relations []RelationExport
	for _, m := range snap.Members {
		memberName := m.MemberName

		// Workers: members THIS member leads
		for _, workerID := range m.Relations.Workers {
			workerName, exists := idToName[workerID]
			if !exists {
				return TeamExport{}, fmt.Errorf("member ID not found in snapshot: %s", workerID)
			}
			relations = append(relations, RelationExport{
				From: memberName,
				To:   workerName,
				Type: RelationLeader,
			})
		}

		// Peers: deduplicate by only emitting when memberName < peerName
		for _, peerID := range m.Relations.Peers {
			peerName, exists := idToName[peerID]
			if !exists {
				return TeamExport{}, fmt.Errorf("member ID not found in snapshot: %s", peerID)
			}
			if memberName < peerName {
				relations = append(relations, RelationExport{
					From: memberName,
					To:   peerName,
					Type: RelationPeer,
				})
			}
		}
	}

	return TeamExport{
		TeamName:       snap.TeamName,
		RootMemberName: rootMemberName,
		Members:        members,
		Relations:      relations,
	}, nil
}

// Validate checks that the TeamExport is well-formed.
// Returns a descriptive error for the first violation found.
func (e TeamExport) Validate() error {
	if e.TeamName == "" {
		return errors.New("team name must not be empty")
	}
	if e.RootMemberName == "" {
		return errors.New("root member name must not be empty")
	}

	if len(e.Members) == 0 {
		return errors.New("members must not be empty")
	}

	// Build set of member names and check for duplicates
	memberSet := make(map[string]struct{}, len(e.Members))
	for _, m := range e.Members {
		if _, exists := memberSet[m.Name]; exists {
			return fmt.Errorf("duplicate member name: %s", m.Name)
		}
		memberSet[m.Name] = struct{}{}
	}

	// RootMemberName must exist in Members
	if _, exists := memberSet[e.RootMemberName]; !exists {
		return fmt.Errorf("root member %q not found in members", e.RootMemberName)
	}

	// All relation From/To must reference existing members, and Type must be valid
	for _, r := range e.Relations {
		if _, exists := memberSet[r.From]; !exists {
			return fmt.Errorf("relation references unknown member: %s", r.From)
		}
		if _, exists := memberSet[r.To]; !exists {
			return fmt.Errorf("relation references unknown member: %s", r.To)
		}
		if r.From == r.To {
			return fmt.Errorf("self-referencing relation: %s", r.From)
		}
		if r.Type != RelationLeader && r.Type != RelationPeer {
			return fmt.Errorf("invalid relation type: %s", r.Type)
		}
	}

	return nil
}
