package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
)

// agentPaths holds resolved file paths for a specific agent profile.
type agentPaths struct {
	instructionFile   string // e.g. {base}/CLAUDE.md
	settingsFile      string // e.g. {base}/.claude/settings.json
	localSettingsFile string // e.g. {base}/.claude/settings.local.json
	skillsDir         string // e.g. {base}/.claude/skills
}

func resolveAgentPaths(base string, profile domain.AgentProfile) agentPaths {
	return agentPaths{
		instructionFile:   filepath.Join(base, profile.InstructionFile),
		settingsFile:      filepath.Join(base, profile.SettingsDir, profile.SettingsFile),
		localSettingsFile: filepath.Join(base, profile.SettingsDir, profile.LocalSettingsFile),
		skillsDir:         filepath.Join(base, profile.SettingsDir, profile.SkillsDir),
	}
}

// Writer fetches member/team definitions from the server and writes
// the corresponding local-clone files (CLAUDE.md, generated protocols,
// settings.json, settings.local.json, skills) to a local directory.
// Referenced resources are materialized from the pinned versions recorded
// in the member or team definition.
// It is a thin layer: fetch -> write.
type Writer struct {
	client *api.Client
	owner  string
	fs     FileMaterializer
	git    GitRepo
}

type memberWriteOptions struct {
	TeamMemberName string
}

// NewWriter creates a Writer that uses the given API client and owner.
func NewWriter(client *api.Client, owner string, fs FileMaterializer, git GitRepo) *Writer {
	return &Writer{client: client, owner: owner, fs: fs, git: git}
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
	profile := domain.ProfileFor(member.AgentType)
	paths := resolveAgentPaths(base, profile)

	if err := ensureRepoDir(w.fs, w.git, member.GitRepoURL, base); err != nil {
		return fmt.Errorf("materialize repo dir: %w", err)
	}
	if err := w.writeWorkLogProtocol(base); err != nil {
		return fmt.Errorf("write work log protocol: %w", err)
	}

	// Write instruction file (CLAUDE.md / AGENTS.md / GEMINI.md)
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
		if err := w.writeFile(paths.instructionFile, content); err != nil {
			return fmt.Errorf("write %s: %w", profile.InstructionFile, err)
		}
	} else {
		content := ComposeMemberClaudeMd("")
		if opts.TeamMemberName != "" {
			content = ComposeTeamClaudeMd(opts.TeamMemberName, "")
		}
		if err := w.writeFile(paths.instructionFile, content); err != nil {
			return fmt.Errorf("write %s: %w", profile.InstructionFile, err)
		}
	}

	// Write agent settings if referenced
	if member.ClaudeSettings != nil {
		cs, err := w.client.GetClaudeSettingsVersion(member.ClaudeSettings.Owner, member.ClaudeSettings.Name, member.ClaudeSettings.Version)
		if err != nil {
			return fmt.Errorf("get claude settings %s/%s: %w", member.ClaudeSettings.Owner, member.ClaudeSettings.Name, err)
		}
		content, err := loadVersionedContent(cs.Content)
		if err != nil {
			return fmt.Errorf("decode claude settings %s/%s@%d: %w", member.ClaudeSettings.Owner, member.ClaudeSettings.Name, member.ClaudeSettings.Version, err)
		}
		if err := w.writeFile(paths.settingsFile, content); err != nil {
			return fmt.Errorf("write settings: %w", err)
		}
	}
	if err := w.writeLocalSettings(base, profile); err != nil {
		return fmt.Errorf("write local settings: %w", err)
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
		skillPath := filepath.Join(paths.skillsDir, skillRef.Name, "SKILL.md")
		if err := w.writeFile(skillPath, content); err != nil {
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
		if err := w.writeFile(protocolPath, protocol); err != nil {
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

func (w *Writer) writeFile(path, content string) error {
	return w.fs.EnsureFile(path, []byte(content))
}

func (w *Writer) writeLocalSettings(base string, profile domain.AgentProfile) error {
	if profile.LocalSettingsFile == "" {
		return nil
	}
	content, err := localSettingsContent(profile)
	if err != nil {
		return err
	}
	return w.writeFile(filepath.Join(base, profile.SettingsDir, profile.LocalSettingsFile), content)
}

func (w *Writer) writeWorkLogProtocol(base string) error {
	return w.writeFile(filepath.Join(base, ".clier", workLogProtocolFileName), BuildAgentFacingWorkLogProtocol())
}

func localSettingsContent(profile domain.AgentProfile) (string, error) {
	if profile.HomeExcludeKey == "" {
		return "{}", nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	payload := map[string]any{
		profile.HomeExcludeKey: []string{
			filepath.ToSlash(filepath.Join(homeDir, profile.SettingsDir)) + "/**",
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal local settings: %w", err)
	}
	return string(data), nil
}
