package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/adapter/api"
	apprun "github.com/jakeraft/clier/internal/app/run"
	"github.com/jakeraft/clier/internal/domain"
)

type Service struct {
	client *api.Client
	fs     FileMaterializer
	git    GitRepo
}

type Status struct {
	WorkingCopy WorkingCopyStatus `json:"working_copy"`
	Local       string            `json:"local"`
	Tracked     []TrackedStatus   `json:"tracked"`
	Runs        RunStatusSummary  `json:"runs"`
}

type WorkingCopyStatus struct {
	Root     string    `json:"root"`
	Kind     string    `json:"kind"`
	Owner    string    `json:"owner"`
	Name     string    `json:"name"`
	ClonedAt time.Time `json:"cloned_at"`
}

type TrackedStatus struct {
	Kind          string `json:"kind"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	Local         string `json:"local"`
	PinnedVersion *int   `json:"pinned_version,omitempty"`
	LatestVersion *int   `json:"latest_version,omitempty"`
	Remote        string `json:"remote,omitempty"`
}

type RunStatusSummary struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
}

type PushResult struct {
	Status          string `json:"status"`
	Pushed          int    `json:"pushed"`
	PulledAfterPush bool   `json:"pulled_after_push"`
}

func NewService(client *api.Client, fs FileMaterializer, git GitRepo) *Service {
	return &Service{client: client, fs: fs, git: git}
}

// --- Clone ---

// agentEntry is a team node that will be materialized as a runnable agent.
type agentEntry struct {
	name       string
	owner      string
	version    int
	projection *TeamProjection
	localBase  string // relative path for tracked resources
}

func (s *Service) Clone(base, owner, name string) (*Manifest, error) {
	resolved, err := s.client.ResolveTeam(owner, name)
	if err != nil {
		return nil, fmt.Errorf("resolve team %s/%s: %w", owner, name, err)
	}

	resourceMap := buildResourceMap(resolved.Resources)
	root := &resolved.Root
	rootProjection := teamProjectionFromResolved(root)

	// Write root team projection.
	if err := WriteTeamProjection(s.fs, TeamProjectionPath(base), rootProjection); err != nil {
		return nil, err
	}

	// Composite pattern: collect all runnable agents from the tree.
	// Every node with a known agent profile is materialized uniformly —
	// no distinction between leaf and composite.
	agents := s.collectCloneAgents(rootProjection, root, resourceMap)
	if len(agents) == 0 {
		return nil, fmt.Errorf("team %s/%s has no runnable agents (unknown agent type %q)", owner, name, rootProjection.AgentType)
	}

	// Build protocol lookup — all agents are peers.
	allKeys := make([]string, 0, len(agents))
	agentsByKey := make(map[string]ProtocolAgent, len(agents))
	for _, a := range agents {
		key := teamKey(a.owner, a.name)
		allKeys = append(allKeys, key)
		agentsByKey[key] = ProtocolAgent{Owner: a.owner, Name: a.name}
	}

	tracked := []TrackedResource{{
		Kind:          string(api.KindTeam),
		Owner:         root.OwnerName,
		Name:          rootProjection.Name,
		LocalPath:     TeamProjectionLocalPath(),
		RemoteVersion: intPtr(root.Version),
		Editable:      true,
	}}
	generated := []string{}
	writer := NewWriter(s.fs, s.git, resourceMap)

	for _, a := range agents {
		agentBase := filepath.Join(base, a.name)

		// Materialize agent files.
		if err := writer.MaterializeAgent(agentBase, a.projection, a.name); err != nil {
			return nil, fmt.Errorf("materialize agent %s: %w", a.name, err)
		}

		// Write team protocol — peers = all other agents.
		aKey := teamKey(a.owner, a.name)
		relations := buildPeerRelations(aKey, allKeys)
		protocol := BuildAgentFacingTeamProtocol(rootProjection.Name, a.name, relations, agentsByKey)
		protocolPath := filepath.Join(agentBase, ".clier", TeamProtocolFileName(a.name))
		if err := s.fs.EnsureFile(protocolPath, []byte(protocol)); err != nil {
			return nil, fmt.Errorf("write protocol for %s: %w", a.name, err)
		}

		// Write child projection (root already written as team.json).
		isRoot := a.name == rootProjection.Name && a.owner == root.OwnerName
		if !isRoot {
			if err := WriteTeamProjection(s.fs, ChildTeamProjectionPath(base, a.name), a.projection); err != nil {
				return nil, err
			}
			tracked = append(tracked, TrackedResource{
				Kind:          string(api.KindTeam),
				Owner:         a.owner,
				Name:          a.name,
				LocalPath:     ChildTeamProjectionLocalPath(a.name),
				RemoteVersion: intPtr(a.version),
				Editable:      true,
			})
		}

		profile, _ := domain.ProfileFor(a.projection.AgentType)
		generated = append(generated,
			filepath.ToSlash(filepath.Join(a.localBase, ".clier", "work-log-protocol.md")),
			filepath.ToSlash(filepath.Join(a.localBase, ".clier", TeamProtocolFileName(a.name))),
		)
		appendAgentTrackedResources(&tracked, &generated, a.projection, a.localBase, profile)
	}

	if err := s.populateBaseHashes(base, tracked); err != nil {
		return nil, err
	}
	manifest := &Manifest{
		Kind:             string(api.KindTeam),
		Owner:            root.OwnerName,
		Name:             rootProjection.Name,
		ClonedAt:         time.Now().UTC(),
		RootResource:     tracked[0],
		TrackedResources: tracked,
		GeneratedFiles:   normalizePaths(generated),
	}
	if err := SaveManifest(s.fs, base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

// collectCloneAgents walks the team tree and collects every node that has
// a known agent profile (ProfileFor succeeds). Uniform for leaf and composite.
func (s *Service) collectCloneAgents(rootProjection *TeamProjection, root *api.ResolvedResource, resourceMap map[string]*api.ResolvedResource) []agentEntry {
	var agents []agentEntry

	// Root node.
	if _, err := domain.ProfileFor(rootProjection.AgentType); err == nil {
		agents = append(agents, agentEntry{
			name: rootProjection.Name, owner: root.OwnerName,
			version: root.Version, projection: rootProjection,
			localBase: filepath.ToSlash(rootProjection.Name),
		})
	}

	// Children.
	for _, child := range rootProjection.Children {
		childKey := teamKey(child.Owner, child.Name)
		childResource, ok := resourceMap[childKey]
		if !ok {
			continue
		}
		childProjection := teamProjectionFromResolved(childResource)
		if _, err := domain.ProfileFor(childProjection.AgentType); err == nil {
			agents = append(agents, agentEntry{
				name: child.Name, owner: child.Owner,
				version: child.ChildVersion, projection: childProjection,
				localBase: filepath.ToSlash(child.Name),
			})
		}
	}

	return agents
}

// --- Pull ---

func (s *Service) Pull(base string, force bool) (*Manifest, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	pulled, err := s.pullTarget(base, manifest, force)
	if err != nil {
		return nil, err
	}
	if err := SaveManifest(s.fs, base, pulled); err != nil {
		return nil, err
	}
	return pulled, nil
}

func (s *Service) pullTarget(base string, manifest *Manifest, force bool) (*Manifest, error) {
	if !force {
		modified, err := s.ModifiedTrackedResources(base)
		if err != nil {
			return nil, err
		}
		if len(modified) > 0 {
			paths := make([]string, 0, len(modified))
			for _, resource := range modified {
				paths = append(paths, resource.LocalPath)
			}
			slices.Sort(paths)
			return nil, fmt.Errorf("local changes prevent pull; push or revert first: %s", strings.Join(paths, ", "))
		}
	}

	// Use ResolveTeam for a full refresh.
	resolved, err := s.client.ResolveTeam(manifest.Owner, manifest.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve team %s/%s: %w", manifest.Owner, manifest.Name, err)
	}
	resourceMap := buildResourceMap(resolved.Resources)
	// Also add root to map for child lookups.
	rootKey := teamKey(resolved.Root.OwnerName, resolved.Root.Name)
	resourceMap[rootKey] = &resolved.Root

	for i := range manifest.TrackedResources {
		tr := &manifest.TrackedResources[i]
		key := teamKey(tr.Owner, tr.Name)
		localPath := filepath.Join(base, filepath.FromSlash(tr.LocalPath))

		switch api.ResourceKind(tr.Kind) {
		case api.KindTeam:
			r, ok := resourceMap[key]
			if !ok {
				return nil, fmt.Errorf("pull team %s: not found in resolve response", key)
			}
			projection := teamProjectionFromResolved(r)
			if err := WriteTeamProjection(s.fs, localPath, projection); err != nil {
				return nil, err
			}
			tr.RemoteVersion = intPtr(r.Version)

		case api.KindInstruction, api.KindClaudeSettings, api.KindCodexSettings, api.KindSkill:
			r, ok := resourceMap[key]
			if !ok {
				return nil, fmt.Errorf("pull %s %s: not found in resolve response", tr.Kind, key)
			}
			spec, err := decodeSnapshot[api.ContentSpec](r.Snapshot)
			if err != nil {
				return nil, fmt.Errorf("decode %s spec: %w", tr.Kind, err)
			}
			content := spec.Content
			if api.IsInstructionKind(tr.Kind) {
				agentName := filepath.ToSlash(tr.LocalPath)
				if idx := strings.Index(agentName, "/"); idx >= 0 {
					agentName = agentName[:idx]
				}
				agentType := tr.AgentType
				if agentType == "" {
					agentType = "claude"
				}
				content = ComposeInstruction(agentType, agentName, content)
			}
			if err := s.fs.EnsureFile(localPath, []byte(content)); err != nil {
				return nil, fmt.Errorf("write %s: %w", tr.LocalPath, err)
			}
			tr.RemoteVersion = intPtr(r.Version)
		}
	}

	// Recalculate base hashes after updating files.
	if err := s.populateBaseHashes(base, manifest.TrackedResources); err != nil {
		return nil, err
	}
	return manifest, nil
}

// --- Status ---

func (s *Service) Status(base string) (*Status, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	tracked, modifiedCount, err := s.trackedStatuses(base, manifest)
	if err != nil {
		return nil, err
	}
	runs, err := s.runSummary(base)
	if err != nil {
		return nil, err
	}
	local := "clean"
	if modifiedCount > 0 {
		local = "modified"
	}
	// Check remote versions — deduplicate by owner/name to avoid redundant API calls.
	type remoteVersion struct {
		latest int
		err    error
	}
	remoteCache := map[string]*remoteVersion{}
	for i, tr := range manifest.TrackedResources {
		if tr.RemoteVersion == nil {
			continue
		}
		key := tr.Owner + "/" + tr.Name
		if _, ok := remoteCache[key]; !ok {
			res, err := s.client.GetResource(tr.Owner, tr.Name)
			if err != nil {
				remoteCache[key] = &remoteVersion{err: err}
			} else {
				remoteCache[key] = &remoteVersion{latest: res.Metadata.LatestVersion}
			}
		}
		rv := remoteCache[key]
		if rv.err != nil {
			continue
		}
		pinned := *tr.RemoteVersion
		tracked[i].PinnedVersion = &pinned
		tracked[i].LatestVersion = &rv.latest
		if pinned < rv.latest {
			tracked[i].Remote = "behind"
		} else {
			tracked[i].Remote = "up-to-date"
		}
	}

	status := &Status{
		WorkingCopy: WorkingCopyStatus{
			Root:     base,
			Kind:     manifest.Kind,
			Owner:    manifest.Owner,
			Name:     manifest.Name,
			ClonedAt: manifest.ClonedAt,
		},
		Local:   local,
		Tracked: tracked,
		Runs:    runs,
	}

	return status, nil
}

// --- Push ---

func (s *Service) Push(base string) (*PushResult, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	modified, err := s.ModifiedTrackedResources(base)
	if err != nil {
		return nil, err
	}
	if len(modified) == 0 {
		return &PushResult{Status: "no_changes", Pushed: 0, PulledAfterPush: false}, nil
	}

	pushed := 0
	targetName := manifest.Name
	for _, resource := range modified {
		if !resource.Editable {
			continue
		}
		pushed++

		kind, body, err := s.preparePushBody(base, manifest, resource)
		if err != nil {
			return nil, err
		}
		updated, err := s.pushResource(resource, kind, body)
		if err != nil {
			return nil, err
		}
		if updated != nil && resource.LocalPath == manifest.RootResource.LocalPath {
			targetName = updated.Metadata.Name
		}
	}

	// Update manifest name if root resource was renamed, then pull latest.
	manifest.Name = targetName
	pulled, err := s.pullTarget(base, manifest, true)
	if err != nil {
		return nil, err
	}
	if err := SaveManifest(s.fs, base, pulled); err != nil {
		return nil, err
	}
	return &PushResult{Status: "pushed", Pushed: pushed, PulledAfterPush: true}, nil
}

// preparePushBody builds the request body for a single tracked resource.
func (s *Service) preparePushBody(base string, manifest *Manifest, r TrackedResource) (api.ResourceKind, any, error) {
	kind := api.ResourceKind(r.Kind)
	switch kind {
	case api.KindTeam:
		projection, err := LoadTeamProjection(s.fs, filepath.Join(base, filepath.FromSlash(r.LocalPath)))
		if err != nil {
			return "", nil, err
		}
		body, err := s.teamMutationFromProjection(projection)
		if err != nil {
			return "", nil, err
		}
		return kind, body, nil
	default:
		content, err := s.readContentForPush(base, manifest, kind, r)
		if err != nil {
			return "", nil, err
		}
		return kind, api.ContentWriteRequest{Name: r.Name, Content: content}, nil
	}
}

// pushResource checks the remote version and uploads the resource.
func (s *Service) pushResource(r TrackedResource, kind api.ResourceKind, body any) (*api.ResourceResponse, error) {
	current, err := s.client.GetResource(r.Owner, r.Name)
	if err != nil {
		return nil, err
	}
	if !versionsMatch(r.RemoteVersion, current.Metadata.LatestVersion) {
		return nil, fmt.Errorf("remote %s %s/%s changed; pull before pushing", r.Kind, r.Owner, r.Name)
	}
	return s.client.UpdateResource(kind, r.Owner, r.Name, body)
}

// readContentForPush reads a content resource from disk, stripping the prelude for instruction kinds.
func (s *Service) readContentForPush(base string, manifest *Manifest, kind api.ResourceKind, r TrackedResource) (string, error) {
	if api.IsInstructionKind(string(kind)) {
		return s.serverInstructionContent(base, manifest, r)
	}
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(r.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", r.LocalPath, err)
	}
	return string(data), nil
}

// teamMutationFromProjection builds a TeamWriteRequest from a TeamProjection.
// Works for both leaf teams (agent fields) and composite teams (children).
func (s *Service) teamMutationFromProjection(projection *TeamProjection) (*api.TeamWriteRequest, error) {
	req := &api.TeamWriteRequest{
		Name:       projection.Name,
		Command:    projection.Command,
		GitRepoURL: projection.GitRepoURL,
	}

	// Instruction and settings refs require an agent profile to resolve the kind.
	// Only resolve when refs are present (leaf teams); composite teams skip this.
	if projection.InstructionRef != nil || projection.SettingsRef != nil {
		profile, err := domain.ProfileFor(projection.AgentType)
		if err != nil {
			return nil, fmt.Errorf("resolve agent profile: %w", err)
		}
		if projection.InstructionRef != nil {
			ref := &api.ResourceRefRequest{
				Owner:   projection.InstructionRef.Owner,
				Name:    projection.InstructionRef.Name,
				Version: projection.InstructionRef.Version,
			}
			req.SetInstructionRef(ref)
		}
		if projection.SettingsRef != nil {
			ref := &api.ResourceRefRequest{
				Owner:   projection.SettingsRef.Owner,
				Name:    projection.SettingsRef.Name,
				Version: projection.SettingsRef.Version,
			}
			req.SetSettingsRef(profile.SettingsKind, ref)
		}
	}

	// Skills
	skillRefs := make([]api.ResourceRefRequest, 0, len(projection.Skills))
	for _, skillRef := range projection.Skills {
		skillRefs = append(skillRefs, api.ResourceRefRequest{
			Owner:   skillRef.Owner,
			Name:    skillRef.Name,
			Version: skillRef.Version,
		})
	}
	req.Skills = skillRefs

	// Children (for composite teams)
	children := make([]api.ChildRefRequest, 0, len(projection.Children))
	for _, child := range projection.Children {
		children = append(children, api.ChildRefRequest{
			Owner:        child.Owner,
			Name:         child.Name,
			ChildVersion: child.ChildVersion,
		})
	}
	req.Children = children

	return req, nil
}

// --- Modified / Tracked ---

func (s *Service) ModifiedTrackedResources(base string) ([]TrackedResource, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}

	var modified []TrackedResource
	for _, resource := range manifest.TrackedResources {
		sum, err := s.fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
		if err != nil {
			return nil, err
		}
		if sum != resource.BaseHash {
			modified = append(modified, resource)
		}
	}
	return modified, nil
}

func (s *Service) trackedStatuses(base string, manifest *Manifest) ([]TrackedStatus, int, error) {
	statuses := make([]TrackedStatus, 0, len(manifest.TrackedResources))
	modifiedCount := 0
	for _, resource := range manifest.TrackedResources {
		sum, err := s.fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
		if err != nil {
			return nil, 0, err
		}
		local := "clean"
		if sum != resource.BaseHash {
			local = "modified"
			modifiedCount++
		}
		statuses = append(statuses, TrackedStatus{
			Kind:  resource.Kind,
			Owner: resource.Owner,
			Name:  resource.Name,
			Path:  resource.LocalPath,
			Local: local,
		})
	}
	slices.SortFunc(statuses, func(a, b TrackedStatus) int {
		return strings.Compare(a.Path, b.Path)
	})
	return statuses, modifiedCount, nil
}

func (s *Service) runSummary(base string) (RunStatusSummary, error) {
	dir := filepath.Join(base, ".clier")
	entries, err := s.fs.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return RunStatusSummary{}, nil
		}
		return RunStatusSummary{}, fmt.Errorf("read runtime dir: %w", err)
	}
	var summary RunStatusSummary
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || name == ManifestFile || name == TeamProjectionFile {
			continue
		}
		plan, err := apprun.LoadPlanFromPath(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		summary.Total++
		if plan.Status == apprun.StatusRunning {
			summary.Running++
		} else {
			summary.Stopped++
		}
	}
	return summary, nil
}

// serverInstructionContent reads the local instruction file and strips the
// agent-specific prelude before sending to the server.
func (s *Service) serverInstructionContent(base string, _ *Manifest, resource TrackedResource) (string, error) {
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
	}
	content := string(data)

	// Determine the agent name from the local path (first path component).
	agentName := filepath.ToSlash(filepath.Clean(resource.LocalPath))
	if idx := strings.Index(agentName, "/"); idx >= 0 {
		agentName = agentName[:idx]
	}

	// Try to find the agent type by looking up the agent's team projection.
	agentType := resource.AgentType
	if agentType == "" {
		agentType = "claude"
	}
	return StripInstructionPrelude(agentType, agentName, content), nil
}

// --- Helpers ---

// buildResourceMap indexes resolved resources by owner/name for O(1) lookup.
func buildResourceMap(resources []api.ResolvedResource) map[string]*api.ResolvedResource {
	m := make(map[string]*api.ResolvedResource, len(resources))
	for i := range resources {
		r := &resources[i]
		m[teamKey(r.OwnerName, r.Name)] = r
	}
	return m
}

// snapshotRef represents a ref entry embedded in a resolve snapshot.
type snapshotRef struct {
	RelType       string `json:"rel_type"`
	TargetName    string `json:"target_name"`
	TargetOwner   string `json:"target_owner"`
	TargetVersion int    `json:"target_version"`
	AgentType     string `json:"agent_type,omitempty"`
	Command       string `json:"command,omitempty"`
}

// snapshotWithRefs is the common wrapper for snapshots that contain refs.
type snapshotWithRefs struct {
	Refs []snapshotRef `json:"refs"`
}

// teamProjectionFromResolved builds a TeamProjection from a ResolvedResource
// (used during Clone and Pull with ResolveTeam responses).
func teamProjectionFromResolved(r *api.ResolvedResource) *TeamProjection {
	projection := &TeamProjection{
		Name:     r.Name,
		Skills:   make([]ResourceRefProjection, 0),
		Children: make([]ChildProjection, 0),
	}

	// Decode spec fields from snapshot.
	if spec, err := decodeSnapshot[api.TeamSpec](r.Snapshot); err == nil {
		projection.AgentType = spec.AgentType
		projection.Command = spec.Command
		projection.GitRepoURL = spec.GitRepoURL
		for _, child := range spec.Children {
			projection.Children = append(projection.Children, ChildProjection{
				Owner:        child.Owner,
				Name:         child.Name,
				ChildVersion: child.Version,
			})
		}
	}

	// Decode refs from snapshot for instruction, settings, skill references.
	var refs snapshotWithRefs
	if err := decodeSnapshotInto(r.Snapshot, &refs); err == nil {
		for _, ref := range refs.Refs {
			rp := ResourceRefProjection{
				Owner:   ref.TargetOwner,
				Name:    ref.TargetName,
				Version: ref.TargetVersion,
			}
			switch ref.RelType {
			case string(api.KindInstruction):
				projection.InstructionRef = &rp
			case string(api.KindClaudeSettings), string(api.KindCodexSettings):
				projection.SettingsRef = &rp
			case string(api.KindSkill):
				projection.Skills = append(projection.Skills, rp)
			}
		}
	}

	return projection
}

// appendAgentTrackedResources adds tracked resources and generated files for
// a single agent (leaf team) to the tracked/generated slices.
func appendAgentTrackedResources(tracked *[]TrackedResource, generated *[]string, projection *TeamProjection, localBase string, profile domain.AgentProfile) {
	agentType := projection.AgentType

	if profile.LocalSettingsFile != "" {
		*generated = append(*generated, filepath.ToSlash(filepath.Join(localBase, profile.SettingsDir, profile.LocalSettingsFile)))
	}

	if projection.InstructionRef != nil {
		*tracked = append(*tracked, TrackedResource{
			Kind:          profile.InstructionKind,
			AgentType:     agentType,
			Owner:         projection.InstructionRef.Owner,
			Name:          projection.InstructionRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(localBase, profile.InstructionFile)),
			RemoteVersion: intPtr(projection.InstructionRef.Version),
			Editable:      true,
		})
	} else {
		*generated = append(*generated, filepath.ToSlash(filepath.Join(localBase, profile.InstructionFile)))
	}

	if projection.SettingsRef != nil {
		*tracked = append(*tracked, TrackedResource{
			Kind:          profile.SettingsKind,
			AgentType:     agentType,
			Owner:         projection.SettingsRef.Owner,
			Name:          projection.SettingsRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(localBase, profile.SettingsDir, profile.SettingsFile)),
			RemoteVersion: intPtr(projection.SettingsRef.Version),
			Editable:      true,
		})
	}

	for _, skillRef := range projection.Skills {
		*tracked = append(*tracked, TrackedResource{
			Kind:          string(api.KindSkill),
			AgentType:     agentType,
			Owner:         skillRef.Owner,
			Name:          skillRef.Name,
			LocalPath:     filepath.ToSlash(filepath.Join(localBase, profile.SettingsDir, profile.SkillsDir, skillRef.Name, "SKILL.md")),
			RemoteVersion: intPtr(skillRef.Version),
			Editable:      true,
		})
	}
}

// buildPeerRelations builds TeamRelations where all other keys are peers (workers).
// In the new flat model, children are peers — no leader/worker hierarchy.
func buildPeerRelations(selfKey string, allKeys []string) domain.TeamRelations {
	workers := make([]string, 0, len(allKeys)-1)
	for _, key := range allKeys {
		if key != selfKey {
			workers = append(workers, key)
		}
	}
	return domain.TeamRelations{
		Leaders: []string{},
		Workers: workers,
	}
}

// teamKey returns the unique identifier for a team (owner/name).
func teamKey(owner, name string) string {
	return owner + "/" + name
}

func (s *Service) populateBaseHashes(base string, tracked []TrackedResource) error {
	for i := range tracked {
		sum, err := s.fileHash(filepath.Join(base, filepath.FromSlash(tracked[i].LocalPath)))
		if err != nil {
			return err
		}
		tracked[i].BaseHash = sum
	}
	return nil
}

func intPtr(v int) *int {
	return &v
}

func normalizePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, filepath.ToSlash(filepath.Clean(path)))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func (s *Service) fileHash(path string) (string, error) {
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func versionsMatch(expected *int, actual int) bool {
	if expected == nil {
		return false
	}
	return *expected == actual
}
