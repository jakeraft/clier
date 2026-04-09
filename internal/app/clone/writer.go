package clone

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
)

// Writer fetches member/team definitions from the server and writes
// the corresponding workspace files (CLAUDE.md, generated protocols, settings.json, settings.local.json, skills)
// to a local directory. It is a thin layer: fetch -> write.
type Writer struct {
	client *api.Client
	owner  string
}

type memberWriteOptions struct {
	TeamMemberName string
}

// NewWriter creates a Writer that uses the given API client and owner.
func NewWriter(client *api.Client, owner string) *Writer {
	return &Writer{client: client, owner: owner}
}

// PrepareMember creates a workspace for a single member (by name).
// Layout:
//
//	{base}/CLAUDE.md              <- generated import wrapper + ClaudeMd
//	{base}/.clier/work-log-protocol.md <- clier-generated work log protocol
//	{base}/.claude/settings.json  <- ClaudeSettings
//	{base}/.claude/settings.local.json <- clier-generated local isolation overlay
//	{base}/.claude/skills/{name}/SKILL.md <- Skills
func (w *Writer) PrepareMember(base, memberName string) error {
	member, err := w.client.GetMember(w.owner, memberName)
	if err != nil {
		return fmt.Errorf("get member %s: %w", memberName, err)
	}
	return w.prepareMemberFromResponse(base, member, memberWriteOptions{})
}

// prepareMemberFromResponse creates workspace files from a MemberResponse.
func (w *Writer) prepareMemberFromResponse(base string, member *api.MemberResponse, opts memberWriteOptions) error {
	if err := ensureRepoDir(member.GitRepoURL, base); err != nil {
		return fmt.Errorf("prepare repo dir: %w", err)
	}
	if err := writeWorkLogProtocol(base); err != nil {
		return fmt.Errorf("write work log protocol: %w", err)
	}

	// Write ClaudeMd if referenced
	if member.ClaudeMd != nil {
		claudeMd, err := w.client.GetClaudeMd(member.ClaudeMd.Owner, member.ClaudeMd.Name)
		if err != nil {
			return fmt.Errorf("get claude md %s/%s: %w", member.ClaudeMd.Owner, member.ClaudeMd.Name, err)
		}
		content := claudeMd.Content
		if opts.TeamMemberName != "" {
			content = ComposeTeamClaudeMd(opts.TeamMemberName, content)
		} else {
			content = ComposeMemberClaudeMd(content)
		}
		if err := writeFile(filepath.Join(base, "CLAUDE.md"), content); err != nil {
			return fmt.Errorf("write CLAUDE.md: %w", err)
		}
	} else {
		content := ComposeMemberClaudeMd("")
		if opts.TeamMemberName != "" {
			content = ComposeTeamClaudeMd(opts.TeamMemberName, "")
		}
		if err := writeFile(filepath.Join(base, "CLAUDE.md"), content); err != nil {
			return fmt.Errorf("write CLAUDE.md: %w", err)
		}
	}

	// Write ClaudeSettings if referenced
	if member.ClaudeSettings != nil {
		cs, err := w.client.GetClaudeSettings(member.ClaudeSettings.Owner, member.ClaudeSettings.Name)
		if err != nil {
			return fmt.Errorf("get claude settings %s/%s: %w", member.ClaudeSettings.Owner, member.ClaudeSettings.Name, err)
		}
		if err := writeFile(filepath.Join(base, ".claude", "settings.json"), cs.Content); err != nil {
			return fmt.Errorf("write settings.json: %w", err)
		}
	}
	if err := writeLocalSettings(base); err != nil {
		return fmt.Errorf("write settings.local.json: %w", err)
	}

	// Write Skills
	for _, skillRef := range member.Skills {
		skill, err := w.client.GetSkill(skillRef.Owner, skillRef.Name)
		if err != nil {
			return fmt.Errorf("get skill %s/%s: %w", skillRef.Owner, skillRef.Name, err)
		}
		skillPath := filepath.Join(base, ".claude", "skills", skill.Name, "SKILL.md")
		if err := writeFile(skillPath, skill.Content); err != nil {
			return fmt.Errorf("write skill %s: %w", skill.Name, err)
		}
	}

	return nil
}

// PrepareTeam creates workspaces for all team members.
// Each member gets a subdirectory named after the team member name.
// The team clone owns a single root .clier directory for runtime metadata,
// while each member owns a generated-only .clier directory for imported
// protocol files inside its own working tree.
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
		if err := w.prepareMemberFromResponse(memberBase, member, memberWriteOptions{
			TeamMemberName: tm.Name,
		}); err != nil {
			return fmt.Errorf("prepare member %s: %w", tm.Name, err)
		}
		protocol := BuildProtocol(team.Name, tm.Name, relMap[tm.ID], membersByID)
		protocolPath := filepath.Join(memberBase, ".clier", TeamProtocolFileName(tm.Name))
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

func writeLocalSettings(base string) error {
	content, err := localSettingsContent()
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(base, ".claude", "settings.local.json"), content)
}

func writeWorkLogProtocol(base string) error {
	return writeFile(filepath.Join(base, ".clier", workLogProtocolFileName), BuildWorkLogProtocol())
}

func localSettingsContent() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	payload := map[string]any{
		"claudeMdExcludes": []string{
			filepath.ToSlash(filepath.Join(homeDir, ".claude")) + "/**",
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal local settings: %w", err)
	}
	return string(data), nil
}
