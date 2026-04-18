package workspacecodec

import (
	"encoding/json"

	domainworkspace "github.com/jakeraft/clier/internal/domain/workspace"
)

type TeamProjectionRecord struct {
	Name           string              `json:"name"`
	AgentType      string              `json:"agent_type,omitempty"`
	Command        string              `json:"command,omitempty"`
	GitRepoURL     string              `json:"git_repo_url,omitempty"`
	InstructionRef *ResourceRefRecord  `json:"instruction_ref,omitempty"`
	SettingsRef    *ResourceRefRecord  `json:"settings_ref,omitempty"`
	Skills         []ResourceRefRecord `json:"skills,omitempty"`
	Children       []ChildRecord       `json:"children,omitempty"`
}

type ResourceRefRecord struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type ChildRecord struct {
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	ChildVersion int    `json:"child_version"`
}

func RecordFromDomain(projection domainworkspace.TeamProjection) TeamProjectionRecord {
	var instruction *ResourceRefRecord
	if projection.InstructionRef != nil {
		instruction = &ResourceRefRecord{
			Owner:   projection.InstructionRef.Owner,
			Name:    projection.InstructionRef.Name,
			Version: projection.InstructionRef.Version,
		}
	}
	var settings *ResourceRefRecord
	if projection.SettingsRef != nil {
		settings = &ResourceRefRecord{
			Owner:   projection.SettingsRef.Owner,
			Name:    projection.SettingsRef.Name,
			Version: projection.SettingsRef.Version,
		}
	}
	skills := make([]ResourceRefRecord, 0, len(projection.Skills))
	for _, skill := range projection.Skills {
		skills = append(skills, ResourceRefRecord{
			Owner:   skill.Owner,
			Name:    skill.Name,
			Version: skill.Version,
		})
	}
	children := make([]ChildRecord, 0, len(projection.Children))
	for _, child := range projection.Children {
		children = append(children, ChildRecord{
			Owner:        child.Owner,
			Name:         child.Name,
			ChildVersion: child.ChildVersion,
		})
	}
	return TeamProjectionRecord{
		Name:           projection.Name,
		AgentType:      projection.AgentType,
		Command:        projection.Command,
		GitRepoURL:     projection.GitRepoURL,
		InstructionRef: instruction,
		SettingsRef:    settings,
		Skills:         skills,
		Children:       children,
	}
}

func (r TeamProjectionRecord) ToDomain() domainworkspace.TeamProjection {
	var instruction *domainworkspace.ResourceRef
	if r.InstructionRef != nil {
		instruction = &domainworkspace.ResourceRef{
			Owner:   r.InstructionRef.Owner,
			Name:    r.InstructionRef.Name,
			Version: r.InstructionRef.Version,
		}
	}
	var settings *domainworkspace.ResourceRef
	if r.SettingsRef != nil {
		settings = &domainworkspace.ResourceRef{
			Owner:   r.SettingsRef.Owner,
			Name:    r.SettingsRef.Name,
			Version: r.SettingsRef.Version,
		}
	}
	skills := make([]domainworkspace.ResourceRef, 0, len(r.Skills))
	for _, skill := range r.Skills {
		skills = append(skills, domainworkspace.ResourceRef{
			Owner:   skill.Owner,
			Name:    skill.Name,
			Version: skill.Version,
		})
	}
	children := make([]domainworkspace.Child, 0, len(r.Children))
	for _, child := range r.Children {
		children = append(children, domainworkspace.Child{
			Owner:        child.Owner,
			Name:         child.Name,
			ChildVersion: child.ChildVersion,
		})
	}
	return domainworkspace.TeamProjection{
		Name:           r.Name,
		AgentType:      r.AgentType,
		Command:        r.Command,
		GitRepoURL:     r.GitRepoURL,
		InstructionRef: instruction,
		SettingsRef:    settings,
		Skills:         skills,
		Children:       children,
	}
}

func Marshal(projection domainworkspace.TeamProjection) ([]byte, error) {
	return json.Marshal(RecordFromDomain(projection))
}

func MarshalIndent(projection domainworkspace.TeamProjection) ([]byte, error) {
	return json.MarshalIndent(RecordFromDomain(projection), "", "  ")
}

func Unmarshal(data []byte) (domainworkspace.TeamProjection, error) {
	var rec TeamProjectionRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return domainworkspace.TeamProjection{}, err
	}
	return rec.ToDomain(), nil
}
