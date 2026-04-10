package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
)

// Writer fetches member/team definitions from the server and writes
// the corresponding local-clone files (CLAUDE.md, generated protocols,
// settings.json, settings.local.json, skills) to a local directory.
// Referenced resources are materialized from the pinned versions recorded
// in the member or team definition.
// It is a thin layer: fetch -> write.
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

// MaterializeMemberFiles writes the local-clone files for a single member.
// Layout:
//
//	{base}/CLAUDE.md              <- generated import wrapper + ClaudeMd
//	{base}/.clier/work-log-protocol.md <- clier-generated work log protocol
//	{base}/.claude/settings.json  <- ClaudeSettings
//	{base}/.claude/settings.local.json <- clier-generated local isolation overlay
//	{base}/.claude/skills/{name}/SKILL.md <- Skills
func (w *Writer) MaterializeMemberFiles(base, memberName string) error {
	member, err := w.client.GetMember(w.owner, memberName)
	if err != nil {
		return fmt.Errorf("get member %s: %w", memberName, err)
	}
	return w.materializeMemberFilesFromResponse(base, member, memberWriteOptions{})
}

// materializeMemberFilesFromResponse writes local-clone files from a MemberResponse.
func (w *Writer) materializeMemberFilesFromResponse(base string, member *api.MemberResponse, opts memberWriteOptions) error {
	if err := ensureRepoDir(member.GitRepoURL, base); err != nil {
		return fmt.Errorf("materialize repo dir: %w", err)
	}
	if err := writeWorkLogProtocol(base); err != nil {
		return fmt.Errorf("write work log protocol: %w", err)
	}

	// Write ClaudeMd if referenced
	if member.ClaudeMd != nil {
		claudeMd, err := w.client.GetClaudeMdVersion(member.ClaudeMd.Owner, member.ClaudeMd.Name, member.ClaudeMd.Version)
		if err != nil {
			return fmt.Errorf("get claude md %s/%s: %w", member.ClaudeMd.Owner, member.ClaudeMd.Name, err)
		}
		content, err := loadVersionedContent(claudeMd.Content)
		if err != nil {
			return fmt.Errorf("decode claude md %s/%s@%d: %w", member.ClaudeMd.Owner, member.ClaudeMd.Name, member.ClaudeMd.Version, err)
		}
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
		cs, err := w.client.GetClaudeSettingsVersion(member.ClaudeSettings.Owner, member.ClaudeSettings.Name, member.ClaudeSettings.Version)
		if err != nil {
			return fmt.Errorf("get claude settings %s/%s: %w", member.ClaudeSettings.Owner, member.ClaudeSettings.Name, err)
		}
		content, err := loadVersionedContent(cs.Content)
		if err != nil {
			return fmt.Errorf("decode claude settings %s/%s@%d: %w", member.ClaudeSettings.Owner, member.ClaudeSettings.Name, member.ClaudeSettings.Version, err)
		}
		if err := writeFile(filepath.Join(base, ".claude", "settings.json"), content); err != nil {
			return fmt.Errorf("write settings.json: %w", err)
		}
	}
	if err := writeLocalSettings(base); err != nil {
		return fmt.Errorf("write settings.local.json: %w", err)
	}

	// Write Skills
	for _, skillRef := range member.Skills {
		skill, err := w.client.GetSkillVersion(skillRef.Owner, skillRef.Name, skillRef.Version)
		if err != nil {
			return fmt.Errorf("get skill %s/%s: %w", skillRef.Owner, skillRef.Name, err)
		}
		content, err := loadVersionedContent(skill.Content)
		if err != nil {
			return fmt.Errorf("decode skill %s/%s@%d: %w", skillRef.Owner, skillRef.Name, skillRef.Version, err)
		}
		skillPath := filepath.Join(base, ".claude", "skills", skillRef.Name, "SKILL.md")
		if err := writeFile(skillPath, content); err != nil {
			return fmt.Errorf("write skill %s: %w", skillRef.Name, err)
		}
	}

	return nil
}

// MaterializeTeamFiles writes local-clone files for all team members.
// Each member gets a subdirectory named after the team member name.
// The team local clone owns a single root .clier directory for runtime metadata,
// while each member owns a generated-only .clier directory for imported
// protocol files inside its own working tree.
func (w *Writer) MaterializeTeamFiles(base, teamName string) error {
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

		member, err := w.loadPinnedMemberResponse(tm.Member.Owner, tm.Member.Name, tm.Member.Version)
		if err != nil {
			return fmt.Errorf("get member %s: %w", tm.Name, err)
		}
		if err := w.materializeMemberFilesFromResponse(memberBase, member, memberWriteOptions{
			TeamMemberName: tm.Name,
		}); err != nil {
			return fmt.Errorf("materialize member %s: %w", tm.Name, err)
		}
		protocol := BuildAgentFacingTeamProtocol(team.Name, tm.Name, relMap[tm.ID], membersByID)
		protocolPath := filepath.Join(memberBase, ".clier", TeamProtocolFileName(tm.Name))
		if err := writeFile(protocolPath, protocol); err != nil {
			return fmt.Errorf("write protocol for %s: %w", tm.Name, err)
		}
	}

	return nil
}

func (w *Writer) loadPinnedMemberResponse(owner, name string, version int) (*api.MemberResponse, error) {
	memberVersion, err := w.client.GetMemberVersion(owner, name, version)
	if err != nil {
		return nil, err
	}
	snapshot, err := loadMemberSnapshot(memberVersion.Content)
	if err != nil {
		return nil, fmt.Errorf("decode member %s/%s@%d: %w", owner, name, version, err)
	}
	return memberResponseFromSnapshot(owner, name, version, snapshot), nil
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
	return writeFile(filepath.Join(base, ".clier", workLogProtocolFileName), BuildAgentFacingWorkLogProtocol())
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
