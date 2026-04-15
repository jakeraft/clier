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

// materializeMemberFiles writes local-clone files from a MemberProjection.
func (w *Writer) materializeMemberFiles(base string, projection *MemberProjection, agentType string, opts memberWriteOptions) error {
	profile, err := domain.ProfileFor(agentType)
	if err != nil {
		return err
	}
	paths := resolveAgentPaths(base, profile)

	if err := ensureRepoDir(w.fs, w.git, projection.GitRepoURL, base); err != nil {
		return fmt.Errorf("materialize repo dir: %w", err)
	}
	if err := w.writeWorkLogProtocol(base); err != nil {
		return fmt.Errorf("write work log protocol: %w", err)
	}

	// Write instruction file (CLAUDE.md / AGENTS.md / GEMINI.md)
	if projection.InstructionRef != nil {
		vr, err := w.client.GetResourceVersion(projection.InstructionRef.Owner, projection.InstructionRef.Name, projection.InstructionRef.Version)
		if err != nil {
			return fmt.Errorf("get instruction %s/%s: %w", projection.InstructionRef.Owner, projection.InstructionRef.Name, err)
		}
		contentSpec, err := decodeSnapshot[api.ContentSpec](vr.Snapshot)
		if err != nil {
			return fmt.Errorf("decode instruction %s/%s@%d: %w", projection.InstructionRef.Owner, projection.InstructionRef.Name, projection.InstructionRef.Version, err)
		}
		content := ComposeInstruction(agentType, opts.TeamMemberName, contentSpec.Content)
		if err := w.writeFile(paths.instructionFile, content); err != nil {
			return fmt.Errorf("write %s: %w", profile.InstructionFile, err)
		}
	} else {
		content := ComposeInstruction(agentType, opts.TeamMemberName, "")
		if err := w.writeFile(paths.instructionFile, content); err != nil {
			return fmt.Errorf("write %s: %w", profile.InstructionFile, err)
		}
	}

	// Write agent settings if referenced
	if projection.SettingsRef != nil {
		vr, err := w.client.GetResourceVersion(projection.SettingsRef.Owner, projection.SettingsRef.Name, projection.SettingsRef.Version)
		if err != nil {
			return fmt.Errorf("get settings %s/%s: %w", projection.SettingsRef.Owner, projection.SettingsRef.Name, err)
		}
		contentSpec, err := decodeSnapshot[api.ContentSpec](vr.Snapshot)
		if err != nil {
			return fmt.Errorf("decode settings %s/%s@%d: %w", projection.SettingsRef.Owner, projection.SettingsRef.Name, projection.SettingsRef.Version, err)
		}
		if err := w.writeFile(paths.settingsFile, contentSpec.Content); err != nil {
			return fmt.Errorf("write settings: %w", err)
		}
	}
	if err := w.writeLocalSettings(base, profile); err != nil {
		return fmt.Errorf("write local settings: %w", err)
	}

	// Write Skills
	for _, skillRef := range projection.Skills {
		vr, err := w.client.GetResourceVersion(skillRef.Owner, skillRef.Name, skillRef.Version)
		if err != nil {
			return fmt.Errorf("get skill %s/%s: %w", skillRef.Owner, skillRef.Name, err)
		}
		contentSpec, err := decodeSnapshot[api.ContentSpec](vr.Snapshot)
		if err != nil {
			return fmt.Errorf("decode skill %s/%s@%d: %w", skillRef.Owner, skillRef.Name, skillRef.Version, err)
		}
		skillPath := filepath.Join(paths.skillsDir, skillRef.Name, "SKILL.md")
		if err := w.writeFile(skillPath, contentSpec.Content); err != nil {
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
func (w *Writer) MaterializeTeamFiles(base string, team *api.ResourceResponse, teamSpec *api.TeamSpec) error {

	// Build member lookup for protocol generation from member refs.
	// Key by owner/name — unique identifier across orgs.
	tmRefs := refsByRelType(team, string(api.KindMember))
	membersByKey := make(map[string]ProtocolMember, len(tmRefs))
	for _, ref := range tmRefs {
		membersByKey[memberKey(ref.OwnerName, ref.Name)] = ProtocolMember{
			Owner: ref.OwnerName,
			Name:  ref.Name,
		}
	}

	// Build relations from team spec.
	relMap := make(map[string]domain.MemberRelations, len(tmRefs))
	for _, ref := range tmRefs {
		relMap[memberKey(ref.OwnerName, ref.Name)] = domain.MemberRelations{Leaders: []string{}, Workers: []string{}}
	}
	for _, r := range teamSpec.Relations {
		fromKey := memberKey(r.From.Owner, r.From.Name)
		toKey := memberKey(r.To.Owner, r.To.Name)

		from := relMap[fromKey]
		from.Workers = append(from.Workers, toKey)
		relMap[fromKey] = from

		to := relMap[toKey]
		to.Leaders = append(to.Leaders, fromKey)
		relMap[toKey] = to
	}

	for _, tm := range tmRefs {
		memberBase := filepath.Join(base, tm.Name)
		tmKey := memberKey(tm.OwnerName, tm.Name)

		projection, agentType, err := w.loadPinnedMember(tm.OwnerName, tm.Name, tm.TargetVersion, tm.AgentType)
		if err != nil {
			return fmt.Errorf("get member %s: %w", tm.Name, err)
		}

		protocol := BuildAgentFacingTeamProtocol(team.Metadata.Name, tm.Name, relMap[tmKey], membersByKey)

		if err := w.materializeMemberFiles(memberBase, projection, agentType, memberWriteOptions{
			TeamMemberName: tm.Name,
		}); err != nil {
			return fmt.Errorf("materialize member %s: %w", tm.Name, err)
		}
		protocolPath := filepath.Join(memberBase, ".clier", TeamProtocolFileName(tm.Name))
		if err := w.writeFile(protocolPath, protocol); err != nil {
			return fmt.Errorf("write protocol for %s: %w", tm.Name, err)
		}
	}

	return nil
}

// loadPinnedMember builds a MemberProjection and resolves agent type from a
// pinned version snapshot. The refAgentType (from the team_member ref) takes
// precedence over the snapshot's agent_type.
func (w *Writer) loadPinnedMember(owner, name string, version int, refAgentType string) (*MemberProjection, string, error) {
	vr, err := w.client.GetResourceVersion(owner, name, version)
	if err != nil {
		return nil, "", err
	}
	projection := memberProjectionFromSnapshot(name, vr)
	agentType := agentTypeFromSnapshot(vr.Snapshot, refAgentType)
	return projection, agentType, nil
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
