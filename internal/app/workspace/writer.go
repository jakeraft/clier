package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
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
func (w *Writer) PrepareTeam(base, teamID string) error {
	team, err := w.client.GetTeam(w.owner, teamID)
	if err != nil {
		return fmt.Errorf("get team %s: %w", teamID, err)
	}

	for _, tm := range team.TeamMembers {
		memberBase := filepath.Join(base, tm.Name)
		if err := w.PrepareMember(memberBase, tm.MemberID); err != nil {
			return fmt.Errorf("prepare member %s: %w", tm.Name, err)
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
