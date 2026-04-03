package domain

import "fmt"

type TeamExport struct {
	TeamID         string           `json:"team_id,omitempty"`
	TeamName       string           `json:"team_name"`
	RootMemberName string           `json:"root_member_name"`
	Members        []MemberExport   `json:"members"`
	Relations      []RelationExport `json:"relations"`
}

type MemberExport struct {
	ID            string           `json:"id,omitempty"`
	Name          string           `json:"name"`
	CliProfile    CliProfileExport `json:"cli_profile"`
	SystemPrompts []PromptSnapshot `json:"system_prompts"`
	Envs          []EnvSnapshot    `json:"envs"`
	GitRepo       *GitRepoSnapshot `json:"git_repo"`
}

type CliProfileExport struct {
	ID         string    `json:"id,omitempty"`
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

	// 3. Convert each TeamMemberSnapshot to MemberExport
	members := make([]MemberExport, 0, len(snap.Members))
	for _, m := range snap.Members {
		members = append(members, MemberExport{
			ID:   m.MemberID,
			Name: m.MemberName,
			CliProfile: CliProfileExport{
				ID:         m.CliProfileID,
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

	// 4. Extract relations from TeamMemberSnapshot.Relations
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
		TeamID:         snap.TeamID,
		TeamName:       snap.TeamName,
		RootMemberName: rootMemberName,
		Members:        members,
		Relations:      relations,
	}, nil
}

