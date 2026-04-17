package workspace

import (
	"fmt"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
)

// agentPaths holds resolved file paths for a specific agent profile.
type agentPaths struct {
	instructionFile string // e.g. {base}/CLAUDE.md
	settingsFile    string // e.g. {base}/.claude/settings.json
	skillsDir       string // e.g. {base}/.claude/skills
}

func resolveAgentPaths(base string, profile domain.AgentProfile) agentPaths {
	return agentPaths{
		instructionFile: filepath.Join(base, profile.InstructionFile),
		settingsFile:    filepath.Join(base, profile.SettingsDir, profile.SettingsFile),
		skillsDir:       filepath.Join(base, profile.SettingsDir, profile.SkillsDir),
	}
}

// Writer materializes agent files from a pre-built resource map.
// The resource map contains all transitive dependencies resolved by
// the ResolveTeam API — the Writer makes ZERO additional API calls.
type Writer struct {
	fs          FileMaterializer
	git         GitRepo
	resourceMap map[string]*api.ResolvedResource
}

// NewWriter creates a Writer backed by a pre-built resource map.
func NewWriter(fs FileMaterializer, git GitRepo, resourceMap map[string]*api.ResolvedResource) *Writer {
	return &Writer{fs: fs, git: git, resourceMap: resourceMap}
}

// MaterializeAgent writes local-clone files for a team that runs as an agent.
func (w *Writer) MaterializeAgent(base string, projection *TeamProjection) error {
	profile, err := domain.ProfileFor(projection.AgentType)
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

	// Write instruction file (CLAUDE.md / AGENTS.md)
	if projection.InstructionRef != nil {
		content, err := w.resolveContent(projection.InstructionRef.Owner, projection.InstructionRef.Name)
		if err != nil {
			return fmt.Errorf("resolve instruction %s/%s: %w", projection.InstructionRef.Owner, projection.InstructionRef.Name, err)
		}
		composed := ComposeInstruction(projection.AgentType, content)
		if err := w.writeFile(paths.instructionFile, composed); err != nil {
			return fmt.Errorf("write %s: %w", profile.InstructionFile, err)
		}
	} else {
		composed := ComposeInstruction(projection.AgentType, "")
		if err := w.writeFile(paths.instructionFile, composed); err != nil {
			return fmt.Errorf("write %s: %w", profile.InstructionFile, err)
		}
	}

	// Write agent settings if referenced
	if projection.SettingsRef != nil {
		content, err := w.resolveContent(projection.SettingsRef.Owner, projection.SettingsRef.Name)
		if err != nil {
			return fmt.Errorf("resolve settings %s/%s: %w", projection.SettingsRef.Owner, projection.SettingsRef.Name, err)
		}
		if err := w.writeFile(paths.settingsFile, content); err != nil {
			return fmt.Errorf("write settings: %w", err)
		}
	}

	// Write skills
	for _, skillRef := range projection.Skills {
		content, err := w.resolveContent(skillRef.Owner, skillRef.Name)
		if err != nil {
			return fmt.Errorf("resolve skill %s/%s: %w", skillRef.Owner, skillRef.Name, err)
		}
		skillPath := SkillFilePath(paths.skillsDir, skillRef.Owner, skillRef.Name)
		if err := w.writeFile(skillPath, content); err != nil {
			return fmt.Errorf("write skill %s: %w", skillRef.Name, err)
		}
	}

	return nil
}

// resolveContent looks up a content resource (instruction, settings, skill)
// from the pre-built resource map and returns its text content.
func (w *Writer) resolveContent(owner, name string) (string, error) {
	key := teamKey(owner, name)
	r, ok := w.resourceMap[key]
	if !ok {
		return "", internalFault("resource %s not found in resolve map", key)
	}
	spec, err := decodeSnapshot[api.ContentSpec](r.Snapshot)
	if err != nil {
		return "", fmt.Errorf("decode content %s: %w", key, err)
	}
	return spec.Content, nil
}

func (w *Writer) writeFile(path, content string) error {
	return w.fs.EnsureFile(path, []byte(content))
}

func (w *Writer) writeWorkLogProtocol(base string) error {
	return w.writeFile(filepath.Join(base, ".clier", workLogProtocolFileName), BuildAgentFacingWorkLogProtocol())
}
