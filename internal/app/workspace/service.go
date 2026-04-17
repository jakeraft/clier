package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

type teamEntry struct {
	id         string
	owner      string
	name       string
	version    int
	projection *TeamProjection
}

type agentEntry struct {
	teamEntry
	localBase string
}

func (s *Service) Clone(base, owner, name string) (*Manifest, error) {
	resolved, err := s.client.ResolveTeam(owner, name)
	if err != nil {
		return nil, fmt.Errorf("resolve team %s/%s: %w", owner, name, err)
	}

	resourceMap := buildResourceMap(resolved.Resources)
	root := &resolved.Root
	manifest, err := s.materializeResolvedTeam(base, root, resourceMap, nil)
	if err != nil {
		return nil, err
	}
	manifest.ClonedAt = time.Now().UTC()
	if err := SaveManifest(s.fs, base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *Service) materializeResolvedTeam(base string, root *api.ResolvedResource, resourceMap map[string]*api.ResolvedResource, previous *Manifest) (*Manifest, error) {
	rootProjection := teamProjectionFromResolved(root)
	teams, agents, err := s.collectResolvedEntries(root.OwnerName, root.Version, rootProjection, resourceMap)
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, fmt.Errorf("team %s/%s has no runnable agents (unknown agent type %q)", root.OwnerName, rootProjection.Name, rootProjection.AgentType)
	}
	localDirs := assignLocalDirs(agents, previous)

	tracked := make([]TrackedResource, 0, len(teams))
	storedTeams := make([]StoredTeamState, 0, len(teams))
	generated := []string{}

	for _, team := range teams {
		localDir := localDirs[team.id]
		storedTeams = append(storedTeams, StoredTeamState{
			Owner:      team.owner,
			Name:       team.name,
			Version:    team.version,
			LocalDir:   localDir,
			Projection: *team.projection,
		})
		tracked = append(tracked, TrackedResource{
			Kind:          string(api.KindTeam),
			AgentType:     team.projection.AgentType,
			Owner:         team.owner,
			Name:          team.name,
			LocalPath:     teamTrackedPath(team.owner, team.name),
			RemoteVersion: intPtr(team.version),
			Editable:      true,
		})
	}

	allKeys := make([]string, 0, len(agents))
	agentsByKey := make(map[string]ProtocolAgent, len(agents))
	for _, agent := range agents {
		allKeys = append(allKeys, agent.id)
		agentsByKey[agent.id] = ProtocolAgent{ID: agent.id, Owner: agent.owner, Name: agent.name}
	}

	writer := NewWriter(s.fs, s.git, resourceMap)
	for _, agent := range agents {
		agent.localBase = localDirs[agent.id]
		agentBase := filepath.Join(base, filepath.FromSlash(agent.localBase))
		if err := writer.MaterializeAgent(agentBase, agent.projection, agent.id); err != nil {
			return nil, fmt.Errorf("materialize agent %s: %w", agent.id, err)
		}

		relations := buildPeerRelations(agent.id, allKeys)
		self := agentsByKey[agent.id]
		protocol := BuildAgentFacingTeamProtocol(rootProjection.Name, self, relations, agentsByKey)
		protocolPath := filepath.Join(agentBase, ".clier", TeamProtocolFileName(agent.id))
		if err := s.fs.EnsureFile(protocolPath, []byte(protocol)); err != nil {
			return nil, fmt.Errorf("write protocol for %s: %w", agent.id, err)
		}

		profile, _ := domain.ProfileFor(agent.projection.AgentType)
		generated = append(generated,
			filepath.ToSlash(filepath.Join(agent.localBase, ".clier", "work-log-protocol.md")),
			filepath.ToSlash(filepath.Join(agent.localBase, ".clier", TeamProtocolFileName(agent.id))),
		)
		appendAgentTrackedResources(&tracked, &generated, agent.projection, agent.localBase, profile)
	}

	manifest := &Manifest{
		Kind:             string(api.KindTeam),
		Owner:            root.OwnerName,
		Name:             rootProjection.Name,
		Teams:            storedTeams,
		RootResource:     tracked[0],
		TrackedResources: tracked,
		GeneratedFiles:   normalizePaths(generated),
	}
	if err := s.populateBaseHashes(base, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *Service) collectResolvedEntries(rootOwner string, rootVersion int, rootProjection *TeamProjection, resourceMap map[string]*api.ResolvedResource) ([]teamEntry, []agentEntry, error) {
	var teams []teamEntry
	var agents []agentEntry

	var walk func(owner string, version int, projection *TeamProjection) error
	walk = func(owner string, version int, projection *TeamProjection) error {
		entry := teamEntry{
			id:         ResourceID(owner, projection.Name),
			owner:      owner,
			name:       projection.Name,
			version:    version,
			projection: projection,
		}
		teams = append(teams, entry)
		if _, err := domain.ProfileFor(projection.AgentType); err == nil {
			agents = append(agents, agentEntry{
				teamEntry: entry,
				localBase: AgentWorkspaceLocalPath(owner, projection.Name),
			})
		}
		for _, child := range projection.Children {
			childResource, ok := resourceMap[teamKey(child.Owner, child.Name)]
			if !ok {
				return fmt.Errorf("resolve child team %s/%s: not found in resolve response", child.Owner, child.Name)
			}
			if err := walk(child.Owner, childResource.Version, teamProjectionFromResolved(childResource)); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(rootOwner, rootVersion, rootProjection); err != nil {
		return nil, nil, err
	}
	return teams, agents, nil
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
	pulled, err := s.materializeResolvedTeam(base, &resolved.Root, resourceMap, manifest)
	if err != nil {
		return nil, err
	}
	pulled.ClonedAt = manifest.ClonedAt
	pulled.FirstRunAt = manifest.FirstRunAt
	if err := s.removeStaleManagedFiles(base, manifest, pulled); err != nil {
		return nil, err
	}
	return pulled, nil
}

// --- Status ---

func (s *Service) Status(base, runsDir string) (*Status, error) {
	manifest, err := LoadManifest(s.fs, base)
	if err != nil {
		return nil, err
	}
	tracked, modifiedCount, err := s.trackedStatuses(base, manifest)
	if err != nil {
		return nil, err
	}
	runs, err := s.runSummary(base, runsDir)
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
	trackedIndex := make(map[string]int, len(tracked))
	for i, tr := range tracked {
		trackedIndex[tr.Path] = i
	}
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
		idx, ok := trackedIndex[manifest.TrackedResources[i].LocalPath]
		if !ok {
			continue
		}
		pinned := *tr.RemoteVersion
		tracked[idx].PinnedVersion = &pinned
		tracked[idx].LatestVersion = &rv.latest
		if pinned < rv.latest {
			tracked[idx].Remote = "behind"
		} else {
			tracked[idx].Remote = "up-to-date"
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
	pushedPaths := map[string]bool{}

	for _, resource := range modified {
		if !resource.Editable || api.ResourceKind(resource.Kind) == api.KindTeam {
			continue
		}
		updated, err := s.pushTrackedResource(base, manifest, resource)
		if err != nil {
			return nil, err
		}
		pushed++
		pushedPaths[resource.LocalPath] = true
		if updated != nil {
			if resource.LocalPath == manifest.RootResource.LocalPath {
				targetName = updated.Metadata.Name
			}
			s.applyPushedResourceVersion(manifest, resource, updated.Metadata.LatestVersion)
			if err := s.recordPushedResourceState(base, manifest, resource); err != nil {
				return nil, err
			}
		}
	}

	for {
		progress := false
		for _, resource := range manifest.TrackedResources {
			if !resource.Editable || api.ResourceKind(resource.Kind) != api.KindTeam {
				continue
			}
			if pushedPaths[resource.LocalPath] {
				continue
			}
			modified, blocked, err := s.teamPushState(base, manifest, resource)
			if err != nil {
				return nil, err
			}
			if !modified || blocked {
				continue
			}
			updated, err := s.pushTrackedResource(base, manifest, resource)
			if err != nil {
				return nil, err
			}
			pushed++
			pushedPaths[resource.LocalPath] = true
			progress = true
			if updated != nil {
				if resource.LocalPath == manifest.RootResource.LocalPath {
					targetName = updated.Metadata.Name
				}
				s.applyPushedResourceVersion(manifest, resource, updated.Metadata.LatestVersion)
				if err := s.recordPushedResourceState(base, manifest, resource); err != nil {
					return nil, err
				}
			}
		}
		if !progress {
			break
		}
	}

	for _, resource := range manifest.TrackedResources {
		if !resource.Editable || api.ResourceKind(resource.Kind) != api.KindTeam {
			continue
		}
		modified, _, err := s.teamPushState(base, manifest, resource)
		if err != nil {
			return nil, err
		}
		if modified {
			return nil, fmt.Errorf("unable to push team %s/%s: unresolved local team changes remain", resource.Owner, resource.Name)
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

func (s *Service) pushTrackedResource(base string, manifest *Manifest, resource TrackedResource) (*api.ResourceResponse, error) {
	kind, body, err := s.preparePushBody(base, manifest, resource)
	if err != nil {
		return nil, err
	}
	return s.pushResource(resource, kind, body)
}

func (s *Service) teamPushState(base string, manifest *Manifest, resource TrackedResource) (modified bool, blocked bool, err error) {
	sum, err := s.trackedResourceHash(base, manifest, resource)
	if err != nil {
		return false, false, err
	}
	if sum == resource.BaseHash {
		return false, false, nil
	}

	team, ok := manifest.FindTeam(resource.Owner, resource.Name)
	if !ok {
		return false, false, fmt.Errorf("team state %s/%s not found", resource.Owner, resource.Name)
	}
	for _, child := range team.Projection.Children {
		childTracked, ok := manifest.FindTrackedResource(teamTrackedPath(child.Owner, child.Name))
		if !ok || !childTracked.Editable {
			continue
		}
		childSum, err := s.trackedResourceHash(base, manifest, *childTracked)
		if err != nil {
			return false, false, err
		}
		if childSum != childTracked.BaseHash {
			return true, true, nil
		}
	}
	return true, false, nil
}

func (s *Service) applyPushedResourceVersion(manifest *Manifest, resource TrackedResource, latest int) {
	for i := range manifest.TrackedResources {
		if manifest.TrackedResources[i].Kind != resource.Kind {
			continue
		}
		if manifest.TrackedResources[i].Owner != resource.Owner || manifest.TrackedResources[i].Name != resource.Name {
			continue
		}
		manifest.TrackedResources[i].RemoteVersion = intPtr(latest)
	}

	if resource.Kind == string(api.KindTeam) {
		if manifest.RootResource.Kind == resource.Kind && manifest.RootResource.Owner == resource.Owner && manifest.RootResource.Name == resource.Name {
			manifest.RootResource.RemoteVersion = intPtr(latest)
		}
		for i := range manifest.Teams {
			if manifest.Teams[i].Owner == resource.Owner && manifest.Teams[i].Name == resource.Name {
				manifest.Teams[i].Version = latest
			}
			for j := range manifest.Teams[i].Projection.Children {
				child := &manifest.Teams[i].Projection.Children[j]
				if child.Owner == resource.Owner && child.Name == resource.Name {
					child.ChildVersion = latest
				}
			}
		}
		return
	}

	for i := range manifest.Teams {
		projection := &manifest.Teams[i].Projection
		if resource.Kind == string(api.KindInstruction) && projection.InstructionRef != nil &&
			projection.InstructionRef.Owner == resource.Owner && projection.InstructionRef.Name == resource.Name {
			projection.InstructionRef.Version = latest
		}
		if (resource.Kind == string(api.KindClaudeSettings) || resource.Kind == string(api.KindCodexSettings)) &&
			projection.SettingsRef != nil &&
			projection.SettingsRef.Owner == resource.Owner && projection.SettingsRef.Name == resource.Name {
			projection.SettingsRef.Version = latest
		}
		if resource.Kind == string(api.KindSkill) {
			for j := range projection.Skills {
				skill := &projection.Skills[j]
				if skill.Owner == resource.Owner && skill.Name == resource.Name {
					skill.Version = latest
				}
			}
		}
	}
}

func (s *Service) recordPushedResourceState(base string, manifest *Manifest, resource TrackedResource) error {
	sum, err := s.trackedResourceHash(base, manifest, resource)
	if err != nil {
		return err
	}
	for i := range manifest.TrackedResources {
		if manifest.TrackedResources[i].Kind != resource.Kind {
			continue
		}
		if manifest.TrackedResources[i].Owner != resource.Owner || manifest.TrackedResources[i].Name != resource.Name {
			continue
		}
		manifest.TrackedResources[i].BaseHash = sum
	}
	if manifest.RootResource.Kind == resource.Kind && manifest.RootResource.Owner == resource.Owner && manifest.RootResource.Name == resource.Name {
		manifest.RootResource.BaseHash = sum
	}
	return nil
}

// preparePushBody builds the request body for a single tracked resource.
func (s *Service) preparePushBody(base string, manifest *Manifest, r TrackedResource) (api.ResourceKind, any, error) {
	kind := api.ResourceKind(r.Kind)
	switch kind {
	case api.KindTeam:
		team, ok := manifest.FindTeam(r.Owner, r.Name)
		if !ok {
			return "", nil, fmt.Errorf("team state %s/%s not found", r.Owner, r.Name)
		}
		body, err := s.teamMutationFromProjection(&team.Projection)
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

// teamMutationFromProjection builds a TeamWriteRequest from a TeamProjection,
// resolving the team's own agent fields and its children.
func (s *Service) teamMutationFromProjection(projection *TeamProjection) (*api.TeamWriteRequest, error) {
	req := &api.TeamWriteRequest{
		Name:       projection.Name,
		Command:    projection.Command,
		GitRepoURL: projection.GitRepoURL,
	}

	// Instruction and settings refs require an agent profile to resolve the kind.
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

	// Children
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
		sum, err := s.trackedResourceHash(base, manifest, resource)
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
		sum, err := s.trackedResourceHash(base, manifest, resource)
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

// runSummary delegates to apprun.ListPlans (single scan policy) and
// counts plans whose WorkingCopyPath matches the given base.
func (s *Service) runSummary(base, runsDir string) (RunStatusSummary, error) {
	if runsDir == "" {
		return RunStatusSummary{}, nil
	}
	plans, err := apprun.ListPlans(runsDir)
	if err != nil {
		return RunStatusSummary{}, err
	}
	var summary RunStatusSummary
	for _, plan := range plans {
		if plan.WorkingCopyPath != base {
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
func (s *Service) serverInstructionContent(base string, manifest *Manifest, resource TrackedResource) (string, error) {
	data, err := s.fs.ReadFile(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
	if err != nil {
		return "", fmt.Errorf("read local resource %s: %w", resource.LocalPath, err)
	}
	content := string(data)

	// Try to find the agent type by looking up the agent's team projection.
	agentType := resource.AgentType
	if agentType == "" {
		agentType = "claude"
	}
	agent, ok := manifest.AgentForLocalPath(resource.LocalPath)
	if !ok {
		return "", fmt.Errorf("derive agent id from %s: no matching local dir in state", resource.LocalPath)
	}
	return StripInstructionPrelude(agentType, ResourceID(agent.Owner, agent.Name), content), nil
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
// a team's agent role to the tracked/generated slices.
func appendAgentTrackedResources(tracked *[]TrackedResource, generated *[]string, projection *TeamProjection, localBase string, profile domain.AgentProfile) {
	agentType := projection.AgentType

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
			LocalPath:     filepath.ToSlash(filepath.Join(localBase, profile.SettingsDir, profile.SkillsDir, skillRef.Owner, skillRef.Name, "SKILL.md")),
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
	return ResourceID(owner, name)
}

func (s *Service) removeStaleManagedFiles(base string, previous, next *Manifest) error {
	keep := make(map[string]struct{}, len(next.TrackedResources)+len(next.GeneratedFiles))
	for _, resource := range next.TrackedResources {
		keep[resource.LocalPath] = struct{}{}
	}
	for _, path := range next.GeneratedFiles {
		keep[path] = struct{}{}
	}

	for _, resource := range previous.TrackedResources {
		if _, ok := keep[resource.LocalPath]; ok {
			continue
		}
		if err := s.fs.RemoveAll(filepath.Join(base, filepath.FromSlash(resource.LocalPath))); err != nil {
			return fmt.Errorf("remove stale tracked file %s: %w", resource.LocalPath, err)
		}
	}
	for _, path := range previous.GeneratedFiles {
		if _, ok := keep[path]; ok {
			continue
		}
		if err := s.fs.RemoveAll(filepath.Join(base, filepath.FromSlash(path))); err != nil {
			return fmt.Errorf("remove stale generated file %s: %w", path, err)
		}
	}
	return nil
}

func (s *Service) populateBaseHashes(base string, manifest *Manifest) error {
	for i := range manifest.TrackedResources {
		sum, err := s.trackedResourceHash(base, manifest, manifest.TrackedResources[i])
		if err != nil {
			return err
		}
		manifest.TrackedResources[i].BaseHash = sum
	}
	if len(manifest.TrackedResources) > 0 {
		manifest.RootResource = manifest.TrackedResources[0]
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

func (s *Service) trackedResourceHash(base string, manifest *Manifest, resource TrackedResource) (string, error) {
	if api.ResourceKind(resource.Kind) == api.KindTeam {
		if manifest == nil {
			return "", errors.New("team hash requires state")
		}
		team, ok := manifest.FindTeam(resource.Owner, resource.Name)
		if !ok {
			return "", fmt.Errorf("team state %s/%s not found", resource.Owner, resource.Name)
		}
		data, err := json.Marshal(team.Projection)
		if err != nil {
			return "", fmt.Errorf("marshal team projection %s/%s: %w", resource.Owner, resource.Name, err)
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:]), nil
	}
	return s.fileHash(filepath.Join(base, filepath.FromSlash(resource.LocalPath)))
}

func teamTrackedPath(owner, name string) string {
	return ".clier/" + ResourceID(owner, name) + ".team"
}

func assignLocalDirs(agents []agentEntry, previous *Manifest) map[string]string {
	assigned := make(map[string]string, len(agents))
	taken := map[string]bool{}

	if previous != nil {
		for _, team := range previous.Teams {
			if team.LocalDir != "" {
				taken[team.LocalDir] = true
			}
		}
	}

	for _, agent := range agents {
		if previous != nil {
			if prior, ok := previous.FindTeam(agent.owner, agent.name); ok && prior.LocalDir != "" {
				assigned[agent.id] = prior.LocalDir
				continue
			}
		}
		base := AgentWorkspaceLocalPath(agent.owner, agent.name)
		candidate := base
		for i := 2; taken[candidate]; i++ {
			candidate = fmt.Sprintf("%s__%d", base, i)
		}
		taken[candidate] = true
		assigned[agent.id] = candidate
	}

	return assigned
}

func versionsMatch(expected *int, actual int) bool {
	if expected == nil {
		return false
	}
	return *expected == actual
}
