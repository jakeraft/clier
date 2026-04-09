package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
)

// Writer fetches member/team definitions from the server and writes
// the corresponding workspace files (CLAUDE.md, settings.json, skills)
// to a local directory. It is a thin layer: fetch -> write.
type Writer struct {
	client *api.Client
	owner  string
}

// NewWriter creates a Writer that uses the given API client and owner.
func NewWriter(client *api.Client, owner string) *Writer {
	return &Writer{client: client, owner: owner}
}

// PrepareMember creates a workspace for a single member (by name).
// Layout:
//
//	{base}/project/CLAUDE.md              <- ClaudeMd
//	{base}/project/.claude/settings.json  <- ClaudeSettings
//	{base}/project/.claude/skills/{name}/SKILL.md <- Skills
func (w *Writer) PrepareMember(base, memberName string) error {
	member, err := w.client.GetMember(w.owner, memberName)
	if err != nil {
		return fmt.Errorf("get member %s: %w", memberName, err)
	}
	return w.prepareMemberFromResponse(base, member)
}

// prepareMemberFromResponse creates workspace files from a MemberResponse.
func (w *Writer) prepareMemberFromResponse(base string, member *api.MemberResponse) error {
	projectDir := filepath.Join(base, "project")

	// Write ClaudeMd if referenced
	if member.ClaudeMd != nil {
		claudeMd, err := w.client.GetClaudeMd(member.ClaudeMd.Owner, member.ClaudeMd.Name)
		if err != nil {
			return fmt.Errorf("get claude md %s/%s: %w", member.ClaudeMd.Owner, member.ClaudeMd.Name, err)
		}
		if err := writeFile(filepath.Join(projectDir, "CLAUDE.md"), claudeMd.Content); err != nil {
			return fmt.Errorf("write CLAUDE.md: %w", err)
		}
	}

	// Write ClaudeSettings if referenced
	if member.ClaudeSettings != nil {
		cs, err := w.client.GetClaudeSettings(member.ClaudeSettings.Owner, member.ClaudeSettings.Name)
		if err != nil {
			return fmt.Errorf("get claude settings %s/%s: %w", member.ClaudeSettings.Owner, member.ClaudeSettings.Name, err)
		}
		if err := writeFile(filepath.Join(projectDir, ".claude", "settings.json"), cs.Content); err != nil {
			return fmt.Errorf("write settings.json: %w", err)
		}
	}

	// Write Skills
	for _, skillRef := range member.Skills {
		skill, err := w.client.GetSkill(skillRef.Owner, skillRef.Name)
		if err != nil {
			return fmt.Errorf("get skill %s/%s: %w", skillRef.Owner, skillRef.Name, err)
		}
		skillPath := filepath.Join(projectDir, ".claude", "skills", skill.Name, "SKILL.md")
		if err := writeFile(skillPath, skill.Content); err != nil {
			return fmt.Errorf("write skill %s: %w", skill.Name, err)
		}
	}

	return nil
}

// PrepareTeam creates workspaces for all team members.
// Each member gets a subdirectory named after the team member name.
// It also writes a team protocol CLAUDE.md to each member's parent directory.
func (w *Writer) PrepareTeam(base, teamName string) error {
	team, err := w.client.GetTeam(w.owner, teamName)
	if err != nil {
		return fmt.Errorf("get team %s: %w", teamName, err)
	}

	// Build member lookup for protocol generation.
	membersByID := make(map[int64]ProtocolMember, len(team.TeamMembers))
	for _, tm := range team.TeamMembers {
		membersByID[tm.ID] = ProtocolMember{
			ID:   tm.ID,
			Name: tm.Name,
		}
	}

	// Build relations from team.Relations.
	relMap := make(map[int64]domain.MemberRelations, len(team.TeamMembers))
	for _, tm := range team.TeamMembers {
		relMap[tm.ID] = domain.MemberRelations{Leaders: []int64{}, Workers: []int64{}}
	}
	for _, r := range team.Relations {
		from := relMap[r.FromTeamMemberID]
		from.Workers = append(from.Workers, r.ToTeamMemberID)
		relMap[r.FromTeamMemberID] = from

		to := relMap[r.ToTeamMemberID]
		to.Leaders = append(to.Leaders, r.FromTeamMemberID)
		relMap[r.ToTeamMemberID] = to
	}

	for _, tm := range team.TeamMembers {
		memberBase := filepath.Join(base, tm.Name)

		// Fetch member and prepare workspace using ResourceRef.
		member, err := w.client.GetMember(tm.Member.Owner, tm.Member.Name)
		if err != nil {
			return fmt.Errorf("get member %s: %w", tm.Name, err)
		}
		if err := w.prepareMemberFromResponse(memberBase, member); err != nil {
			return fmt.Errorf("prepare member %s: %w", tm.Name, err)
		}

		// Write team protocol to parent CLAUDE.md.
		protocol := BuildProtocol(team.Name, tm.Name, relMap[tm.ID], membersByID)
		protocolPath := filepath.Join(memberBase, "CLAUDE.md")
		if err := writeFile(protocolPath, protocol); err != nil {
			return fmt.Errorf("write protocol for %s: %w", tm.Name, err)
		}
	}

	return nil
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
