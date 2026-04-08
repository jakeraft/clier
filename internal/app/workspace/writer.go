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

// PrepareMember creates a workspace for a single member.
// Layout:
//
//	{base}/project/CLAUDE.md              <- ClaudeMd
//	{base}/project/.claude/settings.json  <- ClaudeSettings
//	{base}/project/.claude/skills/{name}/SKILL.md <- Skills
func (w *Writer) PrepareMember(base, memberID string) error {
	member, err := w.client.GetMember(w.owner, memberID)
	if err != nil {
		return fmt.Errorf("get member %s: %w", memberID, err)
	}

	projectDir := filepath.Join(base, "project")

	// Write ClaudeMd if referenced
	if member.ClaudeMdID != "" {
		claudeMd, err := w.client.GetClaudeMd(w.owner, member.ClaudeMdID)
		if err != nil {
			return fmt.Errorf("get claude md %s: %w", member.ClaudeMdID, err)
		}
		if err := writeFile(filepath.Join(projectDir, "CLAUDE.md"), claudeMd.Content); err != nil {
			return fmt.Errorf("write CLAUDE.md: %w", err)
		}
	}

	// Write ClaudeSettings if referenced
	if member.ClaudeSettingsID != "" {
		cs, err := w.client.GetClaudeSettings(w.owner, member.ClaudeSettingsID)
		if err != nil {
			return fmt.Errorf("get claude settings %s: %w", member.ClaudeSettingsID, err)
		}
		if err := writeFile(filepath.Join(projectDir, ".claude", "settings.json"), cs.Content); err != nil {
			return fmt.Errorf("write settings.json: %w", err)
		}
	}

	// Write Skills
	for _, skillID := range member.SkillIDs {
		skill, err := w.client.GetSkill(w.owner, skillID)
		if err != nil {
			return fmt.Errorf("get skill %s: %w", skillID, err)
		}
		skillPath := filepath.Join(projectDir, ".claude", "skills", skill.Name, "SKILL.md")
		if err := writeFile(skillPath, skill.Content); err != nil {
			return fmt.Errorf("write skill %s: %w", skill.Name, err)
		}
	}

	// Git clone if GitRepoURL is set
	// TODO: git clone into projectDir when member.GitRepoURL is non-empty.
	// For now, workspace preparation only writes config files.

	return nil
}

// PrepareTeam creates workspaces for all team members.
// Each member gets a subdirectory named after the team member name.
// It also writes a team protocol CLAUDE.md to each member's parent directory.
func (w *Writer) PrepareTeam(base, teamID string) error {
	team, err := w.client.GetTeam(w.owner, teamID)
	if err != nil {
		return fmt.Errorf("get team %s: %w", teamID, err)
	}

	// Build name lookup for protocol generation.
	nameByID := make(map[string]string, len(team.TeamMembers))
	for _, tm := range team.TeamMembers {
		nameByID[tm.ID] = tm.Name
	}

	// Build relations from team.Relations.
	relMap := make(map[string]domain.MemberRelations, len(team.TeamMembers))
	for _, tm := range team.TeamMembers {
		relMap[tm.ID] = domain.MemberRelations{Leaders: []string{}, Workers: []string{}}
	}
	for _, r := range team.Relations {
		from := relMap[r.From]
		from.Workers = append(from.Workers, r.To)
		relMap[r.From] = from

		to := relMap[r.To]
		to.Leaders = append(to.Leaders, r.From)
		relMap[r.To] = to
	}

	for _, tm := range team.TeamMembers {
		memberBase := filepath.Join(base, tm.Name)

		// Prepare member workspace files.
		if err := w.PrepareMember(memberBase, tm.MemberID); err != nil {
			return fmt.Errorf("prepare member %s: %w", tm.Name, err)
		}

		// Write team protocol to parent CLAUDE.md.
		protocol := BuildProtocol(team.Name, tm.Name, relMap[tm.ID], nameByID)
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
